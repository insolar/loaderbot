/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

var (
	sigs = make(chan os.Signal, 1)
)

func (r *Runner) handleShutdownSignal() {
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-r.TimeoutCtx.Done():
			return
		case <-sigs:
			r.L.Infof("exit signal received, exiting")
			if r.Cfg.GoroutinesDump {
				buf := make([]byte, 1<<20)
				stacklen := runtime.Stack(buf, true)
				r.L.Infof("=== received SIGTERM ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
			}
			os.Exit(1)
		}
	}()
}

// nolint
// debug function to check metric correctness
func (r *Runner) reportEvery100Req() {
	if len(r.resultsLog)%100 == 0 {
		r.metricsMu.Lock()
		defer r.metricsMu.Unlock()
		m := NewMetrics()
		for _, r := range r.resultsLog[len(r.resultsLog)-100:] {
			m.add(r)
		}
		m.update(r)
		r.L.Infof("DEMAND rate [%4f -> %v], perc: 50 [%v] 95 [%v], # requests [%d], # attackers [%d], %% success [%d]",
			m.Rate,
			r.targetRPS,
			m.Latencies.P50,
			m.Latencies.P95,
			m.Requests,
			len(r.attackers),
			m.successLogEntry(),
		)
	}
}

func NewImmediateTicker(repeat time.Duration) *time.Ticker {
	ticker := time.NewTicker(repeat)
	oc := ticker.C
	nc := make(chan time.Time, 1)
	go func() {
		nc <- time.Now()
		for tm := range oc {
			nc <- tm
		}
	}()
	ticker.C = nc
	return ticker
}

func CreateFileOrAppend(fname string) *os.File {
	var file *os.File
	fpath, _ := filepath.Abs(fname)
	_, err := os.Stat(fpath)
	if err != nil {
		file, err = os.Create(fname)
	} else {
		file, err = os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func MaxRPS(array []float64) float64 {
	if len(array) == 0 {
		return 1
	}
	var max = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
	}
	return max
}
