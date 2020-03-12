// Copyright 2019 Lukas Malkmus
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexruf/tankerkoenig-go"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mmcloughlin/geohash"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

// namespace for all metrics of this exporter.
const namespace = "tk"

var (
	apiStations   = kingpin.Flag("api.stations", "ID of a station. Flag can be reused multiple times.").Short('s').Strings()
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9386").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
)

// landingPage contains the HTML served at '/'.
var landingPage = `<html>
	<head>
		<title>Tankerkoenig Exporter</title>
	</head>
	<body>
		<h1>Tankerkoenig Exporter</h1>
		<p>
		<a href=` + *metricsPath + `>Metrics</a>
		</p>
	</body>
</html>`

// Exporter collects stats from the Tankerkoenig API and exports them using the
// prometheus client library.
type Exporter struct {
	logger log.Logger

	mutex    sync.RWMutex
	client   *tankerkoenig.Client
	stations map[string]tankerkoenig.Station

	// Basic exporter metrics.
	up, scrapeDuration          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter

	// Tankerkoenig metrics.
	priceDesc *prometheus.Desc
	openDesc  *prometheus.Desc
}

// NewExporter returns a new, initialized Tankerkoenig Exporter.
func NewExporter(logger log.Logger, apiKey string, apiStations []string) (*Exporter, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 15,
	}

	e := &Exporter{
		logger: logger,

		client:   tankerkoenig.NewClient(apiKey, httpClient),
		stations: make(map[string]tankerkoenig.Station, len(apiStations)),

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
			[]string{"station_id", "station_name", "address", "city", "geohash", "station_brand", "product"},
			nil,
		),
		openDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "station", "open"),
			"Status of the station. 1 for OPEN, 0 for CLOSED.",
			[]string{"station_id", "station_name", "address", "city", "geohash", "station_brand"},
			nil,
		),
	}

	// Retrieve initial station details to validate integrity of user provided
	// station IDs.
	for _, v := range apiStations {
		station, _, err := e.client.Station.Detail(v)
		if err != nil {
			return nil, fmt.Errorf("couldn't retrieve station details for station %s: %w", v, err)
		} else if station.Id == "" {
			return nil, fmt.Errorf("station %q was not found", v)
		}
		e.stations[v] = station
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
}

// Collect the stats from the Tankerkoenig API.
// Implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Protect metrics from concurrent collects.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Scrape metrics from Tankerkoenig API.
	if err := e.scrape(ch); err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape Tankerkoenig API", "err", err)
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
	ids := make([]string, 0)
	for id := range e.stations {
		ids = append(ids, id)
	}

	// Retrieve prices for specified stations.
	prices, _, err := e.client.Prices.Get(ids)
	if err != nil {
		e.up.Set(0)
		e.failedScrapes.Inc()
		return err
	}

	// Set metric values.
	for id, p := range prices {
		s := e.stations[id]
		name := fmt.Sprintf("%s (%s)", s.Name, s.Place)

		geohash := geohash.Encode(s.Lat, s.Lng)
		// string manipulation because of capslock names
		address := strings.TrimSpace(fmt.Sprintf("%v %v", strings.TrimSpace(strings.Title(strings.ToLower(s.Street))), strings.TrimSpace(s.HouseNumber)))
		city := strings.TrimSpace(strings.Title(strings.ToLower(s.Place)))

		// Station status.
		if stat := p.Status; stat == "no prices" {
			continue
		} else if stat == "open" {
			ch <- prometheus.MustNewConstMetric(e.openDesc, prometheus.GaugeValue, 1.0, id, name, address, city, geohash, s.Brand)
		} else {
			ch <- prometheus.MustNewConstMetric(e.openDesc, prometheus.GaugeValue, 0.0, id, name, address, city, geohash, s.Brand)
		}

		// Station prices.
		if v, ok := p.Diesel.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, name, address, city, geohash, s.Brand, "diesel")
		}
		if v, ok := p.E5.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, name, address, city, geohash, s.Brand, "e5")
		}
		if v, ok := p.E10.(float64); ok {
			ch <- prometheus.MustNewConstMetric(e.priceDesc, prometheus.GaugeValue, v, id, name, address, city, geohash, s.Brand, "e10")
		}
	}

	// Scrape was successful.
	e.up.Set(1)

	return nil
}

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	// Print build context and version.
	level.Info(logger).Log("msg", "Starting tankerkoenig_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	// Check if API key is present
	apiKey := os.Getenv("TANKERKOENIG_API_KEY")
	if apiKey == "" {
		level.Error(logger).Log("msg", "No API key present. Please set TANKERKOENIG_API_KEY!")
		os.Exit(1)
	}

	// Create a new Tankerkoenig exporter. Exit if an error is returned.
	exporter, err := NewExporter(logger, apiKey, *apiStations)
	if err != nil {
		level.Error(logger).Log("msg", "Can't create Tankerkoenig exporter", "err", err)
		os.Exit(1)
	}

	// Register Tankerkoenig and the collector for version information.
	// Unregister Go and Process collector which are registered by default.
	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter)
	reg.MustRegister(version.NewCollector("tankerkoenig_exporter"))

	stdlibLogger := stdlog.New(log.NewStdlibAdapter(logger), "", 0)

	// Setup router and handlers.
	mux := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      stdlibLogger,
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	mux.Handle(*metricsPath, metricsHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(landingPage))
	})

	// Setup webserver.
	srv := &http.Server{
		Addr:         *listenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorLog:     stdlibLogger,
	}

	// Listen for termination signals.
	term := make(chan os.Signal, 1)
	defer close(term)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(term)

	// Run webserver in a separate go-routine.
	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	webErr := make(chan error)
	defer close(webErr)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			webErr <- err
		}
	}()

	// Wait for a termination signal and shut down gracefully, but wait no
	// longer than 5 seconds before halting.
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	select {
	case <-term:
		level.Warn(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			level.Error(logger).Log("msg", "Error shutting down HTTP server", "err", err)
		}
	case err := <-webErr:
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
	}
}
