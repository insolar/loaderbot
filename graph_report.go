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
	"github.com/wcharczuk/go-chart"
	"io"
	"log"
	"os"
	"strconv"
)

type ChartLine struct {
	XValues []float64
	YValues []float64
}

func ScalingChart(path string) (*chart.Chart, error) {
	reader, close := openCSV(path)
	defer close()
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
	var series []chart.Series
	var colorIndex int
	for key, value := range requests {
		series = append(series, chart.ContinuousSeries{
			Name: key,
			Style: chart.Style{
				StrokeColor: chart.GetDefaultColor(colorIndex).WithAlpha(255),
				DotWidth:    3.0,
				StrokeWidth: 3,
			},
			XValues: value.XValues,
			YValues: value.YValues,
		})
		colorIndex++
	}
	chartData := &chart.Chart{
		XAxis: chart.XAxis{
			Name: "VE count",
		},
		YAxis: chart.YAxis{
			Name: "Max TargetRPS",
		},
		Series: series,
	}
	chartData.Elements = []chart.Renderable{
		chart.LegendLeft(chartData),
	}
	return chartData, nil
}

func ResponsesChart(chartTitle string, path string) (*chart.Chart, error) {
	reader, close := openCSV(path)
	defer close()
	percs := map[string]*ChartLine{
		"rps": {},
		"p50": {},
		"p95": {},
		"p99": {},
	}
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
	var series []chart.Series
	var colorIndex int
	for key, value := range percs {
		line := chart.ContinuousSeries{
			Name: key,
			Style: chart.Style{
				StrokeColor: chart.GetDefaultColor(colorIndex).WithAlpha(255),
				DotWidth:    3.0,
				StrokeWidth: 3,
			},
			XValues: value.XValues,
			YValues: value.YValues,
		}
		if key == "rps" {
			line.YAxis = chart.YAxisSecondary
		}
		series = append(series, line)
		colorIndex++
	}

	chartData := &chart.Chart{
		Title: chartTitle,
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 150,
			},
		},
		XAxis: chart.XAxis{
			Name: "Test time (Seconds)",
		},
		YAxis: chart.YAxis{
			Name: "Response time (Ms)",
		},
		YAxisSecondary: chart.YAxis{
			Name: "RPS",
		},
		Series: series,
		Width:  800,
		Height: 600,
	}
	chartData.Elements = []chart.Renderable{
		chart.LegendLeft(chartData),
	}
	return chartData, nil
}

// ReportScaling scaling chart, data must be written in csv in format:
// ${handle_name},${network_nodes},${max_rps}
func ReportScaling(inputCsv, outputPng string) {
	chartData, err := ScalingChart(inputCsv)
	if err != nil {
		log.Fatal("Couldn't read and parse requests", err)
	}
	RenderChart(chartData, outputPng)
}

func RenderChart(chartData *chart.Chart, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	err = chartData.Render(chart.PNG, file)
	if err != nil {
		log.Fatal(err)
	}
}

func openCSV(path string) (*csv.Reader, func() error) {
	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	return csv.NewReader(csvFile), csvFile.Close
}
