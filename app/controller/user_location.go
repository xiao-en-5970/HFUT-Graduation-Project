package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

func userLocationToMap(l *model.UserLocation) map[string]interface{} {
	if l == nil {
		return nil
	}
	m := map[string]interface{}{
		"id":         l.ID,
		"user_id":    l.UserID,
		"label":      l.Label,
		"addr":       l.Addr,
		"is_default": l.IsDefault,
		"status":     l.Status,
		"created_at": l.CreatedAt,
		"updated_at": l.UpdatedAt,
	}
	if l.Lat != nil {
		m["lat"] = *l.Lat
	} else {
		m["lat"] = nil
	}
	if l.Lng != nil {
		m["lng"] = *l.Lng
	} else {
		m["lng"] = nil
	}
	return m
}

// UserLocationList GET /user/locations
func UserLocationList(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	list, err := service.UserLocation().List(ctx, userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	out := make([]map[string]interface{}, len(list))
	for i, l := range list {
		out[i] = userLocationToMap(l)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out})
}

// UserLocationCreate POST /user/locations
func UserLocationCreate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.UserLocationCreateReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	id, err := service.UserLocation().Create(ctx, userID, req)
	if err != nil {
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"id": id})
}

// UserLocationUpdate PUT /user/locations/:id
func UserLocationUpdate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	var req service.UserLocationUpdateReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.UserLocation().Update(ctx, userID, uint(id), req); err != nil {
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}

// UserLocationDelete DELETE /user/locations/:id
func UserLocationDelete(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.UserLocation().Delete(ctx, userID, uint(id)); err != nil {
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

// UserLocationSetDefault POST /user/locations/:id/default
func UserLocationSetDefault(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.UserLocation().SetDefault(ctx, userID, uint(id)); err != nil {
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在")
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}
