/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PromReporter struct {
	promTickSuccessRatio prometheus.Gauge
	promTickP50          prometheus.Gauge
	promTickP95          prometheus.Gauge
	promTickP99          prometheus.Gauge
	promTickMax          prometheus.Gauge
	promRPS              prometheus.Gauge
}

func NewPromReporter(label string) *PromReporter {
	m := &PromReporter{}
	m.promTickSuccessRatio = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_success_ratio",
		Help: "Success requests ratio",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promTickSuccessRatio)
	m.promTickP50 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p50",
		Help: "Response time 50 Percentile",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promTickP50)
	m.promTickP95 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p95",
		Help: "Response time 95 Percentile",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promTickP95)
	m.promTickP99 = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p99",
		Help: "Response time 99 Percentile",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promTickP99)
	m.promTickMax = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_max",
		Help: "Response time MAX",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promTickMax)
	m.promRPS = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_rps",
		Help: "Requests per second rate",
		ConstLabels: prometheus.Labels{
			"runner_name": label,
		},
	})
	_ = prometheus.Register(m.promRPS)
	return m
}

func (m *PromReporter) reportTick(tm *TickMetrics) {
	m.promTickP50.Set(float64(tm.Metrics.Latencies.P50.Milliseconds()))
	m.promTickP95.Set(float64(tm.Metrics.Latencies.P95.Milliseconds()))
	m.promTickP99.Set(float64(tm.Metrics.Latencies.P99.Milliseconds()))
	m.promTickMax.Set(float64(tm.Metrics.Latencies.Max.Milliseconds()))
	m.promTickSuccessRatio.Set(tm.Metrics.Success)
	m.promRPS.Set(tm.Metrics.Rate)
}
