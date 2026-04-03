package service

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

type userLocationService struct{}

// UserLocationCreateReq 新增收货地址
type UserLocationCreateReq struct {
	Label     string   `json:"label"`
	Addr      string   `json:"addr"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	IsDefault bool     `json:"is_default"`
}

// UserLocationUpdateReq 更新收货地址（零值表示不修改；is_default 传 true 可设为默认）
type UserLocationUpdateReq struct {
	Label     *string  `json:"label"`
	Addr      *string  `json:"addr"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	IsDefault *bool    `json:"is_default"`
}

func validateUserLocationAddr(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errors.New("addr 不能为空")
	}
	return nil
}

func validateLatLngPair(lat, lng *float64) error {
	if (lat == nil) != (lng == nil) {
		return errors.New("lat 与 lng 须同时填写或同时留空")
	}
	return nil
}

func (s *userLocationService) List(ctx *gin.Context, userID uint) ([]*model.UserLocation, error) {
	return dao.UserLocation().ListByUserID(ctx.Request.Context(), userID)
}

func (s *userLocationService) Create(ctx *gin.Context, userID uint, req UserLocationCreateReq) (uint, error) {
	addr := strings.TrimSpace(req.Addr)
	if err := validateUserLocationAddr(addr); err != nil {
		return 0, err
	}
	if err := validateLatLngPair(req.Lat, req.Lng); err != nil {
		return 0, err
	}
	n, err := dao.UserLocation().CountActiveByUser(ctx.Request.Context(), userID)
	if err != nil {
		return 0, err
	}
	isDef := req.IsDefault || n == 0
	label := strings.TrimSpace(req.Label)

	o := &model.UserLocation{
		UserID:    userID,
		Label:     label,
		Addr:      addr,
		Lat:       req.Lat,
		Lng:       req.Lng,
		IsDefault: isDef,
		Status:    constant.StatusValid,
	}
	if !isDef {
		if err := dao.UserLocation().Create(ctx.Request.Context(), o); err != nil {
			return 0, err
		}
		return o.ID, nil
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.UserLocation().ClearDefaultTx(ctx.Request.Context(), tx, userID); err != nil {
			return err
		}
		o.IsDefault = true
		return dao.UserLocation().CreateTx(ctx.Request.Context(), tx, o)
	})
	if err != nil {
		return 0, err
	}
	return o.ID, nil
}

func (s *userLocationService) Update(ctx *gin.Context, userID, id uint, req UserLocationUpdateReq) error {
	_, err := dao.UserLocation().GetByIDAndUserID(ctx.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrUserLocationNotFound
		}
		return err
	}
	updates := make(map[string]interface{})
	if req.Label != nil {
		updates["label"] = strings.TrimSpace(*req.Label)
	}
	if req.Addr != nil {
		a := strings.TrimSpace(*req.Addr)
		if a == "" {
			return errors.New("addr 不能为空")
		}
		updates["addr"] = a
	}
	if req.Lat != nil || req.Lng != nil {
		if req.Lat == nil || req.Lng == nil {
			return errors.New("更新坐标时 lat 与 lng 须同时传入")
		}
		if err := validateLatLngPair(req.Lat, req.Lng); err != nil {
			return err
		}
		updates["lat"] = req.Lat
		updates["lng"] = req.Lng
	}
	if req.IsDefault != nil && *req.IsDefault {
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.UserLocation().ClearDefaultTx(ctx.Request.Context(), tx, userID); err != nil {
				return err
			}
			updates["is_default"] = true
			return dao.UserLocation().UpdateColumnsTx(ctx.Request.Context(), tx, id, updates)
		})
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.UserLocation().UpdateColumns(ctx.Request.Context(), id, updates)
}

func (s *userLocationService) Delete(ctx *gin.Context, userID, id uint) error {
	row, err := dao.UserLocation().GetByIDAndUserID(ctx.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrUserLocationNotFound
		}
		return err
	}
	if err := dao.UserLocation().SoftDelete(ctx.Request.Context(), id); err != nil {
		return err
	}
	if !row.IsDefault {
		return nil
	}
	other, err := dao.UserLocation().FirstActiveOther(ctx.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return dao.UserLocation().UpdateColumns(ctx.Request.Context(), other.ID, map[string]interface{}{"is_default": true})
}

// AdminDelete 管理端软删除任意用户的收货地址
func (s *userLocationService) AdminDelete(ctx *gin.Context, id uint) error {
	row, err := dao.UserLocation().GetByID(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrUserLocationNotFound
		}
		return err
	}
	if row.Status != constant.StatusValid {
		return errors.New("该地址已删除")
	}
	if err := dao.UserLocation().SoftDelete(ctx.Request.Context(), id); err != nil {
		return err
	}
	if !row.IsDefault {
		return nil
	}
	other, err := dao.UserLocation().FirstActiveOther(ctx.Request.Context(), row.UserID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return dao.UserLocation().UpdateColumns(ctx.Request.Context(), other.ID, map[string]interface{}{"is_default": true})
}

func (s *userLocationService) SetDefault(ctx *gin.Context, userID, id uint) error {
	_, err := dao.UserLocation().GetByIDAndUserID(ctx.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrUserLocationNotFound
		}
		return err
	}
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.UserLocation().ClearDefaultTx(ctx.Request.Context(), tx, userID); err != nil {
			return err
		}
		return dao.UserLocation().UpdateColumnsTx(ctx.Request.Context(), tx, id, map[string]interface{}{"is_default": true})
	})
}
