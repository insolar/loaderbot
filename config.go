/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

//go:generate stringer -type=SystemMode
package loaderbot

import (
	"log"
	"os"
)

type SystemMode int

const (
	BoundRPS SystemMode = iota
	UnboundRPS
	BoundRPSAutoscale
)

// RunnerConfig runner configuration
type RunnerConfig struct {
	// TargetUrl target base url
	TargetUrl string
	// Name of a runner instance
	Name string
	// InstanceType attacker type instance, used only in cluster mode
	InstanceType string
	// SystemMode BoundRPS
	// BoundRPS:
	// if application under test is a private system sync runner attackers will wait for response
	// in case your system is private and you know how many sync clients can act
	// BoundRPSAutoscale:
	// try to scale attackers when all attackers are blocked
	// UnboundRPS:
	// attack as fast as we can with N attackers
	SystemMode SystemMode
	// Attackers constant amount of attackers,
	Attackers int
	// AttackersScaleFactor how much attackers to add when rps is not met, default is 100
	AttackersScaleAmount int
	// AttackersScaleThreshold scale if current rate is less than target rate * threshold,
	// interval of values = [0, 1], default is 0.90
	AttackersScaleThreshold float64
	// AttackerTimeout timeout of attacker
	AttackerTimeout int
	// StartRPS initial requests per seconds rate
	StartRPS int
	// StepDurationSec duration of step in which rps is increased by StepRPS
	StepDurationSec int
	// StepRPS amount of requests per second which will be added in next step,
	// if StepRPS = 0 rate is constant, default StepDurationSec is 30 sec is applied,
	// just to keep 30s aggregation metrics
	StepRPS int
	// TestTimeSec test timeout
	TestTimeSec int
	// WaitBeforeSec time to wait before start in case we didn't know start criteria
	WaitBeforeSec int
	// Dumptransport dumps http requests to stdout
	DumpTransport bool
	// GoroutinesDump dumps goroutines stack for debug purposes
	GoroutinesDump bool
	// SuccessRatio to fail when below
	SuccessRatio float64
	// Metadata all other data required for test setup
	Metadata map[string]interface{}
	// LogLevel debug|info, etc.
	LogLevel string
	// LogEncoding json|console
	LogEncoding string
	// Reporting options, csv/png/stream
	ReportOptions *ReportOptions
	// ClusterOptions
	ClusterOptions *ClusterOptions
	// Prometheus config
	Prometheus *Prometheus
}

type Prometheus struct {
	Enable bool
	Port   int
}

type ClusterOptions struct {
	Nodes []string
}

// ReportOptions reporting options
type ReportOptions struct {
	// Report html directory
	HTMLDir string
	// Report csv directory
	CSVDir string
	// CSV dumps requests/responses data
	CSV bool
	// PNG creates percentiles graph
	PNG bool
	// Stream streams raw and tick aggregated data back to client in cluster mode
	Stream bool
}

func (c *RunnerConfig) Validate() {
	errors := c.validate()
	if len(errors) > 0 {
		for _, e := range errors {
			log.Print(e)
		}
		os.Exit(1)
	}
}

func (c *RunnerConfig) DefaultCfgValues() {
	if c.SystemMode == BoundRPS && c.StartRPS == 0 {
		c.StartRPS = 10
	}
	// constant load
	if c.SystemMode == BoundRPS && c.StepRPS == 0 {
		c.StepDurationSec = 10
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.LogEncoding == "" {
		c.LogEncoding = "console"
	}
	if c.ReportOptions == nil {
		c.ReportOptions = &ReportOptions{
			CSVDir:  "results_csv",
			HTMLDir: "results_html",
			CSV:     true,
			PNG:     true,
		}
	}
	if c.ReportOptions.CSVDir == "" {
		c.ReportOptions.CSVDir = "results_csv"
	}
	if c.ReportOptions.HTMLDir == "" {
		c.ReportOptions.HTMLDir = "results_html"
	}
	if c.Prometheus != nil && c.Prometheus.Port == 0 {
		c.Prometheus.Port = 2112
	}
	if c.SystemMode == BoundRPSAutoscale {
		if c.AttackersScaleAmount == 0 {
			c.AttackersScaleAmount = 100
		}
		if c.AttackersScaleThreshold == 0 {
			c.AttackersScaleThreshold = 0.9
		}
	}
}

// Validate checks all settings and returns a list of strings with problems.
func (c RunnerConfig) validate() (list []string) {
	if c.Name == "" {
		list = append(list, "please set runner name")
	}
	if c.Attackers <= 0 && c.SystemMode == BoundRPS {
		list = append(list, "please set attackers > 0")
	}
	if c.AttackerTimeout <= 0 {
		list = append(list, "please set attacker timeout > 0, seconds")
	}
	if c.StepDurationSec < 0 {
		list = append(list, "please set step duration > 0, seconds")
	}
	if c.SystemMode == BoundRPS && c.StepRPS < 0 {
		list = append(list, "please set step rps > 0")
	}
	if c.TestTimeSec <= 0 {
		list = append(list, "please set test time rps > 0, seconds")
	}
	return
}
