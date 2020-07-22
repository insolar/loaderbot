/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

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
