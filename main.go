package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ccm/exporter/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
}

func main() {
	daystr := time.Now().Format("20060102")
	logFile, err := os.Create("./exporter/log/" + daystr + ".txt")
	defer logFile.Close()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	logger := log.New(logFile, "SR_", log.Ldate|log.Ltime|log.Lshortfile)

	reg := prometheus.NewPedanticRegistry()

	reg.MustRegister(metrics.AddLatency())

	gatherers := prometheus.Gatherers{
		reg,
	}

	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
	log.Println("Start server at :8710")
	logger.Printf("Start server at :8710")

	if err := http.ListenAndServe(":8710", nil); err != nil {
		log.Printf("Error occur when start server %v", err)
		logger.Printf("Error occur when start server %v", err)
		os.Exit(1)
	}
}
