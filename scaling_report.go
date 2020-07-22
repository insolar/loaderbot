/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"encoding/csv"
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

func ReadCsvFile(path string) (map[string]ChartLine, error) {
	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	requests := make(map[string]ChartLine)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) != 3 {
			return nil, err
		}

		methodName := record[0]
		veCount := record[1]
		maxRPS := record[2]

		line := requests[methodName]
		xValue, err := strconv.ParseFloat(veCount, 64)
		if err != nil {
			return nil, err
		}
		// VE count
		line.XValues = append(line.XValues, xValue)
		yValue, err := strconv.ParseFloat(maxRPS, 64)
		if err != nil {
			return nil, err
		}
		// Max RPS
		line.YValues = append(line.YValues, yValue)
		requests[methodName] = line
	}
	return requests, nil
}

// ReportScaling scaling chart, data must be written in csv in format:
// ${handle_name},${network_nodes},${max_rps}
func ReportScaling(inputCsv, outputPng string) {
	lines, err := ReadCsvFile(inputCsv)
	if err != nil {
		log.Fatal("Couldn't read and parse requests", err)
	}

	err = RenderChart(lines, outputPng)
	if err != nil {
		log.Fatal("Couldn't render chart", err)
	}
}

func RenderChart(requests map[string]ChartLine, fileName string) error {
	var series []chart.Series
	var colorIndex int
	for key, value := range requests {
		series = append(series, chart.ContinuousSeries{
			Name: key,
			Style: chart.Style{
				StrokeColor: chart.GetDefaultColor(colorIndex).WithAlpha(255),
				StrokeWidth: 5,
			},
			XValues: value.XValues,
			YValues: value.YValues,
		})
		colorIndex++
	}

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Name: "VE count",
		},
		YAxis: chart.YAxis{
			Name: "Max RPS",
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	err = graph.Render(chart.PNG, file)
	if err != nil {
		log.Fatal(err)
	}
	return graph.Render(chart.PNG, file)
}
