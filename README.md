# Selenium Grid exporter

A [Prometheus](https://prometheus.io/) exporter that collects [Selenium Grid](http://www.seleniumhq.org/projects/grid/) metrics.

### Usage

```sh
$ docker run -it mcopjan/seleniumv4_grid_exporter:latest -h
Usage of /selenium_grid_exporter:
  -listen-address string
      Address on which to expose metrics. (default ":8080")
  -scrape-uri string
      URI on which to scrape Selenium Grid. (default "http://grid.local")
  -telemetry-path string
      Path under which to expose metrics. (default "/metrics")
```

### Prometheus/Grafana example

```
  - run docker-compose -f docker-compose.yml up
  - open grafana at localhost:3000 (admin/foobar)
  - open Dashboards -> Manage -> Selenium4 Grid monitoring
  
```
  ![Screenshot](selenium4_grafana.png)

## Metrics

```
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 2558
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
# HELP selenium_grid_max_session maximum number of sessions
# TYPE selenium_grid_max_session gauge
selenium_grid_max_session 24
# HELP selenium_grid_node_count number of nodes
# TYPE selenium_grid_node_count gauge
selenium_grid_node_count 3
# HELP selenium_grid_session_count number of active sessions
# TYPE selenium_grid_session_count gauge
selenium_grid_session_count 0
# HELP selenium_grid_session_queue_size number of queued sessions
# TYPE selenium_grid_session_queue_size gauge
selenium_grid_session_queue_size 0
# HELP selenium_grid_total_slots total number of usedSlots
# TYPE selenium_grid_total_slots gauge
selenium_grid_total_slots 72
# HELP selenium_grid_up was the last scrape of Selenium Grid successful.
# TYPE selenium_grid_up gauge
selenium_grid_up 1
# HELP selenium_grid_version Hub/Router version
# TYPE selenium_grid_version gauge
selenium_grid_version{version="4.1.0 (revision 87802e897b)"} 1
# HELP selenium_node_max_session maximum number of sessions on node
# TYPE selenium_node_max_session gauge
selenium_node_max_session{node_id="2736eba3-8259-4bd0-98fd-05fa56ddad7f",node_uri="http://10.0.1.1:5555"} 8
selenium_node_max_session{node_id="e8e01785-4079-4adf-9106-d3bf5bc92ed7",node_uri="http://10.0.1.2:5555"} 8
selenium_node_max_session{node_id="fee80b10-a134-4452-88c6-416d1805b3f9",node_uri="http://10.0.1.3:5555"} 8
# HELP selenium_node_session_count number of active sessions on node
# TYPE selenium_node_session_count gauge
selenium_node_session_count{node_id="2736eba3-8259-4bd0-98fd-05fa56ddad7f",node_uri="http://10.0.1.1:5555"} 0
selenium_node_session_count{node_id="e8e01785-4079-4adf-9106-d3bf5bc92ed7",node_uri="http://10.0.1.2:5555"} 0
selenium_node_session_count{node_id="fee80b10-a134-4452-88c6-416d1805b3f9",node_uri="http://10.0.1.3:5555"} 0
# HELP selenium_node_slot_count number of slots on node
# TYPE selenium_node_slot_count gauge
selenium_node_slot_count{node_id="2736eba3-8259-4bd0-98fd-05fa56ddad7f",node_uri="http://10.0.1.1:5555"} 24
selenium_node_slot_count{node_id="e8e01785-4079-4adf-9106-d3bf5bc92ed7",node_uri="http://10.0.1.2:5555"} 24
selenium_node_slot_count{node_id="fee80b10-a134-4452-88c6-416d1805b3f9",node_uri="http://10.0.1.3:5555"} 24
# HELP selenium_node_status node status
# TYPE selenium_node_status gauge
selenium_node_status{node_id="2736eba3-8259-4bd0-98fd-05fa56ddad7f",node_uri="http://10.0.1.1:5555",status="UP"} 1
selenium_node_status{node_id="e8e01785-4079-4adf-9106-d3bf5bc92ed7",node_uri="http://10.0.1.2:5555",status="UP"} 1
selenium_node_status{node_id="fee80b10-a134-4452-88c6-416d1805b3f9",node_uri="http://10.0.1.3:5555",status="UP"} 1
# HELP selenium_node_version Node version
# TYPE selenium_node_version gauge
selenium_node_version{node_id="2736eba3-8259-4bd0-98fd-05fa56ddad7f",node_uri="http://10.0.1.1:5555",version="4.1.0 (revision 87802e897b)"} 1
selenium_node_version{node_id="e8e01785-4079-4adf-9106-d3bf5bc92ed7",node_uri="http://10.0.1.2:5555",version="4.1.0 (revision 87802e897b)"} 1
selenium_node_version{node_id="fee80b10-a134-4452-88c6-416d1805b3f9",node_uri="http://10.0.1.3:5555",version="4.1.0 (revision 87802e897b)"} 1
```
