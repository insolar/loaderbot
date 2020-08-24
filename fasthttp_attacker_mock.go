/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"context"

	"github.com/valyala/fasthttp"
)

type FastHTTPAttackerExample struct {
	*Runner
}

func (a *FastHTTPAttackerExample) Clone(r *Runner) Attack {
	return &FastHTTPAttackerExample{Runner: r}
}

func (a *FastHTTPAttackerExample) Setup(c RunnerConfig) error {
	return nil
}

func (a *FastHTTPAttackerExample) Do(_ context.Context) DoResult {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(a.Cfg.TargetUrl)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	err := a.FastHTTPClient.Do(req, resp)
	if resp.StatusCode() >= 400 {
		return DoResult{
			Error: "request failed",
		}
	}
	if err != nil {
		return DoResult{
			Error: err.Error(),
		}
	}
	return DoResult{RequestLabel: a.Name}
}

func (a *FastHTTPAttackerExample) Teardown() error {
	return nil
}
