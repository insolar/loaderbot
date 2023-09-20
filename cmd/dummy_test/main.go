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
			SystemMode:      loaderbot.BoundRPS,
			Attackers:       1000,
			StartRPS:        20000,
			StepRPS:         5000,
			StepDurationSec: 3,
			AttackerTimeout: 25,
			TestTimeSec:     20,
			SuccessRatio:    0.95,
		}, &loaderbot.HTTPAttackerExample{}, nil)
		_, _ = r.Run(context.TODO())
	}
}
