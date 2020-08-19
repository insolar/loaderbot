/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"encoding/json"
	"log"
	"reflect"

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

func (m *FastHTTPClient) Do(req *fasthttp.Request, respStruct interface{}) (int, interface{}, error) {
	var respStruct2 interface{}
	if m.dump {
		log.Printf(RequestHeader, req.String())
	}
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	if err := m.Client.Do(req, resp); err != nil {
		return -1, nil, err
	}
	if respStruct != nil {
		respStruct2 = UnmarshalAny(resp.Body(), respStruct)
	}
	if m.dump {
		log.Printf(ResponseHeader, resp.String())
	}
	return resp.StatusCode(), respStruct2, nil
}

func UnmarshalAny(d []byte, typ interface{}) interface{} {
	t := reflect.TypeOf(typ).Elem()
	v := reflect.New(t)
	newP := v.Interface()
	if err := json.Unmarshal(d, newP); err != nil {
		log.Fatal(err)
	}
	return newP
}
