package metrics

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once     sync.Once
	instance *MetricsManager
)

type MetricsManager struct {
	reg *prometheus.Registry

	HTTPRequestsTotal  *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	HydrologyDataReceived  *prometheus.CounterVec
	HydrologyDataErrors    *prometheus.CounterVec
	LatestWaterLevel       *prometheus.GaugeVec
	LatestSedimentConcentration *prometheus.GaugeVec
	LatestFlowRate         *prometheus.GaugeVec
	LatestBedElevation     *prometheus.GaugeVec

	SimulationsStarted   *prometheus.CounterVec
	SimulationsCompleted *prometheus.CounterVec
	SimulationsFailed    *prometheus.CounterVec
	SimulationDuration   *prometheus.HistogramVec

	PredictionsStarted   prometheus.Counter
	PredictionsCompleted prometheus.Counter
	PredictionsFailed    prometheus.Counter
	PredictionDuration   prometheus.Histogram

	AlertsTriggered   *prometheus.CounterVec
	AlertsAcknowledged prometheus.Counter

	BusMessagesTotal   *prometheus.CounterVec
	BusMessagesDropped *prometheus.CounterVec

	WebSocketConnections prometheus.Gauge
	WebSocketMessagesSent prometheus.Counter

	MQTTMessagesPublished prometheus.Counter
	MQTTReconnects        prometheus.Counter

	DatabaseQueryDuration *prometheus.HistogramVec
	DatabaseErrors        prometheus.Counter

	GoGoroutines  prometheus.GaugeFunc
	GoMemoryAlloc prometheus.GaugeFunc
}

func GetMetrics() *MetricsManager {
	once.Do(func() {
		instance = newMetrics()
	})
	return instance
}

func newMetrics() *MetricsManager {
	reg := prometheus.NewRegistry()

	m := &MetricsManager{reg: reg}

	m.HTTPRequestsTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.HTTPRequestDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.HTTPRequestsInFlight = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently in flight",
		},
	)

	m.HydrologyDataReceived = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "hydrology_data_received_total",
			Help: "Total number of hydrology data points received",
		},
		[]string{"station_id", "data_type"},
	)

	m.HydrologyDataErrors = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "hydrology_data_errors_total",
			Help: "Total number of hydrology data errors",
		},
		[]string{"station_id", "error_type"},
	)

	m.LatestWaterLevel = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hydrology_water_level_meters",
			Help: "Latest water level in meters",
		},
		[]string{"station_id"},
	)

	m.LatestSedimentConcentration = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hydrology_sediment_concentration_kg_per_m3",
			Help: "Latest sediment concentration in kg/m^3",
		},
		[]string{"station_id"},
	)

	m.LatestFlowRate = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hydrology_flow_rate_m3_per_s",
			Help: "Latest flow rate in m^3/s",
		},
		[]string{"station_id"},
	)

	m.LatestBedElevation = promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hydrology_bed_elevation_meters",
			Help: "Latest bed elevation in meters",
		},
		[]string{"station_id"},
	)

	m.SimulationsStarted = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "simulations_started_total",
			Help: "Total number of simulations started",
		},
		[]string{"sim_type"},
	)

	m.SimulationsCompleted = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "simulations_completed_total",
			Help: "Total number of simulations completed",
		},
		[]string{"sim_type"},
	)

	m.SimulationsFailed = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "simulations_failed_total",
			Help: "Total number of simulations failed",
		},
		[]string{"sim_type"},
	)

	m.SimulationDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "simulation_duration_seconds",
			Help:    "Simulation duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"sim_type"},
	)

	m.PredictionsStarted = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "predictions_started_total",
			Help: "Total number of predictions started",
		},
	)

	m.PredictionsCompleted = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "predictions_completed_total",
			Help: "Total number of predictions completed",
		},
	)

	m.PredictionsFailed = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "predictions_failed_total",
			Help: "Total number of predictions failed",
		},
	)

	m.PredictionDuration = promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "prediction_duration_seconds",
			Help:    "Prediction duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
	)

	m.AlertsTriggered = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_triggered_total",
			Help: "Total number of alerts triggered",
		},
		[]string{"station_id", "severity"},
	)

	m.AlertsAcknowledged = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "alerts_acknowledged_total",
			Help: "Total number of alerts acknowledged",
		},
	)

	m.BusMessagesTotal = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "bus_messages_total",
			Help: "Total number of bus messages",
		},
		[]string{"message_type", "direction"},
	)

	m.BusMessagesDropped = promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "bus_messages_dropped_total",
			Help: "Total number of dropped bus messages",
		},
		[]string{"message_type", "reason"},
	)

	m.WebSocketConnections = promauto.With(reg).NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	m.WebSocketMessagesSent = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		},
	)

	m.MQTTMessagesPublished = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_messages_published_total",
			Help: "Total number of MQTT messages published",
		},
	)

	m.MQTTReconnects = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_reconnects_total",
			Help: "Total number of MQTT reconnections",
		},
	)

	m.DatabaseQueryDuration = promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"query_type"},
	)

	m.DatabaseErrors = promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors",
		},
	)

	m.GoGoroutines = promauto.With(reg).NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines",
		},
		func() float64 {
			return float64(runtime.NumGoroutine())
		},
	)

	m.GoMemoryAlloc = promauto.With(reg).NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "go_memory_alloc_bytes",
			Help: "Bytes of allocated heap objects",
		},
		func() float64 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			return float64(ms.Alloc)
		},
	)

	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return m
}

func (m *MetricsManager) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{
		Registry:          m.reg,
		EnableOpenMetrics: true,
		Timeout:           10 * time.Second,
	})
}

func (m *MetricsManager) Registry() *prometheus.Registry {
	return m.reg
}

func ObserveDuration(hist prometheus.Observer, fn func()) {
	start := time.Now()
	fn()
	hist.Observe(time.Since(start).Seconds())
}
