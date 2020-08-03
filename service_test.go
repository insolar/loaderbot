package loaderbot

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"log"
	"testing"
	"time"
)

const (
	address = "localhost:50051"
)

func TestServiceConnect(t *testing.T) {
	go RunService(address)
	time.Sleep(1 * time.Second)
	c, closer := NewNodeClient(address)
	defer closer()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r, err := c.Run(ctx, &RunConfigRequest{
		Config: MarshalConfigGob(RunnerConfig{
			TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
			Name:            "test_runner",
			SystemMode:      OpenWorldSystem,
			Attackers:       1,
			AttackerTimeout: 1,
			StartRPS:        100,
			StepDurationSec: 5,
			StepRPS:         200,
			TestTimeSec:     20,
			ReportOptions: &ReportOptions{
				Stream: true,
			},
		}),
		AttackerName: "http",
	})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	for {
		res, err := r.Recv()
		if res == nil {
			return
		}
		if err == io.EOF {
			return
		}
		b := bytes.NewBuffer(res.ResultsChunk)
		dec := gob.NewDecoder(b)
		var results []AttackResult
		if err := dec.Decode(&results); err != nil {
			log.Fatal(err)
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Data: %s", results)
	}
}
