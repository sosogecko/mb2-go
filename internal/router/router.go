package router

import (
	"github.com/gin-gonic/gin"

	"mb2-go/internal/handler"
	"mb2-go/internal/web"
)

func New(app *web.App) *gin.Engine {
	if app.Cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(web.AccessLog(app), web.Recovery(app))
	r.NoRoute(handler.Dispatch(app))
	r.Any("/", handler.Dispatch(app))
	r.Any("/index.php", handler.Dispatch(app))
	r.Any("/index.php/*any", handler.Dispatch(app))
	r.Any("/Api/*any", handler.Dispatch(app))
	r.Any("/api/*any", handler.Dispatch(app))
	return r
}
