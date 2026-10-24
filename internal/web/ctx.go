package web

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mb2-go/internal/appcfg"
	"mb2-go/internal/cache"
	"mb2-go/internal/config"
	"mb2-go/internal/i18n"
	"mb2-go/internal/logger"
	"mb2-go/internal/model"
	"mb2-go/internal/util"
)

type App struct {
	Cfg    *config.Config
	DB     *gorm.DB
	Cache  *cache.FileCache
	AppCfg *appcfg.Store
	Log    *logger.Loggers
	Loc    *time.Location
}

func JSON(c *gin.Context, payload any) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Abort()
	c.JSON(http.StatusOK, payload)
}

func Empty(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Abort()
}

func Locale(c *gin.Context) string {
	return util.Get(c, "locale")
}

func T(c *gin.Context, msg string) string {
	return i18n.Lang(msg, Locale(c))
}

func SID(c *gin.Context) string {
	return util.Get(c, "sid")
}

func UserCacheKey(uid any) string {
	return "suidao_api_index_" + util.StringVal(uid)
}

func LoadUser(a *App, uid string) (*model.UserSession, bool) {
	var u model.UserSession
	if !a.Cache.Get(UserCacheKey(uid), &u) {
		return nil, false
	}
	if u.UID == 0 && uid != "" {
		return nil, false
	}
	return &u, true
}

func SaveUser(a *App, u *model.UserSession) error {
	return a.Cache.Set(UserCacheKey(u.UID), u, 24*time.Hour)
}

type bodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func AccessLog(app *App) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("request_id", util.MD5String(time.Now().String() + c.Request.RequestURI + util.ClientIP(c))[:16])
		start := time.Now()
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}
		bw := &bodyWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = bw

		c.Next()

		latency := time.Since(start)
		errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String()
		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("ip", util.ClientIP(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("full_url", c.Request.URL.String()),
			zap.String("request_uri", c.Request.RequestURI),
			zap.Any("headers", util.HeaderMap(c)),
			zap.String("request_body", util.Truncate(string(reqBody))),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("response_body", util.Truncate(bw.buf.String())),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if errMsg != "" {
			fields = append(fields, zap.String("gin_error", errMsg))
			app.Log.App.Error("request error", fields...)
		}
		app.Log.Access.Info("request", fields...)
	}
}

func Recovery(app *App) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				app.Log.App.Error("panic",
					zap.Any("recover", rec),
					zap.String("path", c.Request.URL.Path),
					zap.String("request_uri", c.Request.RequestURI),
					zap.String("ip", util.ClientIP(c)),
				)
				JSON(c, gin.H{"err": 2, "data": T(c, "serverException")})
			}
		}()
		c.Next()
	}
}

func ParseForm(c *gin.Context) {
	_ = c.Request.ParseForm()
	_ = c.Request.ParseMultipartForm(32 << 20)
}
