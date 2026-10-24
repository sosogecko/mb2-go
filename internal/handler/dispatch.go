package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mb2-go/internal/web"
)

func Dispatch(app *web.App) gin.HandlerFunc {
	index := &IndexHandler{App: app}
	user := &UserHandler{App: app}
	auth := &AuthHandler{App: app}

	return func(c *gin.Context) {
		web.ParseForm(c)
		module, controller, action, extra := parseThinkPHP(c)
		for k, v := range extra {
			if c.Query(k) == "" {
				q := c.Request.URL.Query()
				q.Set(k, v)
				c.Request.URL.RawQuery = q.Encode()
			}
		}
		if module != "" && !strings.EqualFold(module, "api") {
			c.Status(404)
			return
		}
		switch strings.ToLower(controller) {
		case "index":
			index.Handle(c, action)
		case "user":
			user.Handle(c, action)
		case "auth":
			auth.Handle(c, action)
		default:
			c.Status(404)
		}
	}
}

func parseThinkPHP(c *gin.Context) (module, controller, action string, extra map[string]string) {
	raw := c.Query("s")
	if raw == "" {
		raw = c.Request.URL.Path
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	lower := strings.ToLower(raw)
	if i := strings.Index(lower, "index.php"); i >= 0 {
		raw = raw[i+len("index.php"):]
	}
	raw = strings.Trim(raw, "/")
	parts := []string{}
	for _, p := range strings.Split(raw, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	extra = map[string]string{}
	if len(parts) > 0 {
		module = parts[0]
	}
	if len(parts) > 1 {
		controller = parts[1]
	}
	if len(parts) > 2 {
		action = parts[2]
	}
	for i := 3; i+1 < len(parts); i += 2 {
		extra[parts[i]] = parts[i+1]
	}
	return
}
