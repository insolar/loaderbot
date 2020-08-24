/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Report struct {
	runId               string
	runName             string
	metricsLogFilename  string
	percsReportFilename string
	percLogFilename     string
	metricsLogFile      *csv.Writer
	percLogFile         *csv.Writer
	reportOptions       *ReportOptions
	L                   *Logger
}

func NewReport(cfg *RunnerConfig) *Report {
	tn := time.Now().Unix()
	runId := uuid.New().String()
	metricsLogFilename := fmt.Sprintf(MetricsLogFile, cfg.Name, runId, tn)
	percsReportFilename := fmt.Sprintf(ReportGraphFile, cfg.Name, runId, tn)
	percLogFilename := fmt.Sprintf(PercsLogFile, cfg.Name, runId, tn)
	r := &Report{
		runId:               runId,
		runName:             cfg.Name,
		metricsLogFilename:  metricsLogFilename,
		percsReportFilename: percsReportFilename,
		percLogFilename:     percLogFilename,
		metricsLogFile:      csv.NewWriter(CreateFileOrReplace(metricsLogFilename)),
		percLogFile:         csv.NewWriter(CreateFileOrReplace(percLogFilename)),
		reportOptions:       cfg.ReportOptions,
		L:                   NewLogger(cfg).With("report", cfg.Name),
	}
	_ = r.metricsLogFile.Write(ResultsCsvHeader)
	_ = r.percLogFile.Write(PercsCsvHeader)
	return r
}

func (r *Report) plot() {
	if r.reportOptions.PNG {
		r.L.Infof("reporting graphs: %s", r.percLogFilename)
		chart, err := PercsChart(r.percLogFilename, r.runName)
		if err != nil {
			r.L.Error(err)
			return
		}
		RenderEChart(chart, r.percsReportFilename)
		// html2png(r.percsReportFilename)
	}
}

func (r *Report) flushLogs() {
	r.percLogFile.Flush()
	r.metricsLogFile.Flush()
}

func (r *Report) writeResultEntry(res AttackResult, errorMsg string) {
	_ = r.metricsLogFile.Write([]string{
		res.DoResult.RequestLabel,
		strconv.Itoa(int(res.Begin.UnixNano())),
		strconv.Itoa(int(res.End.UnixNano())),
		res.Elapsed.String(),
		string(res.DoResult.StatusCode),
		errorMsg,
	})
}

func (r *Report) writePercentilesEntry(res AttackResult, tickMetrics *Metrics) {
	_ = r.percLogFile.Write([]string{
		res.DoResult.RequestLabel,
		strconv.Itoa(res.AttackToken.Tick),
		strconv.Itoa(int(tickMetrics.Rate)),
		strconv.Itoa(int(tickMetrics.Latencies.P50.Milliseconds())),
		strconv.Itoa(int(tickMetrics.Latencies.P95.Milliseconds())),
		strconv.Itoa(int(tickMetrics.Latencies.P99.Milliseconds())),
	})
}
