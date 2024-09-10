// Copyright 2024 Richard Kosegi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/collectors/version"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/dynamic"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	pv "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"github.com/rkosegi/k8s-footprint-exporter/internal"

	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	name = "k8sfootprint_exporter"
)

func main() {
	var (
		metricsFile = kingpin.Flag(
			"metrics-file",
			"Path to configuration file with metrics definitions.",
		).Default("metrics.yaml").String()
		telemetryPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		disableDefaultMetrics = kingpin.Flag(
			"disable-default-metrics",
			"Exclude default metrics about the exporter itself (promhttp_*, process_*, go_*).",
		).Bool()
		namespace = kingpin.Flag(
			"namespace",
			"K8s namespace to watch for resources.",
		).Default("default").String()
		toolkitFlags = kingpinflag.AddFlags(kingpin.CommandLine, ":9998")
	)

	promlogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)

	kingpin.Version(pv.Print(name))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promslog.New(promlogConfig)

	logger.Info("Starting "+name, "version", pv.Info())
	logger.Info("Build context", "build_context", pv.BuildContext())
	logger.Info("Loading metrics definition from file", "file", metricsFile)

	var cfg internal.MetricConfig

	if err := cfg.LoadFrom(*metricsFile); err != nil {
		logger.Error("Couldn't load metrics config file", "err", err)
		os.Exit(1)
	}

	restCfg, err := ctrl.GetConfig()
	if err != nil {
		logger.Error("Couldn't get k8s client config", "err", err)
		os.Exit(1)
	}
	dc, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		logger.Error("Couldn't get dynamic k8s client", "err", err)
		os.Exit(1)
	}
	c, err := clientset.NewForConfig(restCfg)
	if err != nil {
		logger.Error("Couldn't get k8s client", "err", err)
		os.Exit(1)
	}
	vi, err := c.ServerVersion()
	if err != nil {
		logger.Error("Couldn't ping API server", "err", err)
		os.Exit(1)
	}
	logger.Info("Got response from API server", "version", vi.GitVersion)

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector(name))

	if err := r.Register(internal.NewCollector(&internal.CollectorOpts{
		Cfg:           &cfg,
		Namespace:     *namespace,
		DynamicClient: dc,
		Client:        c,
		Log:           logger,
	})); err != nil {
		logger.Error("Couldn't register "+name, "err", err)
		os.Exit(1)
	}

	handler := promhttp.HandlerFor(
		prometheus.Gatherers{r},
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		},
	)

	if !*disableDefaultMetrics {
		r.MustRegister(collectors.NewGoCollector())
		r.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		handler = promhttp.InstrumentMetricHandler(
			r, handler,
		)
	}

	landingPage, err := web.NewLandingPage(web.LandingConfig{
		Name:        strings.ReplaceAll(name, "_", " "),
		Description: "Prometheus Exporter for k8s API resources footprint",
		Version:     pv.Info(),
		Links: []web.LandingLinks{
			{
				Address: *telemetryPath,
				Text:    "Metrics",
			},
			{
				Address: "/health",
				Text:    "Health",
			},
			{
				Address: "/config",
				Text:    "Effective configuration",
			},
		},
	})
	if err != nil {
		logger.Error("Couldn't create landing page", "err", err)
		os.Exit(1)
	}

	http.Handle("/", landingPage)
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	http.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		_ = enc.Encode(cfg)
	})
	http.Handle(*telemetryPath, handler)

	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error starting server", "err", err)
		os.Exit(1)
	}
}
