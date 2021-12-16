package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
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
	listenAddress = flag.String("listen-address", ":8080", "Address on which to expose metrics.")
	metricsPath   = flag.String("telemetry-path", "/metrics", "Path under which to expose metrics.")
	scrapeURI     = flag.String("scrape-uri", "http://grid.local", "URI on which to scrape Selenium Grid.")
)

type Exporter struct {
	URI                                                         string
	mutex                                                       sync.RWMutex
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
			// new information
			NodeCount float64 `json:"nodeCount"`
			Version   string  `json:"version"`
		} `json:"grid"`
		NodesInfo struct {
			Nodes []HubResponseNode `json:"nodes"`
		} `json:"nodesInfo"`
		// TODO sessionsInfo { sessionQueueRequests, sessions { capabilities, startTime, nodeId, sessionDurationMillis } }
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
	log.Infoln("Collecting data from:", uri)

	return &Exporter{
		URI: uri,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "up",
			Help:      "was the last scrape of Selenium Grid successful.",
		}),
		totalSlots: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "total_slots",
			Help:      "total number of usedSlots",
		}),
		maxSession: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "max_session",
			Help:      "maximum number of sessions",
		}),
		sessionCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "session_count",
			Help:      "number of active sessions",
		}),
		sessionQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "session_queue_size",
			Help:      "number of queued sessions",
		}),
		nodeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "node_count",
			Help:      "number of nodes",
		}),
		// NEW
		version: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: gridSubsystem,
			Name:      "version",
			Help:      "Hub/Router version",
		}, []string{versionLabel}),
		// nodes
		nodeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "status",
			Help:      "node status",
		}, []string{nodeIdLabel, nodeUriLabel, statusLabel}),
		nodeMaxSession: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "max_session",
			Help:      "maximum number of sessions on node",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeSlotCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "slot_count",
			Help:      "number of slots on node",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeSessionCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "session_count",
			Help:      "number of active sessions on node",
		}, []string{nodeIdLabel, nodeUriLabel}),
		nodeVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: nodeSubsystem,
			Name:      "version",
			Help:      "Node version",
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

	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.scrape()

	ch <- e.up
	ch <- e.totalSlots
	ch <- e.maxSession
	ch <- e.sessionCount
	ch <- e.sessionQueueSize
	//new
	ch <- e.nodeCount
	e.version.Collect(ch)
	e.nodeStatus.Collect(ch)
	e.nodeMaxSession.Collect(ch)
	e.nodeSlotCount.Collect(ch)
	e.nodeSessionCount.Collect(ch)
	e.nodeVersion.Collect(ch)

	return
}

func (e *Exporter) scrape() {

	e.totalSlots.Set(0)
	e.maxSession.Set(0)
	e.sessionCount.Set(0)
	e.sessionQueueSize.Set(0)
	e.nodeCount.Set(0)
	e.version.Reset()
	e.nodeStatus.Reset()
	e.nodeMaxSession.Reset()
	e.nodeSlotCount.Reset()
	e.nodeSessionCount.Reset()
	e.nodeVersion.Reset()

	body, err := e.fetch()
	if err != nil {
		e.up.Set(0)

		log.Errorf("Can't scrape Selenium Grid: %v", err)
		return
	}

	e.up.Set(1)

	var hResponse hubResponse

	if err := json.Unmarshal(body, &hResponse); err != nil {

		log.Errorf("Can't decode Selenium Grid response: %v", err)
		return
	}
	grid := hResponse.Data.Grid
	e.totalSlots.Set(grid.TotalSlots)
	e.maxSession.Set(grid.MaxSession)
	e.sessionCount.Set(grid.SessionCount)
	e.sessionQueueSize.Set(grid.SessionQueueSize)
	//new
	e.nodeCount.Set(grid.NodeCount)
	e.version.WithLabelValues(grid.Version).Add(1.0)
	for _, n := range hResponse.Data.NodesInfo.Nodes {
		e.nodeStatus.WithLabelValues(n.Id, n.Uri, n.Status).Add(1.0)
		e.nodeMaxSession.WithLabelValues(n.Id, n.Uri).Add(n.MaxSession)
		e.nodeSlotCount.WithLabelValues(n.Id, n.Uri).Add(n.SlotCount)
		e.nodeSessionCount.WithLabelValues(n.Id, n.Uri).Add(n.SessionCount)
		e.nodeVersion.WithLabelValues(n.Id, n.Uri, n.Version).Add(1.0)
	}
}

func (e Exporter) fetch() (output []byte, err error) {

	url := (e.URI + "/graphql")
	method := "POST"

	payload := strings.NewReader(`{
    "query": "{
          grid {totalSlots, maxSession, sessionCount, sessionQueueSize, nodeCount, version },
          nodesInfo { nodes { id, uri, status, maxSession, slotCount, sessionCount, version } }
      }"
  }`)

	client := http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	//s := string(body)
	//fmt.Println(s)
	return body, err
}

func main() {
	flag.Parse()

	log.Infoln("Starting selenium_grid_exporter")

	prometheus.MustRegister(NewExporter(*scrapeURI))
	prometheus.Unregister(prometheus.NewGoCollector())
	prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
