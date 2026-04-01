package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Tracks
	TracksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_tracks_total",
			Help: "Total number of tracks detected",
		},
		[]string{"alert_level", "missile_type"},
	)

	TracksActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_tracks_active",
			Help: "Number of active tracks",
		},
	)

	// Alerts
	AlertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vimi_alerts_total",
			Help: "Total alerts issued",
		},
		[]string{"level"},
	)

	AlertsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vimi_alerts_active",
			Help: "Active alerts by level",
		},
		[]string{"level"},
	)

	// OPIR Ingest
	SightingsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_sightings_total",
			Help: "Total IR sightings ingested",
		},
	)

	// Sensor Fusion
	FusedTracksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_fusion_tracks_total",
			Help: "Total fused tracks produced",
		},
	)

	// DIS/HLA Gateway
	DISPDUsProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_dis_pdus_processed_total",
			Help: "DIS PDUs processed",
		},
	)

	HLAUpdatesProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_hla_updates_processed_total",
			Help: "HLA updates processed",
		},
	)

	// LVC Coordinator
	LVCEntitiesLive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_lvc_entities_live",
			Help: "Live LVC entities",
		},
	)

	LVCEntitiesVirtual = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_lvc_entities_virtual",
			Help: "Virtual LVC entities",
		},
	)

	LVCEntitiesConstructive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_lvc_entities_constructive",
			Help: "Constructive LVC entities",
		},
	)

	// Replay Engine
	RecordingsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_recordings_total",
			Help: "DIS recordings captured",
		},
	)

	ReplayEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_replay_events_total",
			Help: "Events replayed",
		},
	)

	// Data Catalog
	CatalogAssetsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "vimi_catalog_assets_total",
			Help: "Assets indexed in catalog",
		},
	)

	// Env Monitor
	EnvEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "vimi_env_events_total",
			Help: "Environmental events processed",
		},
	)

	// Kafka lag (generic)
	KafkaConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vimi_kafka_consumer_lag",
			Help: "Kafka consumer lag by topic/partition",
		},
		[]string{"topic", "partition"},
	)
)
