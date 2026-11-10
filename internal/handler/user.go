package handler

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mb2-go/internal/model"
	"mb2-go/internal/paymentwall"
	"mb2-go/internal/util"
	"mb2-go/internal/web"
	"mb2-go/internal/xxtea"
)

type UserHandler struct{ *web.App }

func (h *UserHandler) Handle(c *gin.Context, action string) {
	if strings.Contains(strings.ToLower(c.Request.RequestURI), "logout") {
		h.Logout(c)
		return
	}
	user, ok := h.requireUser(c)
	if !ok {
		return
	}
	c.Set("user", user)
	switch strings.ToLower(action) {
	case "logout":
		h.Logout(c)
	case "ping":
		h.Ping(c, user)
	case "getpaymentwallurl":
		h.GetPaymentWallURL(c, user)
	case "getreplylist":
		h.GetReplyList(c, user)
	case "getallnodelist":
		h.GetAllNodeList(c, user)
	case "getandroidnodelist":
		h.GetAndroidNodeList(c, user)
	case "getuserinfo":
		h.GetUserInfo(c, user)
	case "modifypwd":
		h.ModifyPwd(c, user)
	case "checkversion":
		h.CheckVersion(c)
	case "getnoticelist":
		h.GetNoticeList(c)
	case "feedback":
		h.Feedback(c, user)
	case "recordclientlogin":
		h.RecordClientLogin(c, user)
	default:
		c.Status(404)
	}
}

func (h *UserHandler) requireUser(c *gin.Context) (*model.UserSession, bool) {
	uid := util.Post(c, "uid")
	token := util.Post(c, "token")
	var count int64
	if err := h.DB.Table("user_token").Where("uid = ? AND token = ?", uid, token).Count(&count).Error; err != nil {
		h.Log.App.Error("user_token query failed", zap.Error(err), zap.String("uid", uid))
		web.JSON(c, gin.H{"err": -1, "data": c.Request.RequestURI})
		return nil, false
	}
	if count == 0 {
		web.JSON(c, gin.H{"err": -1, "data": c.Request.RequestURI})
		return nil, false
	}
	user, ok := web.LoadUser(h.App, uid)
	if !ok {
		web.JSON(c, gin.H{"err": -1, "data": web.T(c, "timeout")})
		return nil, false
	}
	c.Set("token", token)
	return user, true
}

func (h *UserHandler) Logout(c *gin.Context) {
	uid := util.Post(c, "uid")
	token := util.Post(c, "token")
	sid := web.SID(c)
	if user, ok := web.LoadUser(h.App, uid); ok && user.Email != "" {
		var userCache map[string]int64
		if h.Cache.Get(user.Email, &userCache) && userCache != nil {
			delete(userCache, sid)
			_ = h.Cache.Set(user.Email, userCache, 24*time.Hour)
		}
	}
	if uid != "" && token != "" {
		if err := h.DB.Exec("delete from user_token where uid = ? and token = ?", uid, token).Error; err != nil {
			h.Log.App.Error("logout delete token failed", zap.Error(err), zap.String("uid", uid))
		}
	}
	web.JSON(c, gin.H{"err": 0})
}

func (h *UserHandler) Ping(c *gin.Context, user *model.UserSession) {
	sid := web.SID(c)
	var userCache map[string]int64
	_ = h.Cache.Get(user.Email, &userCache)
	if userCache == nil {
		userCache = map[string]int64{}
	}
	now := time.Now().Unix()
	for k, v := range userCache {
		if now-v > 900 {
			delete(userCache, k)
		}
	}
	userCache[sid] = now
	if err := h.Cache.Set(user.Email, userCache, 24*time.Hour); err != nil {
		h.Log.App.Error("ping cache failed", zap.Error(err), zap.String("email", user.Email))
	}
	web.JSON(c, gin.H{"err": 0})
}

func (h *UserHandler) GetPaymentWallURL(c *gin.Context, user *model.UserSession) {
	goodsID := util.Post(c, "goodsId")
	row, err := h.QueryMap("select iid,title,prices from shop_item where iid = ? limit 1", goodsID)
	if err != nil {
		h.Log.App.Error("getPaymentWallURL query failed", zap.Error(err), zap.String("goodsId", goodsID))
		web.JSON(c, gin.H{"err": 1, "data": "未知商品"})
		return
	}
	if row == nil {
		web.JSON(c, gin.H{"err": 1, "data": "未知商品"})
		return
	}
	price, _ := strconv.ParseFloat(util.StringVal(util.MapGetCI(row, "prices")), 64)
	w := paymentwall.Widget{
		PublicKey:  h.Cfg.Paymentwall.PublicKey,
		PrivateKey: h.Cfg.Paymentwall.PrivateKey,
		UserID:     user.Email,
		WidgetCode: h.Cfg.Paymentwall.Widget,
		Product: paymentwall.Product{
			ID:           goodsID,
			Amount:       price,
			CurrencyCode: "USD",
			Name:         util.StringVal(util.MapGetCI(row, "title")),
			Type:         "fixed",
		},
		Extra: map[string]string{
			"email":                      user.Email,
			"history[registration_date]": util.StringVal(user.CreateTime),
			"history[registration_ip]":   user.IP,
			"timestamp":                  strconv.FormatInt(time.Now().Unix(), 10),
			"ps":                         "all",
		},
	}
	web.JSON(c, gin.H{"err": 0, "data": w.URL()})
}

func (h *UserHandler) GetReplyList(c *gin.Context, user *model.UserSession) {
	isAll := c.Query("isAll") == "1"
	query := "SELECT * FROM gecko.`comment` WHERE ask_user = ? AND readed = 0 AND answer_content is not NULL order BY ask_time DESC"
	if isAll {
		query = "SELECT * FROM gecko.`comment` WHERE ask_user = ? order BY ask_time DESC"
	}
	list, err := h.QueryMaps(query, user.Email)
	if err != nil {
		h.Log.App.Error("getReplyList query failed", zap.Error(err), zap.String("email", user.Email))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	comments := make([]map[string]any, 0, len(list))
	for _, row := range list {
		comments = append(comments, map[string]any{
			"id":           util.MapGetCI(row, "commentid"),
			"question":     util.MapGetCI(row, "ask_content"),
			"questionTime": util.MapGetCI(row, "ask_time"),
			"userEmail":    util.MapGetCI(row, "ask_user"),
			"reply":        util.MapGetCI(row, "answer_content"),
			"replyTime":    util.MapGetCI(row, "answer_time"),
		})
	}
	if !isAll {
		if err := h.DB.Exec("update gecko.`comment` set readed = 1 where ask_user = ? and readed = 0", user.Email).Error; err != nil {
			h.Log.App.Error("getReplyList mark read failed", zap.Error(err), zap.String("email", user.Email))
		}
	}
	web.JSON(c, gin.H{"err": 0, "data": comments})
}

func (h *UserHandler) GetAllNodeList(c *gin.Context, user *model.UserSession) {
	token := c.GetString("token")
	from := util.Get(c, "client_type")
	vip := user.ExpireSenconds > time.Now().Unix()
	if from == "WIN" || from == "MAC" {
		list, err := h.QueryMaps("select * from desktop_node where status = 1 order by `level`")
		if err != nil {
			h.Log.App.Error("getAllNodeList desktop_node failed", zap.Error(err))
			web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
			return
		}
		result := make([]map[string]any, 0, len(list))
		for _, n := range list {
			if util.StringVal(util.MapGetCI(n, "use_for")) == "android" {
				continue
			}
			if !vip && util.IsTruthyOne(util.MapGetCI(n, "vip")) {
				n["address"] = ""
			}
			n["xhttpRealPort"] = util.MapGetCI(n, "xhttprealport")
			n["xhttpTlsPort"] = util.MapGetCI(n, "xhttptlsport")
			n["xhttpRealServer"] = util.MapGetCI(n, "xhttprealserver")
			n["shortIds"] = util.MapGetCI(n, "shortids")
			result = append(result, n)
		}
		web.JSON(c, gin.H{"err": 0, "data": encryptNodes(result, token)})
		return
	}
	list, err := h.QueryMaps("select * from all_node where status = 1 order by `level`")
	if err != nil {
		h.Log.App.Error("getAllNodeList all_node failed", zap.Error(err))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	for _, n := range list {
		if !vip && util.IsTruthyOne(util.MapGetCI(n, "vip")) {
			n["node_conf"] = ""
		}
	}
	web.JSON(c, gin.H{"err": 0, "data": encryptNodes(list, token)})
}

func (h *UserHandler) GetAndroidNodeList(c *gin.Context, user *model.UserSession) {
	token := c.GetString("token")
	vip := user.ExpireSenconds > time.Now().Unix()
	list, err := h.QueryMaps("select * from desktop_node where status = 1 order by `level`")
	if err != nil {
		h.Log.App.Error("getAndroidNodeList query failed", zap.Error(err))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	result := make([]map[string]any, 0, len(list))
	for _, n := range list {
		if util.StringVal(util.MapGetCI(n, "use_for")) == "pc" {
			continue
		}
		if !vip && util.IsTruthyOne(util.MapGetCI(n, "vip")) {
			n["address"] = ""
		}
		n["xhttpRealPort"] = util.MapGetCI(n, "xhttprealport")
		n["xhttpTlsPort"] = util.MapGetCI(n, "xhttptlsport")
		n["xhttpRealServer"] = util.MapGetCI(n, "xhttprealserver")
		n["shortIds"] = util.MapGetCI(n, "shortids")
		result = append(result, n)
	}
	web.JSON(c, gin.H{"err": 0, "data": encryptNodes(result, token)})
}

func (h *UserHandler) GetUserInfo(c *gin.Context, user *model.UserSession) {
	refresh := util.Post(c, "refresh")
	if refresh == "true" {
		row, err := h.QueryMap("select user_email,vip,expire from gecko.user_profile where user_email = ? and actively = 1", user.Email)
		if err != nil {
			h.Log.App.Error("getUserInfo refresh failed", zap.Error(err), zap.String("email", user.Email))
		} else if row != nil {
			expire := util.StringVal(util.MapGetCI(row, "expire"))
			if expire != "" {
				user.Expire = expire
				user.ExpireSenconds = util.ParseExpire(expire, h.Loc)
				if err := web.SaveUser(h.App, user); err != nil {
					h.Log.App.Error("getUserInfo save session failed", zap.Error(err), zap.Int64("uid", user.UID))
				}
			} else {
				user.ExpireSenconds = 0
			}
		}
	}
	now := time.Now().Unix()
	expireTime := user.ExpireSenconds
	userdata := gin.H{
		"uid":        user.UID,
		"username":   user.Username,
		"email":      user.Email,
		"expiration": util.FormatDateTime(expireTime, h.Loc),
	}
	if expireTime > now {
		userdata["level"] = 1
		userdata["leftTime"] = expireTime - now
	} else {
		userdata["level"] = 0
		userdata["leftTime"] = 0
	}
	web.JSON(c, gin.H{"err": 0, "data": userdata})
}

func (h *UserHandler) ModifyPwd(c *gin.Context, user *model.UserSession) {
	oldPwd := util.Post(c, "oldPwd")
	newPwd := util.Post(c, "newPwd")
	row, err := h.QueryMap("select * from gecko.user_profile where user_email = ? and actively = 1", user.Email)
	if err != nil {
		h.Log.App.Error("modifyPwd query failed", zap.Error(err), zap.String("email", user.Email))
		web.JSON(c, gin.H{"err": 1, "data": "未知用户！"})
		return
	}
	if row == nil {
		web.JSON(c, gin.H{"err": 1, "data": "未知用户！"})
		return
	}
	dbPwd := util.StringVal(util.MapGetCI(row, "password"))
	if dbPwd != util.MD5String(oldPwd) {
		web.JSON(c, gin.H{"err": 1, "data": "旧密码错误！"})
		return
	}
	if err := h.DB.Exec("update gecko.user_profile set password = ? where user_email = ?", util.MD5String(newPwd), user.Email).Error; err != nil {
		h.Log.App.Error("modifyPwd update failed", zap.Error(err), zap.String("email", user.Email))
		web.JSON(c, gin.H{"err": 1, "data": "未知用户！"})
		return
	}
	web.JSON(c, gin.H{"err": 0})
}

func (h *UserHandler) CheckVersion(c *gin.Context) {
	from := util.Get(c, "client_type")
	version := util.Get(c, "version")
	switch from {
	case "IOS":
		c1 := strings.Compare(h.AppCfg.Get("ios_min_version"), version)
		c2 := strings.Compare(h.AppCfg.Get("ios_max_version"), version)
		if c1 > 0 || c2 < 0 {
			web.JSON(c, gin.H{"err": 2})
		} else if c1 <= 0 && c2 > 0 {
			web.JSON(c, gin.H{"err": 1})
		} else {
			web.JSON(c, gin.H{"err": 0})
		}
	case "Android":
		c1 := strings.Compare(h.AppCfg.Get("android_min_version"), version)
		c2 := strings.Compare(h.AppCfg.Get("android_max_version"), version)
		if c1 > 0 || c2 < 0 {
			web.JSON(c, gin.H{"err": 2})
		} else if c1 <= 0 && c2 > 0 {
			web.JSON(c, gin.H{"err": 1})
		} else {
			web.JSON(c, gin.H{"err": 0})
		}
	default:
		web.JSON(c, gin.H{"err": 0})
	}
}

func (h *UserHandler) GetNoticeList(c *gin.Context) {
	from := strings.ToUpper(util.Get(c, "client_type"))
	typ := 4
	switch from {
	case "WIN":
		typ = 4
	case "MAC":
		typ = 5
	case "ANDROID":
		typ = 6
	}
	list, err := h.QueryMaps("select id,title as cn_title,content as cn_content,en_title,en_content from gecko.notice where active = 1 and (type = ? or type = 10) order by last_modify desc", typ)
	if err != nil {
		h.Log.App.Error("getNoticeList query failed", zap.Error(err))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	local := web.Locale(c)
	for _, notice := range list {
		if local == "zh_CN" {
			notice["title"] = util.MapGetCI(notice, "cn_title")
			notice["content"] = util.MapGetCI(notice, "cn_content")
		} else {
			notice["title"] = util.MapGetCI(notice, "en_title")
			notice["content"] = util.MapGetCI(notice, "en_content")
		}
		deleteCI(notice, "cn_title")
		deleteCI(notice, "cn_content")
		deleteCI(notice, "en_content")
		deleteCI(notice, "en_title")
	}
	web.JSON(c, gin.H{"err": 0, "data": list})
}

func (h *UserHandler) Feedback(c *gin.Context, user *model.UserSession) {
	from := util.Get(c, "client_type")
	message := from + ":" + util.Post(c, "message")
	res := h.DB.Exec("insert into gecko.comment(ask_content,ask_time,ask_user) values(?,current_timestamp,?)", message, user.Email)
	if res.Error != nil {
		h.Log.App.Error("feedback insert failed", zap.Error(res.Error), zap.String("email", user.Email))
		web.JSON(c, gin.H{"err": 1})
		return
	}
	if res.RowsAffected > 0 {
		web.JSON(c, gin.H{"err": 0})
		return
	}
	web.JSON(c, gin.H{"err": 1})
}

func (h *UserHandler) RecordClientLogin(c *gin.Context, user *model.UserSession) {
	clientUUID := util.Post(c, "clientUUID")
	apiSource := util.Post(c, "apiSource")
	version := util.Post(c, "version")
	clientApiVersion := util.Post(c, "clientVersion")
	clientOs := util.Post(c, "clientOs")
	clientOsMsg := util.Post(c, "clientOsMsg")
	clientOsArch := util.Post(c, "clientOsArch")
	hostname := util.Post(c, "hostname")
	if clientUUID == "" || apiSource == "" {
		web.JSON(c, gin.H{"err": -1, "data": web.T(c, "invalidParms")})
		return
	}
	loginID := user.Email
	now := time.Now().In(h.Loc).Format("2006-01-02 15:04:05")
	ua := c.Request.UserAgent()
	apiSourceLower := strings.ToLower(apiSource)
	if apiSourceLower == "web" || clientOs == "" {
		parsedOs, parsedOsMsg := parseUA(ua)
		if apiSourceLower == "web" || clientOs == "" {
			clientOs = parsedOs
		}
		if apiSourceLower == "web" || clientOsMsg == "" {
			clientOsMsg = parsedOsMsg
		}
	}

	cleanKey := "tbuserclient_clean_day"
	today := time.Now().In(h.Loc).Format("20060102")
	var lastDay string
	_ = h.Cache.Get(cleanKey, &lastDay)
	if lastDay != today {
		expireTime := time.Now().In(h.Loc).Add(-7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
		if err := h.DB.Exec("delete from gecko.tbuserclient where refreshTime < ?", expireTime).Error; err != nil {
			h.Log.App.Error("tbuserclient clean failed", zap.Error(err))
		}
		_ = h.Cache.Set(cleanKey, today, 24*time.Hour)
	}

	exist, err := h.QueryMap("select * from gecko.tbuserclient where loginId = ? and clientId = ? limit 1", loginID, clientUUID)
	if err != nil {
		h.Log.App.Error("recordClientLogin query failed", zap.Error(err), zap.String("loginId", loginID))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	ip := util.ClientIP(c)
	if exist != nil {
		lid := util.MapGetCI(exist, "lid")
		if lid == nil {
			lid = util.MapGetCI(exist, "lId")
		}
		if err := h.DB.Exec(`update gecko.tbuserclient set clientMsg = ?, clientOs = ?, clientOsMsg = ?, clientOsArch = ?,
			clientVersion = ?, clientApiVersion = ?, hostname = ?, refreshIp = ?, refreshTime = ?, state = 1, logout = 0 where lid = ?`,
			apiSource, clientOs, clientOsMsg, clientOsArch, version, clientApiVersion, hostname, ip, now, lid).Error; err != nil {
			h.Log.App.Error("recordClientLogin update failed", zap.Error(err), zap.String("loginId", loginID))
		}
	} else {
		if err := h.DB.Exec(`insert into gecko.tbuserclient(lId,loginId,loginIp,loginTime,clientId,clientMsg,clientOs,clientOsMsg,clientOsArch,clientVersion,clientApiVersion,hostname,refreshIp,refreshTime,state,logout)
			values(null,?,?,?,?,?,?,?,?,?,?,?,?,?,1,0)`,
			loginID, ip, now, clientUUID, apiSource, clientOs, clientOsMsg, clientOsArch, version, clientApiVersion, hostname, ip, now).Error; err != nil {
			h.Log.App.Error("recordClientLogin insert failed", zap.Error(err), zap.String("loginId", loginID))
		}
	}
	web.JSON(c, gin.H{"err": 0})
}

func encryptNodes(data any, token string) string {
	b, _ := json.Marshal(data)
	key := "BWAjPh" + token + "FhShxnc5"
	return xxtea.EncryptString(string(b), key)
}

func parseUA(ua string) (osName, osMsg string) {
	osName = "Unknown"
	switch {
	case strings.Contains(ua, "Win"):
		osName = "Windows"
	case strings.Contains(ua, "Mac"):
		osName = "macOS"
	case regexp.MustCompile(`(?i)Android`).MatchString(ua):
		osName = "Android"
	case regexp.MustCompile(`(?i)iPhone|iPad|iPod`).MatchString(ua):
		osName = "iOS"
	case strings.Contains(ua, "Linux"):
		osName = "Linux"
	}

	browser, browserVer := "Unknown", "Unknown"
	switch {
	case strings.Contains(ua, "OPR") || strings.Contains(ua, "Opera"):
		browser = "Opera"
		if m := regexp.MustCompile(`OPR/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			browserVer = m[1]
		}
	case strings.Contains(ua, "Edg"):
		browser = "Edge"
		if m := regexp.MustCompile(`Edg/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			browserVer = m[1]
		}
	case strings.Contains(ua, "Chrome"):
		browser = "Chrome"
		if m := regexp.MustCompile(`Chrome/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			browserVer = m[1]
		}
	case strings.Contains(ua, "Firefox"):
		browser = "Firefox"
		if m := regexp.MustCompile(`Firefox/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			browserVer = m[1]
		}
	case strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") && !strings.Contains(ua, "Chromium"):
		browser = "Safari"
		if m := regexp.MustCompile(`Version/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			browserVer = m[1]
		}
	case strings.Contains(ua, "MSIE") || strings.Contains(ua, "Trident"):
		browser = "IE"
	}

	parts := []string{browser + " " + browserVer}
	if osName == "Windows" {
		if m := regexp.MustCompile(`Windows NT ([0-9.]+)`).FindStringSubmatch(ua); len(m) > 1 {
			parts = append(parts, "Windows NT "+m[1])
		}
	}
	return osName, strings.Join(parts, " / ")
}
