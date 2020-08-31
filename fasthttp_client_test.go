/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"html/template"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// nolint
func runTestServer() string {
	target := "0.0.0.0:9031"
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	html := template.Must(template.New("html1").Parse(`
<html>
<head>
  <title>Test</title>
</head>
<body>
</body>
</html>
`))
	r.SetHTMLTemplate(html)
	r.GET("/json_body", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/html", func(c *gin.Context) {
		c.HTML(http.StatusOK, "html1", nil)
	})
	// nolint
	go r.Run(target)
	return target
}

// nolint
type MsgStruct struct {
	Message string
}

func TestManualFastHttpMarshal(t *testing.T) {
	target := runTestServer()
	c := NewLoggingFastHTTPClient(true)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://" + target + "/json_body")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	err := c.Do(req, resp)
	require.NoError(t, err)
	respBody, err := UnmarshalAnyJson(resp.Body(), &MsgStruct{})
	require.NoError(t, err)
	require.Equal(t, &MsgStruct{Message: "pong"}, respBody)
}

func TestManualFastHttpHtml(t *testing.T) {
	target := runTestServer()
	c := NewLoggingFastHTTPClient(true)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://" + target + "/html")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	err := c.Do(req, resp)
	require.Nil(t, resp)
	require.NoError(t, err)
}
