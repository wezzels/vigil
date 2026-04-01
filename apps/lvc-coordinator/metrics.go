package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func init() {
	_ = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vimi_sightings_processed_total",
		Help: "Total IR sightings processed",
	})
	_ = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vimi_kafka_messages_sent_total",
		Help: "Kafka messages sent by topic",
	}, []string{"topic"})
	_ = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vimi_kafka_messages_received_total",
		Help: "Kafka messages received by topic",
	}, []string{"topic"})
	_ = promauto.NewCounter(prometheus.CounterOpts{
		Name: "vimi_errors_total",
		Help: "Total errors",
	})
}
