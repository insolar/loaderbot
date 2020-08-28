/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// promGoroutines1 = promauto.NewGauge(prometheus.GaugeOpts{
	// 	Name: "loaderbot_goroutines_1",
	// 	Help: "loaderbot_goroutines_1",
	// })
	//
	// promGoroutines2 = promauto.NewGauge(prometheus.GaugeOpts{
	// 	Name: "loaderbot_goroutines_2",
	// 	Help: "loaderbot_goroutines_2",
	// })

	promTickSuccessRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_success_ratio",
		Help: "Success requests ratio",
	})
	promTickP50 = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p50",
		Help: "Response time 50 Percentile",
	})
	promTickP95 = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p95",
		Help: "Response time 95 Percentile",
	})
	promTickP99 = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_p99",
		Help: "Response time 99 Percentile",
	})
	promTickMax = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_max",
		Help: "Response time MAX",
	})
	promRPS = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "loaderbot_tick_rps",
		Help: "Requests per second rate",
	})
)

type PromReporter struct{}

func (m *PromReporter) reportTick(tm *TickMetrics) {
	promTickP50.Set(float64(tm.Metrics.Latencies.P50.Milliseconds()))
	promTickP95.Set(float64(tm.Metrics.Latencies.P95.Milliseconds()))
	promTickP99.Set(float64(tm.Metrics.Latencies.P99.Milliseconds()))
	promTickMax.Set(float64(tm.Metrics.Latencies.Max.Milliseconds()))
	promTickSuccessRatio.Set(tm.Metrics.Success)
	promRPS.Set(tm.Metrics.Rate)
}
