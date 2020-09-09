#### Loaderbot
Minimalistic load tool

#### How to use

Implement attacker
```go
package attackers

import (
	"context"
	"net/http"

	"github.com/insolar/loaderbot"
)

type AttackerExample struct {
	*loaderbot.Runner
	client *http.Client
}

func (a *AttackerExample) Clone(r *loaderbot.Runner) loaderbot.Attack {
	return &AttackerExample{Runner: r}
}

func (a *AttackerExample) Setup(c loaderbot.RunnerConfig) error {
	a.client = loaderbot.NewLoggingHTTPClient(c.DumpTransport, 10)
	return nil
}

func (a *AttackerExample) Do(_ context.Context) loaderbot.DoResult {
	_, err := a.client.Get(a.Cfg.TargetUrl)
	return loaderbot.DoResult{
		RequestLabel: a.Name,
		Error:        err.Error(),
	}
}

func (a *AttackerExample) Teardown() error {
	return nil
}
```
Run test with sync attackers when system is "closed" type, when response time increases,
 attackers may be blocked
```go
cfg := &loaderbot.RunnerConfig{
		TargetUrl:       "https://clients5.google.com/pagead/drt/dn/",
		Name:            "abc",
		SystemMode:      loaderbot.PrivateSystem,
		Attackers:       100,
		AttackerTimeout: 5,
		StartRPS:        100,
		StepDurationSec: 10,
		StepRPS:         300,
		TestTimeSec:     200,
	}
lt := loaderbot.NewRunner(cfg, &loaderbot.HTTPAttackerExample{}, nil)
maxRPS, _ := lt.Run()
```
see more [examples](examples/tests)

Config options
```go
// RunnerConfig runner configuration
type RunnerConfig struct {
	// TargetUrl target base url
	TargetUrl string
	// Name of a runner instance
	Name string
	// SystemMode PrivateSystem
	// PrivateSystem:
	// if application under test is a private system sync runner attackers will wait for response
	// in case your system is private and you know how many sync Nodes can act
	SystemMode SystemMode
	// Attackers constant amount of attackers,
	Attackers int
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
```

#### Development
```
make test
make lint
```