package main

import (
	internalPrometheus "PrometheusF6005/prometheus"
	"cmp"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	reloginDelay := 60 * time.Second
	if seconds, err := strconv.Atoi(os.Getenv("ONT_RELOGIN_DELAY")); err == nil && seconds >= 0 {
		reloginDelay = time.Duration(seconds) * time.Second
	}

	collector := internalPrometheus.NewONTCollector(
		cmp.Or(strings.TrimRight(os.Getenv("ENDPOINT"), "/"), "http://192.168.1.1"),
		cmp.Or(os.Getenv("ONT_USERNAME"), "admin"),
		cmp.Or(os.Getenv("ONT_PASSWORD"), "admin"),
		reloginDelay,
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: false,
	}))

	log.Fatal(http.ListenAndServe(":80", nil))
}
