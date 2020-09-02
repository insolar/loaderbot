/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommonReportScaling(t *testing.T) {
	ReportScaling("example_csv_data/scaling.csv", "scaling.html")
}

func TestCommonReportScalingSlack(t *testing.T) {
	ReportScalingSlack("example_csv_data/scaling.csv", "scaling.png")
}

func TestCommonRenderPercs(t *testing.T) {
	data, err := PercsChart("example_csv_data/percs.csv", "Response times")
	if err != nil {
		log.Fatal(err)
	}
	RenderEChart(data, "responses.html")
}

func TestCommonRenderErr(t *testing.T) {
	_, err := PercsChart("example_csv_data/empty.csv", "Response times")
	require.Error(t, err)
}
