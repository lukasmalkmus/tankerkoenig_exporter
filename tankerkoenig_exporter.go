package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexruf/tankerkoenig-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// namespace for all metrics of this exporter.
	namespace = "tk"
)

var (
	apiKey           = kingpin.Flag("api.key", "Personal API key used to authenticate against the tankerkoenig API").String()
	apiStations      = kingpin.Flag("api.stations", "Comma seperated list of stations").Short('s').Strings()
	webListenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9386").String()
	webMetricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
)

// landingPage contains the HTML served at '/'.
var landingPage = `<html>
	<head>
		<title>Tankerkoenig Exporter</title>
	</head>
	<body>
		<h1>Tankerkoenig Exporter</h1>
		<p>
		<a href=` + *webMetricsPath + `>Metrics</a>
		</p>
	</body>
</html>`

// Exporter collects stats from the Tankerkoenig API and exports them using the
// prometheus client library.
type Exporter struct {
	mutex    sync.RWMutex
	client   *tankerkoenig.Client
	stations map[string]tankerkoenig.Station

	// Basic exporter metrics.
	up, scrapeDuration          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter

	// Tankerkoenig metrics.
	price *prometheus.GaugeVec
	open  *prometheus.GaugeVec
}

// New returns a new, initialized Tankerkoenig Exporter.
func New(apiKey string, apiStations []string) (*Exporter, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			TLSHandshakeTimeout:   3 * time.Second,
		},
	}

	e := &Exporter{
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
		price: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "station",
			Name:      "price_euro",
			Help:      "Gas prices in EURO (€).",
		}, []string{"station_id", "station_name", "product"}),
		open: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "station",
			Name:      "open",
			Help:      "Status of the station. 1 for OPEN, 0 for CLOSED.",
		}, []string{"station_id", "station_name"}),
	}

	// Retrieve initial station details to validate integrity of user provided
	// station IDs.
	for _, v := range apiStations {
		station, _, err := e.client.Station.Detail(v)
		if err != nil {
			return nil, fmt.Errorf("Couldn't retrieve station details for station %s: %s", v, err)
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
	e.price.Describe(ch)
	e.open.Describe(ch)
}

// Collect the stats from the configured ArmA 3 server and deliver them as
// Prometheus metrics.
// Implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Protect metrics from concurrent collects.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Reset metrics.
	e.reset()

	// Scrape metrics from Tankerkoenig API.
	if err := e.scrape(); err != nil {
		log.Error(err)
	}

	// Collect metrics.
	e.up.Collect(ch)
	e.scrapeDuration.Collect(ch)
	e.failedScrapes.Collect(ch)
	e.totalScrapes.Collect(ch)
	e.price.Collect(ch)
	e.open.Collect(ch)
}

// reset resets the vector metrics.
func (e *Exporter) reset() {
	e.open.Reset()
	e.price.Reset()
}

// scrape performs the API call and meassures its duration.
func (e *Exporter) scrape() error {
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

		// Station status.
		if stat := p.Status; stat == "no prices" {
			continue
		} else if stat == "open" {
			e.open.WithLabelValues(id, name).Set(1.0)
		} else {
			e.open.WithLabelValues(id, name).Set(0.0)
		}

		// Station prices.
		if f, ok := p.Diesel.(float64); ok {
			e.price.WithLabelValues(id, name, "diesel").Set(f)
		}
		if f, ok := p.E5.(float64); ok {
			e.price.WithLabelValues(id, name, "e5").Set(f)
		}
		if f, ok := p.E10.(float64); ok {
			e.price.WithLabelValues(id, name, "e10").Set(f)
		}
	}

	// Scrape was successful.
	e.up.Set(1)

	return nil
}

func main() {
	os.Exit(Main())
}

// Main manages the complete application lifecycle, from startup to shutdown.
func Main() int {
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("tankerkoenig_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Print build context and version.
	log.Info("Starting tankerkoenig_exporter", version.Info())
	log.Info("Build context", version.BuildContext())

	// Create a new Tankerkoenig exporter. Exit if an error is returned.
	exporter, err := New(*apiKey, *apiStations)
	if err != nil {
		log.Error(err)
		return 0
	}

	// Register Tankerkoenig and the collector for version information.
	// Unregister Go and Process collector which are registered by default.
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("tk_exporter"))
	prometheus.Unregister(prometheus.NewGoCollector())
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))

	// Setup router and handlers.
	router := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{ErrorLog: log.NewErrorLogger()})
	// TODO: InstrumentHandler is depracted. Additional tools will be available
	// soon in the promhttp package.
	//router.Handle(*webMetricsPath, prometheus.InstrumentHandler("prometheus", metricsHandler))
	router.Handle(*webMetricsPath, metricsHandler)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(landingPage))
	})

	// Setup webserver.
	srv := &http.Server{
		Addr:           *webListenAddress,
		Handler:        router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       log.NewErrorLogger(),
	}

	// Listen for termination signals.
	term := make(chan os.Signal, 1)
	webErr := make(chan error)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	// Run webserver in a separate goroutine.
	log.Infoln("Listening on", *webListenAddress)
	go func() { webErr <- srv.ListenAndServe() }()

	// Wait for a termination signal and shut down gracefully, but wait no
	// longer than 5 seconds before halting.
	select {
	case <-term:
		log.Warn("Received SIGTERM, exiting gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Warnln("Error shutting down http server:", err)
		}
	case err := <-webErr:
		log.Errorln("Error starting http server, exiting gracefully:", err)
	}

	log.Info("See you next time!")

	return 0
}
