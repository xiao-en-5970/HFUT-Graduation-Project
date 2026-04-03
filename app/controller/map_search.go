package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/amap"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// MapInputTips GET /map/input-tips?keywords=xxx&city=合肥&citylimit=1 — 输入提示（联想）
func MapInputTips(ctx *gin.Context) {
	if config.AmapKey == "" {
		reply.ReplyErrWithMessage(ctx, "未配置 AMAP_KEY，无法搜索地点")
		return
	}
	keywords := strings.TrimSpace(ctx.Query("keywords"))
	if keywords == "" {
		reply.ReplyInvalidParams(ctx, errors.New("keywords 必填"))
		return
	}
	city := strings.TrimSpace(ctx.Query("city"))
	citylimit := ctx.Query("citylimit") == "1" || ctx.Query("citylimit") == "true"
	list, err := amap.InputTips(ctx.Request.Context(), config.AmapKey, keywords, city, citylimit)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list})
}

// MapPlaceText GET /map/place-text?keywords=xxx&city=合肥&page=1&offset=20 — POI 关键字搜索
func MapPlaceText(ctx *gin.Context) {
	if config.AmapKey == "" {
		reply.ReplyErrWithMessage(ctx, "未配置 AMAP_KEY，无法搜索地点")
		return
	}
	keywords := strings.TrimSpace(ctx.Query("keywords"))
	if keywords == "" {
		reply.ReplyInvalidParams(ctx, errors.New("keywords 必填"))
		return
	}
	city := strings.TrimSpace(ctx.Query("city"))
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "20"))
	list, err := amap.PlaceTextSearch(ctx.Request.Context(), config.AmapKey, keywords, city, page, offset)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "page": page, "offset": offset})
}

// MapDistrict GET /map/district?keywords=100000&page=1&offset=20 — 下级行政区（省→市→区级联）
func MapDistrict(ctx *gin.Context) {
	if config.AmapKey == "" {
		reply.ReplyErrWithMessage(ctx, "未配置 AMAP_KEY")
		return
	}
	keywords := strings.TrimSpace(ctx.Query("keywords"))
	if keywords == "" {
		keywords = "100000"
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "20"))
	list, err := amap.DistrictChildren(ctx.Request.Context(), config.AmapKey, keywords, page, offset)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": list, "keywords": keywords, "page": page, "offset": offset})
}
