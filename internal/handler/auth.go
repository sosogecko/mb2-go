package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mb2-go/internal/i18n"
	"mb2-go/internal/model"
	"mb2-go/internal/util"
	"mb2-go/internal/web"
)

type AuthHandler struct{ *web.App }

func (h *AuthHandler) Handle(c *gin.Context, action string) {
	switch strings.ToLower(action) {
	case "login":
		h.Login(c)
	case "findpwd":
		h.FindPwd(c)
	case "register":
		h.Register(c)
	default:
		c.Status(404)
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	local := web.Locale(c)
	email := util.Post(c, "email")
	password := util.Post(c, "password")
	md5pwd := util.MD5String(password)
	sid := web.SID(c)
	ip := util.ClientIP(c)

	actionName := "login_" + util.Post(c, "client_type") + util.Get(c, "client_type")
	if err := h.DB.Exec("insert into user_action_log(user_email, action, ip, time) values(?,?,?,?)",
		email, actionName, ip, time.Now().Unix()).Error; err != nil {
		h.Log.App.Error("login action log failed", zap.Error(err), zap.String("email", email))
	}

	row, err := h.QueryMap(`select user_email,vip,expire from gecko.user_profile
		where user_email = ? and actively = 1 and password = ?
		and user_email not in (select value from gecko.deny where type = 1)`, email, md5pwd)
	if err != nil {
		h.Log.App.Error("login query profile failed", zap.Error(err), zap.String("email", email))
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	if row == nil {
		web.JSON(c, gin.H{"err": 1, "data": i18n.Lang("invalidAccount", local)})
		return
	}

	expire := util.StringVal(util.MapGetCI(row, "expire"))
	expireSec := util.ParseExpire(expire, h.Loc)

	var userCache map[string]int64
	_ = h.Cache.Get(email, &userCache)
	if userCache == nil {
		userCache = map[string]int64{}
	}
	now := time.Now().Unix()
	for k, v := range userCache {
		if now-v > 600 {
			delete(userCache, k)
		}
	}
	userCache[sid] = now
	_ = h.Cache.Set(email, userCache, 24*time.Hour)

	var panel model.PanelUser
	err = h.DB.Where("email = ?", email).Take(&panel).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.Log.App.Error("login query user failed", zap.Error(err), zap.String("email", email))
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		panel = model.PanelUser{
			Username:  email,
			Email:     email,
			Password:  md5pwd,
			Passwd:    "fs9527",
			Method:    h.AppCfg.Get("DEFAULT_METHOD"),
			Protocol:  h.AppCfg.Get("DEFAULT_PROTOCOL"),
			Obfs:      h.AppCfg.Get("DEFAULT_OBFS"),
			InvitedBy: 0,
			Money:     0,
			Plan:      h.AppCfg.Get("DEFAULT_GROUP"),
			// PlanEndTime:    time.Now().Unix() + int64(h.AppCfg.Int("DEFAULT_EXPIRATION", 86400)),
			TransferEnable: 0,
			InviteCodeAmt:  int64(h.AppCfg.Int("DEFAULT_INVITE_AMOUNT", 2)),
			Port:           999,
			RebateID:       util.CRC32Uniq(),
			RebateFirst:    h.AppCfg.Get("DEFAULT_REBATE_FIRST"),
			RebateCharge:   h.AppCfg.Get("DEFAULT_REBATE_CHARGE"),
		}
		if err := h.DB.Create(&panel).Error; err != nil {
			h.Log.App.Error("login create user failed", zap.Error(err), zap.String("email", email))
			web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
			return
		}
	}

	sess := &model.UserSession{
		UserEmail:       util.StringVal(util.MapGetCI(row, "user_email")),
		VIP:             util.MapGetCI(row, "vip"),
		Expire:          expire,
		ExpireSenconds:  expireSec,
		Passwd:          panel.Passwd,
		Port:            panel.Port,
		Method:          panel.Method,
		Protocol:        panel.Protocol,
		Pbfs:            panel.Obfs,
		UID:             panel.UID,
		Username:        panel.Username,
		Email:           panel.Email,
		LastCheckInTime: panel.LastCheckInTime,
		Connections:     panel.Connections,
	}
	if err := web.SaveUser(h.App, sess); err != nil {
		h.Log.App.Error("login save session failed", zap.Error(err), zap.Int64("uid", panel.UID))
	}

	token := util.PHPMd5Token(email)
	expDays := h.AppCfg.Int("API_TOKEN_EXP", 7)
	if err := h.DB.Exec("delete from user_token where uid = ? and expiration < ?", panel.UID, time.Now().Unix()).Error; err != nil {
		h.Log.App.Error("login delete expired token failed", zap.Error(err), zap.Int64("uid", panel.UID))
	}
	if err := h.DB.Exec("insert into user_token(uid, token, expiration) values(?,?,?)",
		panel.UID, token, 86400*int64(expDays)+time.Now().Unix()).Error; err != nil {
		h.Log.App.Error("login insert token failed", zap.Error(err), zap.Int64("uid", panel.UID))
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	web.JSON(c, gin.H{"err": 0, "data": gin.H{"uid": panel.UID, "token": token}})
}

func (h *AuthHandler) FindPwd(c *gin.Context) {
	email := util.Post(c, "email")
	local := web.Locale(c)
	host := strings.TrimRight(h.AppCfg.Get("HOST_API"), "/")
	reqURL := host + "?action=SEND_PWD_EMAIL&userEmail=" + url.QueryEscape(email)
	html, err := httpGet(reqURL)
	if err != nil {
		h.Log.App.Error("findPwd host api failed", zap.Error(err), zap.String("url", reqURL))
		web.JSON(c, gin.H{"err": 0, "html": "", "data": "服务器异常，请稍后再试！"})
		return
	}
	if !strings.HasPrefix(html, "{") {
		web.JSON(c, gin.H{"err": 0, "html": html, "data": "服务器异常，请稍后再试！"})
		return
	}
	var rd map[string]any
	if err := json.Unmarshal([]byte(html), &rd); err != nil {
		web.JSON(c, gin.H{"err": 0, "html": html, "data": "服务器异常，请稍后再试！"})
		return
	}
	status := util.StringVal(rd["status"])
	message := util.StringVal(rd["message"])
	if status == "500" {
		switch message {
		case "NotExsitUserEmail":
			web.JSON(c, gin.H{"err": 1, "data": "您输入得邮箱不存在！"})
		case "SendEmailFaild":
			web.JSON(c, gin.H{"err": 1, "data": "重置密码邮箱发送失败！"})
		default:
			web.JSON(c, gin.H{"err": 1, "data": i18n.Lang("serverException", local)})
		}
		return
	}
	if status == "200" {
		web.JSON(c, gin.H{"err": 0, "data": "重置密码链接已发送到您的邮箱！"})
		return
	}
	web.JSON(c, gin.H{"err": 0, "html": html, "data": "服务器异常，请稍后再试！"})
}

func (h *AuthHandler) Register(c *gin.Context) {
	local := web.Locale(c)
	email := util.Post(c, "email")
	password := util.Post(c, "password")
	md5pwd := util.MD5String(password)
	ip := util.ClientIP(c)

	if err := h.DB.Exec("insert into user_action_log(user_email, action, ip, time) values(?,?,?,?)",
		email, "register", ip, time.Now().Unix()).Error; err != nil {
		h.Log.App.Error("register action log failed", zap.Error(err), zap.String("email", email))
	}

	host := strings.TrimRight(h.AppCfg.Get("HOST_API"), "/")
	reqURL := host + "?action=REGISTER_ACTION&userEmail=" + url.QueryEscape(email) + "&password=" + url.QueryEscape(md5pwd) + "&ip=" + url.QueryEscape(ip)
	h.Log.App.Info("register request url", zap.String("url", reqURL))
	html, err := httpGet(reqURL)
	if err != nil {
		h.Log.App.Error("register host api failed", zap.Error(err), zap.String("url", reqURL))
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	if !strings.HasPrefix(html, "{") {
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	var rd map[string]any
	if err := json.Unmarshal([]byte(html), &rd); err != nil {
		web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		return
	}
	status := util.StringVal(rd["status"])
	message := util.StringVal(rd["message"])
	if status == "500" {
		switch message {
		case "InvalidUserEmail":
			web.JSON(c, gin.H{"err": 1, "data": i18n.Lang("errorFormatEmail", local)})
		case "InvalidPassword":
			web.JSON(c, gin.H{"err": 1, "data": i18n.Lang("inputPwd", local)})
		case "ExsitUserEmail":
			web.JSON(c, gin.H{"err": 1, "data": i18n.Lang("existUserEmail", local)})
		default:
			web.JSON(c, gin.H{"err": 2, "data": i18n.Lang("serverException", local)})
		}
		return
	}
	if status == "200" {
		web.JSON(c, gin.H{"err": 0, "data": fmt.Sprintf(i18n.Lang("sendEmail", local), email)})
		return
	}
	web.JSON(c, gin.H{"err": 0, "data": fmt.Sprintf(i18n.Lang("registerOK", local), "")})
}

func httpGet(rawURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/html; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
