package attackers

import (
	"context"
	"net/http"

	"github.com/insolar/loaderbot"
)

func init() {
	loaderbot.RegisterAttacker("HTTPAttackerExample", &AttackerExample{})
}

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
	if err != nil {
		return loaderbot.DoResult{
			RequestLabel: a.Name,
			Error:        err.Error(),
		}
	}
	return loaderbot.DoResult{
		RequestLabel: a.Name,
	}
}

func (a *AttackerExample) Teardown() error {
	return nil
}
