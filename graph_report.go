/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/go-echarts/go-echarts/charts"
)

type ChartLine struct {
	XValues []float64
	YValues []float64
}

func parseScalingData(path string) (map[string]*ChartLine, error) {
	reader := openCSV(path)
	requests := make(map[string]*ChartLine)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) != 3 {
			return nil, errors.New("malformed csv")
		}

		methodName := record[0]
		veCount := record[1]
		maxRPS := record[2]
		if _, ok := requests[methodName]; !ok {
			requests[methodName] = &ChartLine{}
		}
		xValue, err := strconv.ParseFloat(veCount, 64)
		if err != nil {
			return nil, err
		}
		// VE count
		requests[methodName].XValues = append(requests[methodName].XValues, xValue)
		yValue, err := strconv.ParseFloat(maxRPS, 64)
		if err != nil {
			return nil, err
		}
		// Max TargetRPS
		requests[methodName].YValues = append(requests[methodName].YValues, yValue)
	}
	for _, v := range requests {
		if len(v.XValues) == 0 || len(v.YValues) == 0 {
			return nil, errors.New("empty csv, nothing to plot")
		}
	}
	return requests, nil
}

func parsePercsData(path string) (map[string]*ChartLine, error) {
	reader := openCSV(path)
	percs := map[string]*ChartLine{
		"rps": {},
		"p50": {},
		"p95": {},
		"p99": {},
	}
	// skip csv header
	_, _ = reader.Read()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) != 6 {
			return nil, errors.New("malformed csv")
		}

		second := record[1]
		rps := record[2]
		p50 := record[3]
		p95 := record[4]
		p99 := record[5]

		xValue, err := strconv.ParseFloat(second, 64)
		if err != nil {
			return nil, err
		}
		// seconds
		percs["rps"].XValues = append(percs["rps"].XValues, xValue)
		percs["p50"].XValues = append(percs["p50"].XValues, xValue)
		percs["p95"].XValues = append(percs["p95"].XValues, xValue)
		percs["p99"].XValues = append(percs["p99"].XValues, xValue)
		yRPS, err := strconv.ParseFloat(rps, 64)
		if err != nil {
			return nil, err
		}
		// rps
		percs["rps"].YValues = append(percs["rps"].YValues, yRPS)
		yValue, err := strconv.ParseFloat(p50, 64)
		if err != nil {
			return nil, err
		}
		// percentiles
		percs["p50"].YValues = append(percs["p50"].YValues, yValue)
		yValue95, err := strconv.ParseFloat(p95, 64)
		if err != nil {
			return nil, err
		}
		percs["p95"].YValues = append(percs["p95"].YValues, yValue95)
		yValue99, err := strconv.ParseFloat(p99, 64)
		if err != nil {
			return nil, err
		}
		percs["p99"].YValues = append(percs["p99"].YValues, yValue99)
	}
	for _, v := range percs {
		if len(v.XValues) == 0 || len(v.YValues) == 0 {
			return nil, errors.New("empty csv, nothing to plot")
		}
	}
	return percs, nil
}

func PercsChart(path string, title string) (*charts.Line, error) {
	d, err := parsePercsData(path)
	if err != nil {
		return nil, err
	}
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.DataZoomOpts{},
		charts.TitleOpts{Title: title},
		charts.XAxisOpts{Name: "Time (sec)"},
		charts.YAxisOpts{Name: "Response (ms)"},
	)
	line.AddXAxis(d["rps"].XValues)
	for k, v := range d {
		line.AddYAxis(k, v.YValues, defaultMaxLabel(k)...)
	}
	return line, nil
}

func ScalingChart(path string, title string) (*charts.Line, error) {
	d, err := parseScalingData(path)
	if err != nil {
		return nil, err
	}
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.TitleOpts{Title: title},
		charts.XAxisOpts{Name: "Nodes"},
		charts.YAxisOpts{Name: "RPS"},
	)
	for k, v := range d {
		line.AddXAxis(v.XValues)
		line.AddYAxis(k, v.YValues, defaultMaxLabel(k)...)
	}
	return line, nil
}

func RenderEChart(data *charts.Line, name string) {
	f, err := os.Create(name)
	if err != nil {
		log.Println(err)
	}
	if err := data.Render(f); err != nil {
		log.Fatal(err)
	}
}

// ReportScaling scaling chart, data must be written in csv in format:
// ${handle_name},${network_nodes},${max_rps}
func ReportScaling(inputCsv, outHtml string) {
	chartData, err := ScalingChart(inputCsv, "scaling")
	if err != nil {
		log.Fatal("Couldn't read and parse requests", err)
	}
	RenderEChart(chartData, outHtml)
	// html2png(outHtml)
}

// draws max label for every line
func defaultMaxLabel(metric string) []charts.SeriesOptser {
	return []charts.SeriesOptser{
		charts.MPNameTypeItem{Name: "max " + metric, Type: "max"},
		charts.MPStyleOpts{Label: charts.LabelTextOpts{Show: true}},
	}
}

func openCSV(path string) *csv.Reader {
	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	return csv.NewReader(csvFile)
}
