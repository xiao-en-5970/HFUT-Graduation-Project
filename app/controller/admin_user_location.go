package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// AdminUserLocationList GET /admin/user-locations
func AdminUserLocationList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	userID, _ := strconv.ParseUint(ctx.DefaultQuery("user_id", "0"), 10, 32)
	allStatus := ctx.Query("all_status") == "1" || ctx.Query("all_status") == "true"
	list, total, err := dao.UserLocation().ListForAdmin(ctx.Request.Context(), page, pageSize, uint(userID), allStatus)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	ids := make([]uint, 0, len(list))
	seen := map[uint]struct{}{}
	for _, l := range list {
		if _, ok := seen[l.UserID]; ok {
			continue
		}
		seen[l.UserID] = struct{}{}
		ids = append(ids, l.UserID)
	}
	users, _ := dao.User().GetByIDs(ctx.Request.Context(), ids)
	out := make([]map[string]interface{}, len(list))
	for i, l := range list {
		m := userLocationToMap(l)
		if m == nil {
			m = map[string]interface{}{}
		}
		if u := users[l.UserID]; u != nil {
			m["username"] = u.Username
		} else {
			m["username"] = ""
		}
		out[i] = m
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": out, "total": total, "page": page, "page_size": pageSize})
}

// AdminUserLocationDelete DELETE /admin/user-locations/:id 软删除
func AdminUserLocationDelete(ctx *gin.Context) {
	id, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := service.UserLocation().AdminDelete(ctx, id); err != nil {
		if errors.Is(err, errno.ErrUserLocationNotFound) {
			reply.ReplyErrWithMessage(ctx, "收货地址不存在")
			return
		}
		reply.ReplyErrWithMessage(ctx, err.Error())
		return
	}
	reply.ReplyOK(ctx)
}
