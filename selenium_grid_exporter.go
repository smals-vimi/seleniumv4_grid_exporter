package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	nameSpace     = "selenium"
	gridSubsystem = "grid"
	nodeSubsystem = "node"
	nodeIdLabel   = "node_id"
	nodeUriLabel  = "node_uri"
	statusLabel   = "status"
	versionLabel  = "version"
)

var (
	versionFlag   = flag.Bool("version", false, "Prints the version and exits.")
	listenAddress = flag.String("listen-address", getEnv("LISTEN_ADDRESS", ":8080"), "Address on which to expose metrics.")
	metricsPath   = flag.String("telemetry-path", getEnv("TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
	scrapeURI     = flag.String("scrape-uri", getEnv("SCRAPE_URI", "http://grid.local"), "URI on which to scrape Selenium Grid.")
	httpTimeout   = flag.Duration("http-timeout", parseDuration(getEnv("HTTP_TIMEOUT", "5s")), "HTTP client timeout for scraping Selenium Grid.")
)

var (
	version   string
	gitCommit string
)

type Exporter struct {
	URI                                                         string
	up, totalSlots, maxSession, sessionCount, sessionQueueSize  prometheus.Gauge
	version                                                     *prometheus.GaugeVec
	nodeCount                                                   prometheus.Gauge
	nodeStatus, nodeMaxSession, nodeSlotCount, nodeSessionCount *prometheus.GaugeVec
	nodeVersion                                                 *prometheus.GaugeVec
}

type hubResponse struct {
	Data struct {
		Grid struct {
			TotalSlots       float64 `json:"totalSlots"`
			MaxSession       float64 `json:"maxSession"`
			SessionCount     float64 `json:"sessionCount"`
			SessionQueueSize float64 `json:"sessionQueueSize"`
			NodeCount        float64 `json:"nodeCount"`
			Version          string  `json:"version"`
		} `json:"grid"`
		NodesInfo struct {
			Nodes []HubResponseNode `json:"nodes"`
		} `json:"nodesInfo"`
	} `json:"data"`
}

type HubResponseNode struct {
	Id           string  `json:"id"`
	Uri          string  `json:"uri"`
	Status       string  `json:"status"`
	MaxSession   float64 `json:"maxSession"`
	SlotCount    float64 `json:"slotCount"`
	SessionCount float64 `json:"sessionCount"`
	Version      string  `json:"version"`
}

func NewExporter(uri string) *Exporter {
	logrus.Infoln("Collecting data from:", uri)

	return &Exporter{
		URI: uri,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "up",
			Help:      "Was the last scrape of Selenium Grid successful.",
		}),
		totalSlots: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "total_slots",
			Help:      "Total number of slots.",
		}),
		maxSession: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "max_session",
			Help:      "Maximum number of sessions.",
		}),
		sessionCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "session_count",
			Help:      "Number of active sessions.",
		}),
		sessionQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "session_queue_size",
			Help:      "Number of queued sessions.",
		}),
		nodeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "node_count",
			Help:      "Number of nodes.",
		}),
		version: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "version",
			Help:      "Hub/Router version.",
		}, []string{versionLabel}),
		nodeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "status",
			Help:      "Node status.",
		}, []string{nodeIdLabel, nodeUriLabel, statusLabel}),
		nodeMaxSession: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "max_session",
			Help:      "Maximum number of sessions on node.",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeSlotCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "slot_count",
			Help:      "Number of slots on node.",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeSessionCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "session_count",
			Help:      "Number of active sessions on node.",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "version",
			Help:      "Node version.",
		}, []string{nodeIdLabel, nodeUriLabel, versionLabel}),
	}
}

/*
Describe is called by Prometheus on startup of this monitor. It needs to tell
the caller about all of the available metrics. It is also called during "unregister".
*/
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.totalSlots.Describe(ch)
	e.maxSession.Describe(ch)
	e.sessionCount.Describe(ch)
	e.sessionQueueSize.Describe(ch)
	e.nodeCount.Describe(ch)
	e.version.Describe(ch)
	e.nodeStatus.Describe(ch)
	e.nodeMaxSession.Describe(ch)
	e.nodeSlotCount.Describe(ch)
	e.nodeSessionCount.Describe(ch)
	e.nodeVersion.Describe(ch)
}

/*
Collect is called by Prometheus at regular intervals to provide current data
*/
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape()

	ch <- e.up
	ch <- e.totalSlots
	ch <- e.maxSession
	ch <- e.sessionCount
	ch <- e.sessionQueueSize
	ch <- e.nodeCount
	e.version.Collect(ch)
	e.nodeStatus.Collect(ch)
	e.nodeMaxSession.Collect(ch)
	e.nodeSlotCount.Collect(ch)
	e.nodeSessionCount.Collect(ch)
	e.nodeVersion.Collect(ch)
}

// func (e *Exporter) scrape() {

// 	e.totalSlots.Set(0)
// 	e.maxSession.Set(0)
// 	e.sessionCount.Set(0)
// 	e.sessionQueueSize.Set(0)
// 	e.nodeCount.Set(0)
// 	e.version.Reset()
// 	e.nodeStatus.Reset()
// 	e.nodeMaxSession.Reset()
// 	e.nodeSlotCount.Reset()
// 	e.nodeSessionCount.Reset()
// 	e.nodeVersion.Reset()

// 	body, err := e.fetch()
// 	if err != nil {
// 		e.up.Set(0)

// 		logrus.Errorf("Error scraping Selenium Grid: %v", err)
// 		return
// 	}

// 	e.up.Set(1)

// 	var hResponse hubResponse

// 	if err := json.Unmarshal(body, &hResponse); err != nil {

// 		logrus.Errorf("Error decoding Selenium Grid response: %v", err)
// 		return
// 	}
// 	grid := hResponse.Data.Grid
// 	e.totalSlots.Set(grid.TotalSlots)
// 	e.maxSession.Set(grid.MaxSession)
// 	e.sessionCount.Set(grid.SessionCount)
// 	e.sessionQueueSize.Set(grid.SessionQueueSize)
// 	//new
// 	e.nodeCount.Set(grid.NodeCount)
// 	e.version.WithLabelValues(grid.Version).Set(1.0)
// 	for _, n := range hResponse.Data.NodesInfo.Nodes {
// 		e.nodeStatus.WithLabelValues(n.Id, n.Uri, n.Status).Set(1.0)
// 		e.nodeMaxSession.WithLabelValues(n.Id, n.Uri).Set(n.MaxSession)
// 		e.nodeSlotCount.WithLabelValues(n.Id, n.Uri).Set(n.SlotCount)
// 		e.nodeSessionCount.WithLabelValues(n.Id, n.Uri).Set(n.SessionCount)
// 		e.nodeVersion.WithLabelValues(n.Id, n.Uri, n.Version).Set(1.0)
// 	}
// }

func (e *Exporter) scrape() {
	body, err := e.fetch()
	if err != nil {
		e.up.Set(0) // Indicate scrape failure
		logrus.Errorf("Error scraping Selenium Grid: %v", err)

		// Clear node-specific metrics completely
		e.nodeStatus.Reset()
		e.nodeMaxSession.Reset()
		e.nodeSlotCount.Reset()
		e.nodeSessionCount.Reset()
		e.nodeVersion.Reset()
		return
	}

	e.up.Set(1) // Indicate scrape success
	logrus.Info("Successfully scraped Selenium Grid")

	var hResponse hubResponse
	if err := json.Unmarshal(body, &hResponse); err != nil {
		logrus.Errorf("Error decoding Selenium Grid response: %v", err)
		e.up.Set(0)

		// Clear node-specific metrics completely
		e.nodeStatus.Reset()
		e.nodeMaxSession.Reset()
		e.nodeSlotCount.Reset()
		e.nodeSessionCount.Reset()
		e.nodeVersion.Reset()
		return
	}

	// Update grid metrics
	grid := hResponse.Data.Grid
	e.totalSlots.Set(grid.TotalSlots)
	e.maxSession.Set(grid.MaxSession)
	e.sessionCount.Set(grid.SessionCount)
	e.sessionQueueSize.Set(grid.SessionQueueSize)
	e.nodeCount.Set(grid.NodeCount)
	e.version.WithLabelValues(grid.Version).Set(1.0)

	// Update node-specific metrics
	e.nodeStatus.Reset()
	e.nodeMaxSession.Reset()
	e.nodeSlotCount.Reset()
	e.nodeSessionCount.Reset()
	e.nodeVersion.Reset()

	for _, n := range hResponse.Data.NodesInfo.Nodes {
		e.nodeStatus.WithLabelValues(n.Id, n.Uri, n.Status).Set(1.0)
		e.nodeMaxSession.WithLabelValues(n.Id, n.Uri).Set(n.MaxSession)
		e.nodeSlotCount.WithLabelValues(n.Id, n.Uri).Set(n.SlotCount)
		e.nodeSessionCount.WithLabelValues(n.Id, n.Uri).Set(n.SessionCount)
		e.nodeVersion.WithLabelValues(n.Id, n.Uri, n.Version).Set(1.0)
	}
}

func (e Exporter) fetch() ([]byte, error) {
	client := http.Client{Timeout: *httpTimeout}
	req, err := http.NewRequest("POST", e.URI+"/graphql", strings.NewReader(`{
        "query": "{
            grid {totalSlots, maxSession, sessionCount, sessionQueueSize, nodeCount, version },
            nodesInfo { nodes { id, uri, status, maxSession, slotCount, sessionCount, version } }
        }"
    }`))
	if err != nil {
		logrus.Errorf("Failed to create request: %v", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("Unexpected HTTP status: %s", resp.Status)
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Failed to read response body: %v", err)
		return nil, err
	}

	return body, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseDuration(duration string) time.Duration {
	d, err := time.ParseDuration(duration)
	if err != nil {
		logrus.Warnf("Invalid duration format for HTTP_TIMEOUT: %v, defaulting to 5s", err)
		return 5 * time.Second
	}
	return d
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Selenium Grid Exporter v%s (%s)\n", version, gitCommit)
		os.Exit(0)
	}

	logrus.Infof("Starting Selenium Grid Exporter version %s", version)
	logrus.Infof("Listening on %s", *listenAddress)
	logrus.Infof("Scraping Selenium Grid at %s", *scrapeURI)
	logrus.Infof("Metrics path: %s", *metricsPath)
	logrus.Infof("HTTP client timeout: %s", httpTimeout.String())

	exporter := NewExporter(*scrapeURI)
	prometheus.MustRegister(exporter)
	prometheus.Unregister(prometheus.NewGoCollector())
	prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to Selenium Grid Exporter! Metrics are available at " + *metricsPath))
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	logrus.Fatal(http.ListenAndServe(*listenAddress, nil))
}
