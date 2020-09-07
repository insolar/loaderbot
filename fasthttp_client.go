/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"log"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"

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
		fasthttp.Client{
			// Dial: func(addr string) (net.Conn, error) {
			// 	return fasthttp.DialTimeout(addr, 1*time.Second)
			// },
			MaxConnsPerHost:           65535,
			MaxIdleConnDuration:       90 * time.Second,
			MaxIdemponentCallAttempts: 0,
		},
	}
}

func (m *FastHTTPClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	if m.dump {
		log.Printf(RequestHeader, req.String())
	}
	if err := m.Client.DoRedirects(req, resp, 5); err != nil {
		return err
	}
	if m.dump {
		log.Printf(ResponseHeader, resp.String())
	}
	return nil
}

func UnmarshalAnyJson(d []byte, typ interface{}) (interface{}, error) {
	if typ == nil || d == nil {
		return nil, nil
	}
	t := reflect.TypeOf(typ).Elem()
	v := reflect.New(t)
	newP := v.Interface()
	if err := jsoniter.Unmarshal(d, newP); err != nil {
		return nil, err
	}
	return newP, nil
}
