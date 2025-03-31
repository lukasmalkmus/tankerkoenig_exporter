package exporter

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/alexruf/tankerkoenig-go"
	"github.com/mmcloughlin/geohash"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"maps"

	"github.com/lukasmalkmus/tankerkoenig_exporter/internal/client"
)

const namespace = "tk"

var caser = cases.Title(language.German)

// Exporter collects stats from the Tankerkoenig API and exports them using the
// prometheus client library.
type Exporter struct {
	logger *log.Logger

	mutex    sync.RWMutex
	client   *tankerkoenig.Client
	stations map[string]tankerkoenig.Station

	// Basic exporter metrics.
	up, scrapeDuration          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter

	// Tankerkoenig metrics.
	priceDesc   *prometheus.Desc
	openDesc    *prometheus.Desc
	detailsDesc *prometheus.Desc
}

// NewForStations returns a new, initialized Tankerkoenig API exporter for the
// given stations.
func NewForStations(logger *log.Logger, apiClient *client.Client, apiStations []string) (*Exporter, error) {
	e := newExporter(logger, apiClient)

	e.stations = make(map[string]tankerkoenig.Station, len(apiStations))

	// Retrieve initial station details to validate integrity of user provided
	// station IDs.
	for _, id := range apiStations {
		station, _, err := apiClient.Station.Detail(id)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve station details for station %s: %w", id, err)
		} else if station.Id == "" {
			return nil, fmt.Errorf("station %q was not found", id)
		}
		e.stations[id] = station
	}

	return e, nil
}

// NewForLocation returns a new, initialized Tankerkoenig API exporter for the
// stations that are in the given radius around the given location.
func NewForLocation(logger *log.Logger, apiClient *client.Client, location string, radius int) (*Exporter, error) {
	e := newExporter(logger, apiClient)

	lat, lng := geohash.Decode(location)

	stations, _, err := apiClient.Station.List(lat, lng, radius)
	if err != nil {
		return nil, fmt.Errorf("could not list stations: %w", err)
	}

	e.stations = make(map[string]tankerkoenig.Station, len(stations))

	for _, station := range stations {
		e.stations[station.Id] = station
	}

	return e, nil
}

// Describe all the metrics collected by the Tankerkoenig exporter.
// Implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.scrapeDuration.Describe(ch)
	e.failedScrapes.Describe(ch)
	e.totalScrapes.Describe(ch)
	ch <- e.priceDesc
	ch <- e.openDesc
	ch <- e.detailsDesc
}

// Collect the stats from the Tankerkoenig API.
// Implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Protect metrics from concurrent collects.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Scrape metrics from Tankerkoenig API.
	if err := e.scrape(ch); err != nil {
		e.logger.Printf("error: cannot scrape tankerkoenig api: %v", err)
	}

	// Collect metrics.
	e.up.Collect(ch)
	e.scrapeDuration.Collect(ch)
	e.failedScrapes.Collect(ch)
	e.totalScrapes.Collect(ch)
}

// scrape performs the API call and meassures its duration.
func (e *Exporter) scrape(ch chan<- prometheus.Metric) error {
	// Meassure scrape duration.
	defer func(begun time.Time) {
		e.scrapeDuration.Set(time.Since(begun).Seconds())
	}(time.Now())

	e.totalScrapes.Inc()

	// Extract station IDs for price request.
	ids := make([]string, 0, len(e.stations))
	for id := range e.stations {
		ids = append(ids, id)
	}

	// Retrieve prices for specified stations. Since the API will only allow for
	// ten stations to be queried with one request, we work them of in batches
	// of ten.
	const batchSize = 10
	var (
		prices   = make(map[string]tankerkoenig.Price, len(ids))
		pricesMu sync.Mutex
		errGroup errgroup.Group
	)
	for i := 0; i < len(ids); i += batchSize {
		j := min(i+batchSize, len(ids))

		errGroup.Go(func(batch []string) func() error {
			return func() error {
				batchPrices, _, err := e.client.Prices.Get(batch...)
				if err != nil {
					return err
				}

				pricesMu.Lock()
				maps.Copy(prices, batchPrices)
				pricesMu.Unlock()

				return nil
			}
		}(ids[i:j]))
	}

	if err := errGroup.Wait(); err != nil {
		e.up.Set(0)
		e.failedScrapes.Inc()
		return err
	}

	// Set metric values.
	for id, price := range prices {
		station := e.stations[id]

		// Station metadata. We do some string manipulation on the address and
		// city to make it look nicer as the come in all uppercase.
		city := strings.TrimSpace(caser.String(station.Place))
		street := strings.TrimSpace(caser.String(station.Street))
		no := strings.TrimSpace(station.HouseNumber)
		address := fmt.Sprintf("%s %s", street, no)
		ch <- prometheus.MustNewConstMetric(e.detailsDesc, prometheus.GaugeValue, 1, id,
			station.Name,
			address,
			city,
			geohash.Encode(station.Lat, station.Lng),
			station.Brand,
		)

		// Station status.
		if stat := price.Status; stat == "no prices" {
			e.logger.Printf("warning: station %q (%s) has no prices, skipping...", id, station.Name)
			continue
		} else if stat == "open" {
			ch <- prometheus.MustNewConstMetric(e.openDesc, prometheus.GaugeValue, 1, id)
		} else {
			ch <- prometheus.MustNewConstMetric(e.openDesc, prometheus.GaugeValue, 0, id)
		}

		// Station prices.
		if v, ok := price.Diesel.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, "diesel")
		}
		if v, ok := price.E5.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, "e5")
		}
		if v, ok := price.E10.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, "e10")
		}
	}

	// Scrape was successful.
	e.up.Set(1)

	return nil
}

func newExporter(logger *log.Logger, apiClient *client.Client) *Exporter {
	return &Exporter{
		logger: logger,

		client: apiClient,

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of the Tankerkoenig API successful?",
		}),
		scrapeDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "Duration of the scrape of metrics from the Tankerkoenig API.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrapes_total",
			Help:      "Total Tankerkoenig API scrapes.",
		}),
		failedScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_failures_total",
			Help:      "Total amount of scrape failures.",
		}),
		priceDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "station", "price_euro"),
			"Gas prices in EURO (€).",
			[]string{"id", "product"},
			nil,
		),
		openDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "station", "open"),
			"Status of the station. 1 for OPEN, 0 for CLOSED.",
			[]string{"id"},
			nil,
		),
		detailsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "station", "details"),
			"Associated details of a station. Always 1.",
			[]string{"id", "name", "address", "city", "geohash", "brand"},
			nil,
		),
	}
}
