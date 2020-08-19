/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"log"

	"github.com/valyala/fasthttp"
)

type FastHTTPClient struct {
	dump bool
	fasthttp.Client
}

// NewLoggingFastHTTPClient creates new client with debug http
func NewLoggingFastHTTPClient(debug bool) *FastHTTPClient {
	return &FastHTTPClient{
		debug,
		fasthttp.Client{},
	}
}

func (m *FastHTTPClient) Do(req *fasthttp.Request) (int, []byte, error) {
	if m.dump {
		log.Printf(RequestHeader, req.String())
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	if err := m.Client.Do(req, resp); err != nil {
		return -1, nil, err
	}
	if m.dump {
		log.Printf(ResponseHeader, resp.String())
	}
	return resp.StatusCode(), resp.Body(), nil
}
