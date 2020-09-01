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

	"github.com/gin-gonic/gin"
)

// nolint
func RunTestServer(target string) string {
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
	r.Run(target)
	return target
}
