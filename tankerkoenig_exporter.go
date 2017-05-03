package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexruf/tankerkoenig-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

const (
	// namespace for all metrics of this exporter.
	namespace = "tk"
)

var (
	apiKey           = flag.String("api.key", "", "Personal API key used to authenticate against the tankerkoenig API.")
	apiStations      = flag.String("api.stations", "", "IDs of stations to scrape prices from.")
	webListenAddress = flag.String("web.listen-address", ":9243", "Address on which to expose metrics and web interface.")
	webMetricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	showVersion      = flag.Bool("version", false, "Print version information.")
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

var httpClient = &http.Client{
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

// Exporter collects stats from the Tankerkoenig API and exports them using the
// prometheus client library.
type Exporter struct {
	mutex           sync.RWMutex
	client          *tankerkoenig.Client
	requestInterval *time.Ticker
	stationsIDs     []string

	quitCh chan struct{}
	doneCh chan struct{}

	// Basic exporter metrics.
	up, scrapeDuration          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter

	// Tankerkoenig metrics.
	price *prometheus.GaugeVec
}

// New returns a new, initialized Tankerkoenig Exporter.
func New(apiKey string, stations []string) (*Exporter, error) {
	e := &Exporter{
		client:          tankerkoenig.NewClient(apiKey, httpClient),
		requestInterval: time.NewTicker(5 * time.Minute),
		stationsIDs:     stations,

		quitCh: make(chan struct{}),
		doneCh: make(chan struct{}),

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
		}, []string{"station_id", "status", "product"}),
	}

	// Initial scrape.
	e.scrape()

	// Background scraper.
	go func() {
		for {
			select {
			case <-e.requestInterval.C:
				// Reset and scrape metrics. Prevent access by locking RWMutex.
				e.mutex.Lock()
				defer e.mutex.Unlock()
				e.price.Reset()
				e.scrape()
			case <-e.quitCh:
				close(e.doneCh)
				break
			}
		}
	}()

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
}

// Collect the stats from the configured ArmA 3 server and deliver them as
// Prometheus metrics.
// Implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// Protect metrics from concurrent collects.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Collect metrics.
	e.up.Collect(ch)
	e.scrapeDuration.Collect(ch)
	e.failedScrapes.Collect(ch)
	e.totalScrapes.Collect(ch)
	e.price.Collect(ch)
}

// Close the Exporter gracefully. This will shut down the background scraper.
func (e *Exporter) Close(ctx context.Context) {
	close(e.quitCh)
	select {
	case <-e.doneCh:
	case <-ctx.Done():
	}
}

// scrape is where the magic happens. FIXME: Better description.
func (e *Exporter) scrape() {
	// Meassure scrape duration.
	defer func(begun time.Time) {
		e.scrapeDuration.Set(time.Since(begun).Seconds())
	}(time.Now())
	e.totalScrapes.Inc()

	// Retrieve prices.
	prices, _, err := e.client.Prices.Get(e.stationsIDs)
	if err != nil {
		e.up.Set(0)
		e.failedScrapes.Inc()
		log.Errorln(err)
		return
	}

	// Set metric values.
	for id, p := range prices {
		if f, ok := p.Diesel.(float64); ok {
			e.price.WithLabelValues(id, "diesel", p.Status).Set(f)
		}
		if f, ok := p.E5.(float64); ok {
			e.price.WithLabelValues(id, "e5", p.Status).Set(f)
		}
		if f, ok := p.E10.(float64); ok {
			e.price.WithLabelValues(id, "e10", p.Status).Set(f)
		}
	}

	// Scrape was successful.
	e.up.Set(1)
}

func main() {
	flag.Parse()

	// If the version Flag is set, print detailed version information and exit.
	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("tankerkoenig_exporter"))
		os.Exit(0)
	}

	// Print build context and version.
	log.Infoln("Starting tankerkoenig_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	// Create a new Tankerkoenig exporter. Exit if an error is returned.
	exporter, err := New(*apiKey, strings.Split(*apiStations, ","))
	if err != nil {
		log.Fatalln(err)
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
	term := make(chan os.Signal)
	webErr := make(chan error)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	// Run webserver in a separate go-routine.
	log.Infoln("Listening on", *webListenAddress)
	go func() {
		webErr <- srv.ListenAndServe()
	}()

	// Wait for a termination signal and shut down gracefully, but wait no
	// longer than 5 seconds before halting.
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	select {
	case <-term:
		log.Warn("Received SIGTERM, exiting gracefully...")
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
		exporter.Close(ctx)
	case err := <-webErr:
		log.Errorln("Error starting web server, exiting gracefully:", err)
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exporter.Close(ctx)
	}

	// Did the context throw an error?
	if err := ctx.Err(); err != nil {
		log.Warnln(err)
	}

	log.Infoln("See you next time!")
}
