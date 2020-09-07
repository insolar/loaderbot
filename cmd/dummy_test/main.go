/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/insolar/loaderbot"
)

func main() {
	// go loaderbot.RunTestServer("0.0.0.0:9031")
	// nolint
	target := os.Getenv("TARGET")
	// target = "http://localhost:9031/json_body"
	for i := 0; i < 10; i++ {
		r := loaderbot.NewRunner(&loaderbot.RunnerConfig{
			TargetUrl:       target,
			Name:            fmt.Sprintf("dummy_test_%d", i),
			SystemMode:      loaderbot.OpenWorldSystem,
			Attackers:       20000,
			AttackerTimeout: 25,
			StartRPS:        10000,
			StepDurationSec: 5,
			StepRPS:         5000,
			TestTimeSec:     60,
			SuccessRatio:    0.95,
			// Prometheus:   &loaderbot.Prometheus{Enable: true},
		}, &loaderbot.FastHTTPAttackerExample{}, nil)
		_, _ = r.Run(context.TODO())
	}
}
