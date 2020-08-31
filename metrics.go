/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"strconv"
	"time"

	"github.com/streadway/quantile"
)

type Metrics struct {
	// Latencies holds computed request latency Metrics.
	Latencies LatencyMetrics `json:"latencies"`
	// First is the earliest timestamp in a Result set.
	Earliest time.Time `json:"earliest"`
	// Latest is the latest timestamp in a Result set.
	Latest time.Time `json:"latest"`
	// End is the latest timestamp in a Result set plus its latency.
	End time.Time `json:"End"`
	// Duration is the duration of the attack.
	Duration time.Duration `json:"duration"`
	// Wait is the extra time waiting for responses from targets.
	Wait time.Duration `json:"wait"`
	// Requests is the total number of requests executed.
	Requests uint64 `json:"requests"`
	// TargetRate is the rate of requests per second demanded in current step.
	TargetRate float64 `json:"target_rate"`
	// Rate is the rate of requests per second.
	Rate float64 `json:"rate"`
	// Success is the percentage of non-error responses.
	Success float64 `json:"success"`
	// StatusCodes is a histogram of the responses' status codes.
	StatusCodes map[string]int `json:"status_codes"`
	// Errors is a set of unique Errors returned by the targets during the attack.
	Errors []string `json:"Errors"`

	errors      map[string]struct{}
	errorsCount int64
	success     int64
	latencies   *quantile.Estimator
}

// LatencyMetrics holds computed request latency Metrics.
type LatencyMetrics struct {
	// Total is the total latency sum of all requests in an attack.
	Total time.Duration `json:"total"`
	// Mean is the mean request latency.
	Mean time.Duration `json:"mean"`
	// P50 is the 50th percentile request latency.
	P50 time.Duration `json:"50th"`
	// P95 is the 95th percentile request latency.
	P95 time.Duration `json:"95th"`
	// P99 is the 99th percentile request latency.
	P99 time.Duration `json:"99th"`
	// Max is the maximum observed request latency.
	Max time.Duration `json:"max"`
}

func NewMetrics() *Metrics {
	m := &Metrics{}
	m.init()
	return m
}

func (m Metrics) successLogEntry() float64 {
	s := m.Success * 100.0
	if s < 0 {
		return 0
	}
	return s
}

// nolint
func (m Metrics) meanLogEntry() time.Duration {
	lm := m.Latencies.Mean
	if lm < 0 {
		return time.Duration(0)
	}
	return lm
}

func (m *Metrics) add(r AttackResult) {
	m.Requests++
	// StatusCode is optional
	if r.DoResult.StatusCode > 0 {
		m.StatusCodes[strconv.Itoa(r.DoResult.StatusCode)]++
	}
	m.Latencies.Total += r.Elapsed

	m.latencies.Add(float64(r.Elapsed))

	if m.Earliest.IsZero() || m.Earliest.After(r.Begin) {
		m.Earliest = r.Begin
	}

	if r.Begin.After(m.Latest) {
		m.Latest = r.Begin
	}

	if end := r.End; end.After(m.End) {
		m.End = end
	}

	if r.Elapsed > m.Latencies.Max {
		m.Latencies.Max = r.Elapsed
	}

	if r.DoResult.Error != "" {
		if _, ok := m.errors[r.DoResult.Error]; !ok {
			m.errors[r.DoResult.Error] = struct{}{}
			m.Errors = append(m.Errors, r.DoResult.Error)
		}
		m.errorsCount++
	} else {
		if r.DoResult.StatusCode == 0 || (r.DoResult.StatusCode >= 200 && r.DoResult.StatusCode < 400) {
			m.success++
		}
	}
}

// update computes derived summary Metrics which don't need to be Run on every add call.
func (m *Metrics) update() {
	fRequests := float64(m.Requests)
	m.Duration = m.Latest.Sub(m.Earliest)
	if secs := m.Duration.Seconds(); secs > 0 {
		m.Rate = fRequests / secs
	}
	m.Wait = m.End.Sub(m.Latest)
	m.Success = float64(m.success) / fRequests
	m.Latencies.Mean = time.Duration(float64(m.Latencies.Total) / fRequests)
	m.Latencies.P50 = time.Duration(m.latencies.Get(0.50))
	m.Latencies.P95 = time.Duration(m.latencies.Get(0.95))
	m.Latencies.P99 = time.Duration(m.latencies.Get(0.99))
}

func (m *Metrics) init() {
	if m.latencies == nil {
		m.StatusCodes = map[string]int{}
		m.errors = map[string]struct{}{}
		m.latencies = quantile.New(
			quantile.Known(0.50, 0.01),
			quantile.Known(0.95, 0.001),
			quantile.Known(0.99, 0.0005),
		)
	}
}
