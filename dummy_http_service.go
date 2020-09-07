/*
 * // Copyright 2020 Insolar Network Ltd.
 * // All rights reserved.
 * // This material is licensed under the Insolar License version 1.0,
 * // available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package loaderbot

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// nolint
func RunTestServer(target string, sleep time.Duration) *http.Server {
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
		time.Sleep(sleep)
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/html", func(c *gin.Context) {
		time.Sleep(sleep)
		c.HTML(http.StatusOK, "html1", nil)
	})
	srv := &http.Server{
		Addr:    target,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf(err.Error())
		}
	}()
	return srv
}
