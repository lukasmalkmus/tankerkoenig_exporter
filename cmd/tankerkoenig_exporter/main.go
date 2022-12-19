package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"

	"github.com/lukasmalkmus/tankerkoenig_exporter/internal/client"
	"github.com/lukasmalkmus/tankerkoenig_exporter/internal/exporter"
)

const usage = `Usage:
    tankerkoenig_exporter [--tankerkoenig.api-key KEY] (--tankerkoenig.stations UUID... | --tankerkoenig.location GEOHASH [--tankerkoenig.radius KM] [--tankerkoenig.radius e5|e10|diesel|all]) [--web.listen-address ADDRESS] [--web.telemetry-path PATH]

Options:
	--tankerkoenig.api-key KEY       API key for the Tankerkoenig API (default: TANKERKOENIG_API_KEY environment variable)
	--tankerkoenig.stations UUID     UUID of a station. The flag can be reused to specify multiple stations
	--tankerkoenig.location GEOHASH  Location at which to search for stations
	--tankerkoenig.radius KM         Kilometer radius in which to search for stations (default: 10)
	--tankerkoenig.product PRODUCT   Only include stations which have given product. Must be one of e5, e10, diesel or all (default: all)
	--web.listen-address ADDRESS     Listen address for the web server (default: :9386)
	--web.telemetry-path PATH        Path under which to expose metrics (default: /metrics)

Example:
    $ tankerkoenig_exporter --tankerkoenig.stations 51d4b55e-a095-1aa0-e100-80009459e03a
    $ tankerkoenig_exporter --tankerkoenig.location u0yjjd6jk0zj7 --tankerkoenig.radius=3 --tankerkoenig.product=e5

The --tankerkoenig.stations flag is mutually exclusive with the --tankerkoenig.location flag.
`

type stringSliceValue []string

func newStringSliceValue(p *[]string) *stringSliceValue {
	return (*stringSliceValue)(p)
}

// Set implements [flag.Value].
func (v *stringSliceValue) Set(s string) error {
	if strings.Contains(s, ",") {
		*v = strings.Split(s, ",")
	} else {
		*v = append(*v, s)
	}
	return nil
}

// String implements [flag.Value].
func (v stringSliceValue) String() string { return strings.Join(v, ",") }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.SetFlags(0)

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	var (
		versionFlag bool
		tkAPIKey    string
		tkStations  []string
		tkLocation  string
		tkRadius    int
		// tkProduct        string
		webListenAddress string
		webTelemetryPath string
	)

	flag.BoolVar(&versionFlag, "v", false, "print the version")
	flag.BoolVar(&versionFlag, "version", false, "print the version")
	flag.StringVar(&tkAPIKey, "tankerkoenig.api-key", os.Getenv("TANKERKOENIG_API_KEY"), "api key")
	flag.Var(newStringSliceValue(&tkStations), "tankerkoenig.stations", "station ids")
	flag.StringVar(&tkLocation, "tankerkoenig.location", "", "search location")
	flag.IntVar(&tkRadius, "tankerkoenig.radius", 10, "search radius")
	// flag.StringVar(&tkProduct, "tankerkoenig.product", "all", "only include stations with given product")
	flag.StringVar(&webListenAddress, "web.listen-address", ":9386", "listen address")
	flag.StringVar(&webTelemetryPath, "web.telemetry-path", "/metrics", "metrics path")

	flag.Parse()

	if versionFlag {
		if version.Print("tankerkoenig_exporter") != "" {
		} else if buildInfo, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(buildInfo.Main.Version)
		} else {
			fmt.Println("(unknown)")
		}
		return
	}

	if flag.NArg() != 0 {
		errorf("too many arguments")
	}

	if len(tkAPIKey) == 0 {
		errorWithHint("missing api key", "did you forget to export TANKERKOENIG_API_KEY?")
	}
	if len(webListenAddress) == 0 {
		errorWithHint("missing listen address", "did you forget to specify --web.listen-address?")
	}
	if len(webTelemetryPath) == 0 {
		errorWithHint("missing telemetry path", "did you forget to specify --web.telemetry-path?")
	}

	switch {
	case len(tkStations) > 0:
		if len(tkLocation) > 0 {
			errorf("--tankerkoenig.location can't be used with --tankerkoenig.stations")
		}
	case len(tkLocation) > 0:
		if len(tkStations) > 0 {
			errorf("--tankerkoenig.stations can't be used with --tankerkoenig.location")
		}
		if tkRadius == 0 {
			errorWithHint("missing radius", "did you forget to specify --tankerkoenig.radius?")
		}
		// if tkProduct != "e5" && tkProduct != "e10" && tkProduct != "diesel" && tkProduct != "all" {
		// 	errorWithHint("invalid product", "--tankerkoenig.product must be one of e5, e10, diesel or all")
		// }
	default:
		errorf("must specify one of --tankerkoenig.stations or --tankerkoenig.location")
	}

	var (
		logger    = log.New(os.Stderr, "exporter", 0)
		apiClient = client.New(tkAPIKey)
		collector prometheus.Collector
		err       error
	)
	switch {
	case len(tkStations) > 0:
		collector, err = exporter.NewForStations(logger, apiClient, tkStations)
	case len(tkLocation) > 0:
		collector, err = exporter.NewForLocation(logger, apiClient, tkLocation, tkRadius)
		// collector, err = exporter.NewForLocation(logger, apiClient, tkLocation, tkRadius, tkProduct)
	}
	if err != nil {
		errorf("create exporter: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()

	if err := reg.Register(collector); err != nil {
		errorf("register tankerkoenig collector: %v", err)
	}
	if err := reg.Register(version.NewCollector("tk_exporter")); err != nil {
		errorf("register version collector: %v", err)
	}

	mux := http.NewServeMux()

	mux.Handle(webTelemetryPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog: log.New(os.Stderr, "promhttp", 0),
		Timeout:  time.Second * 15,
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
		<head><title>Tankerkoenig API Exporter</title></head>
		<body>
		<h1>Tankerkoenig API Exporter</h1>
		<p><a href='` + webTelemetryPath + `'>Metrics</a></p>
		</body>
		</html>`))
	})

	srv := &http.Server{
		Addr:         webListenAddress,
		Handler:      mux,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 15,
		ErrorLog:     log.New(os.Stderr, "server", 0),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		if err := srv.Shutdown(context.Background()); err != nil {
			errorf("shutdown server: %v", err)
		}
	case err := <-errCh:
		errorf("server error: %v", err)
	}
}

func errorf(format string, v ...any) {
	log.Fatalf("retentioner: error: "+format, v...)
}

func errorWithHint(msg string, hints ...string) {
	log.Printf("retentioner: error: %s", msg)
	for _, hint := range hints {
		log.Printf("retentioner: hint: %s", hint)
	}
	os.Exit(1)
}
