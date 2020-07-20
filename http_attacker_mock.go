package loaderbot

import (
	"context"
	"net/http"
)

type HTTPAttackerExample struct {
	*Runner
	client *http.Client
}

func (a *HTTPAttackerExample) Clone(r *Runner) Attack {
	return &HTTPAttackerExample{Runner: r}
}

func (a *HTTPAttackerExample) Setup(c RunnerConfig) error {
	a.client = NewLoggingHTTPClient(c.DumpTransport, 10)
	return nil
}

func (a *HTTPAttackerExample) Do(_ context.Context) DoResult {
	_, err := a.client.Get(a.Cfg.TargetUrl)
	return DoResult{
		RequestLabel: a.Name,
		Error:        err,
	}
}

func (a *HTTPAttackerExample) Teardown() error {
	return nil
}
