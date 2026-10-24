package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mb2-go/internal/util"
	"mb2-go/internal/web"
)

type IndexHandler struct{ *web.App }

func (h *IndexHandler) Handle(c *gin.Context, action string) {
	switch strings.ToLower(action) {
	case "getgoodslist":
		h.GetGoodsList(c)
	case "getconfig":
		h.GetConfig(c)
	case "checkversion":
		h.CheckVersion(c)
	case "checkpackage":
		h.CheckPackage(c)
	case "checkwinversion":
		h.CheckWinVersion(c)
	case "checkmacversion":
		h.CheckMacVersion(c)
	case "checkwgversion":
		h.CheckWGVersion(c)
	default:
		c.Status(404)
	}
}

func (h *IndexHandler) GetGoodsList(c *gin.Context) {
	list, err := h.QueryMaps("select iid,title,introduction,prices,amount,cn_memo,en_memo from superpanel.shop_item where status = 1 order by iid desc")
	if err != nil {
		h.Log.App.Error("getGoodsList query failed", zap.Error(err))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	local := web.Locale(c)
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if local != "zh_CN" {
			iid := util.StringVal(util.MapGetCI(item, "iid"))
			switch iid {
			case "1":
				item["title"] = "Month"
			case "3":
				item["title"] = "Quarter"
			case "12":
				item["title"] = "Year"
			default:
				item["title"] = "Half Year"
			}
			item["introduction"] = "Unlimited flow, and support PC, Android, IOS connections"
			item["memo"] = util.MapGetCI(item, "en_memo")
		} else {
			item["memo"] = util.MapGetCI(item, "cn_memo")
		}
		deleteCI(item, "en_memo")
		deleteCI(item, "cn_memo")
		out = append(out, item)
	}
	web.JSON(c, gin.H{"err": 0, "data": out})
}

func (h *IndexHandler) GetConfig(c *gin.Context) {
	list, err := h.QueryMaps("select field_name,field_value1 from gecko.sys_params where field_name like 'app_config%'")
	if err != nil {
		h.Log.App.Error("getConfig query failed", zap.Error(err))
		web.JSON(c, gin.H{"err": 2, "data": web.T(c, "serverException")})
		return
	}
	data := map[string]any{}
	for _, item := range list {
		name := util.StringVal(util.MapGetCI(item, "field_name"))
		data[name] = util.MapGetCI(item, "field_value1")
	}
	web.JSON(c, gin.H{"err": 0, "data": data})
}

func (h *IndexHandler) CheckVersion(c *gin.Context) {
	from := strings.ToUpper(util.Get(c, "client_type"))
	switch from {
	case "MAC":
		h.CheckMacVersion(c)
	case "WIN":
		h.CheckWinVersion(c)
	default:
		h.CheckWGVersion(c)
	}
}

func (h *IndexHandler) CheckPackage(c *gin.Context) {
	from := util.Get(c, "client_type")
	switch from {
	case "MAC":
		web.JSON(c, gin.H{"err": 0, "data": gin.H{
			"packageVersion": h.AppCfg.Get("mac_package_ver"),
			"packageUrl":     h.AppCfg.Get("mac_package_url"),
		}})
	case "WIN":
		web.JSON(c, gin.H{"err": 0, "data": gin.H{
			"packageVersion": h.AppCfg.Get("win_package_ver"),
			"packageUrl":     h.AppCfg.Get("win_package_url"),
		}})
	default:
		web.Empty(c)
	}
}

func (h *IndexHandler) CheckWinVersion(c *gin.Context) {
	h.replyClientVersion(c, "win_min_version", "win_update_version", "win_update_info", "win_update_url", "win_update_size", false)
}

func (h *IndexHandler) CheckMacVersion(c *gin.Context) {
	h.replyClientVersion(c, "mac_min_version", "mac_update_version", "mac_update_info", "mac_update_url", "mac_update_size", false)
}

func (h *IndexHandler) CheckWGVersion(c *gin.Context) {
	h.replyClientVersion(c, "wg_min_version", "wg_update_version", "wg_update_info", "wg_update_url", "wg_update_size", true)
}

func (h *IndexHandler) replyClientVersion(c *gin.Context, minKey, newKey, infoKey, urlKey, sizeKey string, withCode bool) {
	version := util.Get(c, "version")
	minVer := h.AppCfg.Get(minKey)
	newVer := h.AppCfg.Get(newKey)
	vc := util.VersionCompare(version, minVer)
	cc := util.VersionCompare(version, newVer)
	if vc < 0 {
		data := gin.H{
			"forceUpdate":   1,
			"updateInfo":    h.AppCfg.Get(infoKey),
			"updateUrl":     h.AppCfg.Get(urlKey),
			"updateVersion": newVer,
			"updateSize":    h.AppCfg.Get(sizeKey),
		}
		if withCode {
			data["versionCode"] = h.AppCfg.Get("wg_version_code")
		}
		web.JSON(c, gin.H{"err": 0, "data": data})
		return
	}
	if cc >= 0 {
		web.JSON(c, gin.H{"err": 0})
		return
	}
	data := gin.H{
		"forceUpdate":   0,
		"updateInfo":    h.AppCfg.Get(infoKey),
		"updateUrl":     h.AppCfg.Get(urlKey),
		"updateVersion": newVer,
		"updateSize":    h.AppCfg.Get(sizeKey),
	}
	if withCode {
		data["versionCode"] = h.AppCfg.Get("wg_version_code")
	}
	web.JSON(c, gin.H{"err": 0, "data": data})
}

func deleteCI(m map[string]any, key string) {
	for k := range m {
		if strings.EqualFold(k, key) {
			delete(m, k)
		}
	}
}
