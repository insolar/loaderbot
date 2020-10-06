/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// NewLoggintHTTPClient creates new client with debug http
func NewLoggingHTTPClient(debug bool, transportTimeout int) *http.Client {
	var transport http.RoundTripper
	http.DefaultTransport.(*http.Transport).MaxConnsPerHost = 65535
	http.DefaultTransport.(*http.Transport).MaxIdleConns = 65535
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 65535
	http.DefaultTransport.(*http.Transport).DisableCompression = true
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Duration(transportTimeout) * time.Second
	if debug {
		transport = &DumpTransport{
			http.DefaultTransport,
		}
	} else {
		transport = http.DefaultTransport
	}
	cookieJar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(transportTimeout) * time.Second,
		Jar:       cookieJar,
	}
}

const (
	RequestHeader      = "========== REQUEST ==========\n%s\n"
	RequestHeaderBody  = "========== REQUEST ==========\n%s\n%s\n"
	ResponseHeaderBody = "========== RESPONSE ==========\n%s\n%s\n"
	ResponseHeader     = "========== RESPONSE ==========\n%s\n"
	HTTPBodyDelimiter  = "\r\n\r\n"
)

// DumpTransport log http request/responses, pprint bodies
type DumpTransport struct {
	r http.RoundTripper
}

func (d *DumpTransport) RoundTrip(h *http.Request) (*http.Response, error) {
	var respString string
	var pprintBody string
	dump, _ := httputil.DumpRequestOut(h, true)
	if bodyIsJson(h.Header) {
		req, pprintBody := d.prettyPrintJsonBody(dump)
		fmt.Printf(RequestHeaderBody, req, pprintBody)
	} else {
		fmt.Printf(RequestHeader, dump)
	}
	resp, err := d.r.RoundTrip(h)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Body != nil && bodyIsJson(resp.Header) {
		defer resp.Body.Close()
		dump, _ = httputil.DumpResponse(resp, true)
		respString, pprintBody = d.prettyPrintJsonBody(dump)
		fmt.Printf(ResponseHeaderBody, respString, pprintBody)
		return resp, err
	}
	dump, _ = httputil.DumpResponse(resp, true)
	fmt.Printf(ResponseHeader, dump)
	return resp, err
}

// prettyPrintJsonBody returns http format request and pretty printed json body
func (d *DumpTransport) prettyPrintJsonBody(b []byte) (string, string) {
	var pprintBody []byte
	s := string(b)
	sp := strings.Split(s, HTTPBodyDelimiter)
	if len(sp) == 2 {
		body := sp[1]
		if strings.HasPrefix(body, "[") {
			var objmap []*jsoniter.RawMessage
			err := jsoniter.Unmarshal([]byte(body), &objmap)
			if err != nil {
				log.Fatal(err)
			}

			pprintBody, err = jsoniter.MarshalIndent(objmap, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			return sp[0], string(pprintBody)
		}
		var objmap map[string]*jsoniter.RawMessage
		err := jsoniter.Unmarshal([]byte(body), &objmap)
		if err != nil {
			log.Fatal(err)
		}
		pprintBody, err = jsoniter.MarshalIndent(objmap, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
	}
	return sp[0], string(pprintBody)
}

func bodyIsJson(h http.Header) bool {
	return strings.Contains(h.Get("content-type"), "application/json")
}
