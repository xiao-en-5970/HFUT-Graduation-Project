package service

import (
	"errors"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/snowflake"
	"gorm.io/gorm"
)

type goodService struct{}

type CreateGoodReq struct {
	Title         string   `json:"title" binding:"required"`
	Content       string   `json:"content" binding:"required"`
	GoodsCategory int16    `json:"goods_category"` // 1二手买卖 2有偿求助，默认1
	GoodsType     int16    `json:"goods_type"`     // 1送货上门 2自提 3在线，默认1
	GoodsAddr     string   `json:"goods_addr"`     // 商品地址：默认发货地/自提点（优先）
	PickupAddr    string   `json:"pickup_addr"`    // 兼容旧字段，与 goods_addr 合并存库
	Price         int      `json:"price"`          // 价格（分）
	MarkedPrice   int      `json:"marked_price"`   // 标价（分）
	Stock         int      `json:"stock"`          // 库存
	Images        []string `json:"images"`         // 图片 URL 列表
	PaymentQRURL  string   `json:"payment_qr_url"` // 收款码图片 URL；仅二手买卖有效
	HasDeadline   bool     `json:"has_deadline"`   // 是否启用定时下架
	Deadline      *string  `json:"deadline"`       // RFC3339/"2006-01-02 15:04:05" 时间字符串；HasDeadline=true 时必填
	GoodsLat      *float64 `json:"goods_lat"`      // 商品位置纬度 WGS84，与发货地一致
	GoodsLng      *float64 `json:"goods_lng"`      // 商品位置经度 WGS84
}

type UpdateGoodReq struct {
	Title         *string   `json:"title"`
	Content       *string   `json:"content"`
	GoodsCategory *int16    `json:"goods_category"`
	GoodsType     *int16    `json:"goods_type"`
	GoodsAddr     *string   `json:"goods_addr"`
	PickupAddr    *string   `json:"pickup_addr"`
	Price         *int      `json:"price"`
	MarkedPrice   *int      `json:"marked_price"`
	Stock         *int      `json:"stock"`
	Images        *[]string `json:"images"`
	PaymentQRURL  *string   `json:"payment_qr_url"`
	HasDeadline   *bool     `json:"has_deadline"`
	Deadline      *string   `json:"deadline"` // 字符串；空串或 "null" 表示清空 deadline
	GoodsLat      *float64  `json:"goods_lat"`
	GoodsLng      *float64  `json:"goods_lng"`
}

// parseDeadline 解析前端传来的 deadline 字符串；接受 RFC3339 或 "2006-01-02 15:04:05"。
// 返回 *time.Time（nil 表示无截止时间），以及可能的校验错误。
// 可接受的"空"输入：空串 / "null"。
func parseDeadline(s string, hasDeadline bool) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if !hasDeadline || s == "" || strings.EqualFold(s, "null") {
		return nil, nil
	}
	// 优先 RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, nil
	}
	// 退化为本地时区的 "YYYY-MM-DD HH:MM:SS"
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return &t, nil
	}
	// 只给日期，按当天 23:59:59
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		tt := t.Add(24*time.Hour - time.Second)
		return &tt, nil
	}
	return nil, errors.New("deadline 格式非法，应为 RFC3339 或 YYYY-MM-DD [HH:MM:SS]")
}

func (s *goodService) Create(ctx *gin.Context, userID uint, schoolID uint, req CreateGoodReq) (uint, error) {
	if schoolID == 0 {
		return 0, errno.ErrSchoolNotBound
	}
	if req.Price < 0 {
		req.Price = 0
	}
	if req.Stock < 0 {
		req.Stock = 0
	}
	category := req.GoodsCategory
	if !constant.IsValidGoodsCategory(category) {
		category = constant.GoodsCategoryNormal
	}
	gt := req.GoodsType
	if gt != constant.GoodsTypeDelivery && gt != constant.GoodsTypePickup && gt != constant.GoodsTypeOnline {
		// 求助默认在线协作（无物流）；二手默认送货上门
		if category == constant.GoodsCategoryHelp {
			gt = constant.GoodsTypeOnline
		} else {
			gt = constant.GoodsTypeDelivery
		}
	}
	qr := strings.TrimSpace(req.PaymentQRURL)
	if category == constant.GoodsCategoryHelp {
		// 有偿求助：发布者是付款方，创建时不接受收款码；安全清空
		qr = ""
	}
	deadlineAt, err := parseDeadline(safeDeref(req.Deadline), req.HasDeadline)
	if err != nil {
		return 0, err
	}
	// 启用 deadline 但时间已过去，拒绝创建
	if req.HasDeadline && deadlineAt != nil && !deadlineAt.After(time.Now()) {
		return 0, errors.New("deadline 必须晚于当前时间")
	}
	uid := int(userID)
	sid := int(schoolID)
	addr := strings.TrimSpace(req.GoodsAddr)
	if addr == "" {
		addr = strings.TrimSpace(req.PickupAddr)
	}
	g := &model.Good{
		UserID:        &uid,
		SchoolID:      &sid,
		Title:         req.Title,
		Content:       req.Content,
		GoodsCategory: category,
		GoodsType:     gt,
		GoodsAddr:     addr,
		PickupAddr:    addr,
		Price:         req.Price,
		MarkedPrice:   req.MarkedPrice,
		Stock:         req.Stock,
		PaymentQRURL:  oss.PathForStorage(qr),
		HasDeadline:   req.HasDeadline && deadlineAt != nil,
		Deadline:      deadlineAt,
		GoodStatus:    dao.GoodStatusOffShelf, // 新建为下架，需上架后才可见
		Status:        constant.StatusValid,
	}
	if req.GoodsLat != nil && req.GoodsLng != nil {
		g.GoodsLat = req.GoodsLat
		g.GoodsLng = req.GoodsLng
	}
	if len(req.Images) > 0 {
		paths := make([]string, len(req.Images))
		for i, p := range req.Images {
			paths[i] = oss.PathForStorage(p)
		}
		g.Images = pq.StringArray(paths)
		g.ImageCount = len(req.Images)
	}
	return dao.Good().Create(ctx.Request.Context(), g)
}

// safeDeref 简化 *string 取值，nil 返回空串
func safeDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *goodService) Get(ctx *gin.Context, id uint, viewerID uint, schoolID uint) (*model.Good, error) {
	g, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), id, schoolID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errno.ErrGoodNotFoundOrNoPermission
		}
		return nil, err
	}
	return g, nil
}

func (s *goodService) GetAllowOffShelf(ctx *gin.Context, id uint, viewerID uint, schoolID uint) (*model.Good, error) {
	g, err := dao.Good().GetByIDWithSchoolAllowOffShelf(ctx.Request.Context(), id, schoolID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errno.ErrGoodNotFoundOrNoPermission
		}
		return nil, err
	}
	// 下架商品仅卖家本人可见
	if g.GoodStatus == dao.GoodStatusOffShelf && g.UserID != nil && uint(*g.UserID) != viewerID {
		return nil, errno.ErrGoodNotFoundOrNoPermission
	}
	return g, nil
}

func (s *goodService) Update(ctx *gin.Context, id uint, userID uint, schoolID uint, req UpdateGoodReq) error {
	ok, err := dao.Good().IsOwnedByUser(ctx.Request.Context(), id, userID)
	if err != nil || !ok {
		return errno.ErrGoodNotFoundOrNoPermission
	}
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Price != nil && *req.Price >= 0 {
		updates["price"] = *req.Price
	}
	if req.MarkedPrice != nil && *req.MarkedPrice >= 0 {
		updates["marked_price"] = *req.MarkedPrice
	}
	if req.Stock != nil && *req.Stock >= 0 {
		updates["stock"] = *req.Stock
	}
	if req.Images != nil {
		paths := make([]string, len(*req.Images))
		for i, p := range *req.Images {
			paths[i] = oss.PathForStorage(p)
		}
		updates["images"] = pq.StringArray(paths)
		updates["image_count"] = len(paths)
	}
	if req.GoodsType != nil {
		gt := *req.GoodsType
		if gt == constant.GoodsTypeDelivery || gt == constant.GoodsTypePickup || gt == constant.GoodsTypeOnline {
			updates["goods_type"] = gt
		}
	}
	if req.GoodsCategory != nil {
		if constant.IsValidGoodsCategory(*req.GoodsCategory) {
			updates["goods_category"] = *req.GoodsCategory
			// 切换到有偿求助时，顺手清空收款码
			if *req.GoodsCategory == constant.GoodsCategoryHelp {
				updates["payment_qr_url"] = ""
			}
		}
	}
	if req.PaymentQRURL != nil {
		// 如果同 payload 里已显式把 category 切成求助，尊重那次清空，不覆盖
		if _, alreadyCleared := updates["payment_qr_url"]; !alreadyCleared {
			updates["payment_qr_url"] = oss.PathForStorage(strings.TrimSpace(*req.PaymentQRURL))
		}
	}
	if req.HasDeadline != nil {
		updates["has_deadline"] = *req.HasDeadline
		// 关闭 deadline 时一并清空时间点
		if !*req.HasDeadline {
			updates["deadline"] = nil
		}
	}
	if req.Deadline != nil {
		// 允许前端传空串主动清空
		want := req.HasDeadline != nil && *req.HasDeadline
		// 若本次 payload 没显式传 HasDeadline，则沿用当前 has_deadline=true 的意图
		if req.HasDeadline == nil {
			want = true
		}
		t, err := parseDeadline(*req.Deadline, want)
		if err != nil {
			return err
		}
		if want && t != nil && !t.After(time.Now()) {
			return errors.New("deadline 必须晚于当前时间")
		}
		if t != nil {
			updates["deadline"] = *t
		} else {
			updates["deadline"] = nil
		}
	}
	if req.GoodsAddr != nil || req.PickupAddr != nil {
		addr := ""
		if req.GoodsAddr != nil {
			addr = strings.TrimSpace(*req.GoodsAddr)
		}
		if addr == "" && req.PickupAddr != nil {
			addr = strings.TrimSpace(*req.PickupAddr)
		}
		updates["goods_addr"] = addr
		updates["pickup_addr"] = addr
	}
	if req.GoodsLat != nil && req.GoodsLng != nil {
		updates["goods_lat"] = *req.GoodsLat
		updates["goods_lng"] = *req.GoodsLng
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, updates)
}

func (s *goodService) Publish(ctx *gin.Context, id uint, userID uint) error {
	ok, err := dao.Good().IsOwnedByUser(ctx.Request.Context(), id, userID)
	if err != nil || !ok {
		return errno.ErrGoodNotFoundOrNoPermission
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, map[string]interface{}{"good_status": dao.GoodStatusOnSale})
}

func (s *goodService) OffShelf(ctx *gin.Context, id uint, userID uint) error {
	ok, err := dao.Good().IsOwnedByUser(ctx.Request.Context(), id, userID)
	if err != nil || !ok {
		return errno.ErrGoodNotFoundOrNoPermission
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, map[string]interface{}{"good_status": dao.GoodStatusOffShelf})
}

func (s *goodService) List(ctx *gin.Context, schoolID uint, page, pageSize int, keyword string, sort string, category int16) ([]*model.Good, int64, error) {
	return dao.Good().List(ctx.Request.Context(), schoolID, page, pageSize, keyword, sort, category)
}

func (s *goodService) ListByUserID(ctx *gin.Context, targetUserID uint, viewerSchoolID uint, includeOffShelf bool, ownList bool, page, pageSize int) ([]*model.Good, int64, error) {
	return dao.Good().ListByUserID(ctx.Request.Context(), targetUserID, viewerSchoolID, includeOffShelf, ownList, page, pageSize)
}

func (s *goodService) uploadGoodImages(ctx *gin.Context, id uint, files []*multipart.FileHeader) ([]string, error) {
	if len(files) == 0 {
		return nil, errors.New("至少需要上传一张图片")
	}
	urls := make([]string, 0, len(files))
	for _, f := range files {
		ext := oss.ExtFromFilename(f.Filename)
		sfID := snowflake.NextID()
		relPath := oss.GoodImagePathWithSnowflake(id, sfID, ext)
		url, err := oss.Save(f, relPath)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (s *goodService) UploadImages(ctx *gin.Context, id uint, userID uint, files []*multipart.FileHeader) ([]string, error) {
	ok, err := dao.Good().IsOwnedByUser(ctx.Request.Context(), id, userID)
	if err != nil || !ok {
		return nil, errno.ErrGoodNotFoundOrNoPermission
	}
	return s.uploadGoodImages(ctx, id, files)
}

// AdminCreateGoodReq 管理端创建商品
type AdminCreateGoodReq struct {
	UserID      uint     `json:"user_id" binding:"required"`
	SchoolID    uint     `json:"school_id" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	GoodsType   int16    `json:"goods_type"`
	GoodsAddr   string   `json:"goods_addr"`
	PickupAddr  string   `json:"pickup_addr"`
	Price       int      `json:"price"`
	MarkedPrice int      `json:"marked_price"`
	Stock       int      `json:"stock"`
	Images      []string `json:"images"`
	GoodStatus  int      `json:"good_status"` // 可选 1在售 2下架 3已售出，默认下架
	GoodsLat    *float64 `json:"goods_lat"`
	GoodsLng    *float64 `json:"goods_lng"`
}

// AdminUpdateGoodReq 管理端更新商品
type AdminUpdateGoodReq struct {
	UserID      *uint     `json:"user_id"`
	SchoolID    *uint     `json:"school_id"`
	Title       *string   `json:"title"`
	Content     *string   `json:"content"`
	GoodsType   *int16    `json:"goods_type"`
	GoodsAddr   *string   `json:"goods_addr"`
	PickupAddr  *string   `json:"pickup_addr"`
	Price       *int      `json:"price"`
	MarkedPrice *int      `json:"marked_price"`
	Stock       *int      `json:"stock"`
	Images      *[]string `json:"images"`
	GoodStatus  *int      `json:"good_status"`
	Status      *int16    `json:"status"` // 1正常 2禁用
	GoodsLat    *float64  `json:"goods_lat"`
	GoodsLng    *float64  `json:"goods_lng"`
}

func (s *goodService) AdminCreate(ctx *gin.Context, req AdminCreateGoodReq) (uint, error) {
	if _, err := dao.User().GetByID(ctx.Request.Context(), req.UserID); err != nil {
		return 0, errors.New("用户不存在")
	}
	if _, err := dao.School().GetByID(ctx.Request.Context(), req.SchoolID); err != nil {
		return 0, errors.New("学校不存在")
	}
	if req.Price < 0 {
		req.Price = 0
	}
	if req.Stock < 0 {
		req.Stock = 0
	}
	gs := req.GoodStatus
	if gs != dao.GoodStatusOnSale && gs != dao.GoodStatusOffShelf && gs != dao.GoodStatusSold {
		gs = dao.GoodStatusOffShelf
	}
	gt := req.GoodsType
	if gt != constant.GoodsTypeDelivery && gt != constant.GoodsTypePickup && gt != constant.GoodsTypeOnline {
		gt = constant.GoodsTypeDelivery
	}
	uid := int(req.UserID)
	sid := int(req.SchoolID)
	addr := strings.TrimSpace(req.GoodsAddr)
	if addr == "" {
		addr = strings.TrimSpace(req.PickupAddr)
	}
	g := &model.Good{
		UserID:      &uid,
		SchoolID:    &sid,
		Title:       req.Title,
		Content:     req.Content,
		GoodsType:   gt,
		GoodsAddr:   addr,
		PickupAddr:  addr,
		Price:       req.Price,
		MarkedPrice: req.MarkedPrice,
		Stock:       req.Stock,
		GoodStatus:  gs,
		Status:      constant.StatusValid,
	}
	if req.GoodsLat != nil && req.GoodsLng != nil {
		g.GoodsLat = req.GoodsLat
		g.GoodsLng = req.GoodsLng
	}
	if len(req.Images) > 0 {
		paths := make([]string, len(req.Images))
		for i, p := range req.Images {
			paths[i] = oss.PathForStorage(p)
		}
		g.Images = pq.StringArray(paths)
		g.ImageCount = len(req.Images)
	}
	return dao.Good().Create(ctx.Request.Context(), g)
}

func (s *goodService) AdminUpdate(ctx *gin.Context, id uint, req AdminUpdateGoodReq) error {
	if _, err := dao.Good().GetByIDAdmin(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrGoodNotFoundOrNoPermission
		}
		return err
	}
	updates := make(map[string]interface{})
	if req.UserID != nil {
		if *req.UserID > 0 {
			if _, err := dao.User().GetByID(ctx.Request.Context(), *req.UserID); err != nil {
				return errors.New("用户不存在")
			}
			updates["user_id"] = int(*req.UserID)
		}
	}
	if req.SchoolID != nil {
		if *req.SchoolID > 0 {
			if _, err := dao.School().GetByID(ctx.Request.Context(), *req.SchoolID); err != nil {
				return errors.New("学校不存在")
			}
			updates["school_id"] = int(*req.SchoolID)
		}
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Price != nil && *req.Price >= 0 {
		updates["price"] = *req.Price
	}
	if req.MarkedPrice != nil && *req.MarkedPrice >= 0 {
		updates["marked_price"] = *req.MarkedPrice
	}
	if req.Stock != nil && *req.Stock >= 0 {
		updates["stock"] = *req.Stock
	}
	if req.Images != nil {
		paths := make([]string, len(*req.Images))
		for i, p := range *req.Images {
			paths[i] = oss.PathForStorage(p)
		}
		updates["images"] = pq.StringArray(paths)
		updates["image_count"] = len(paths)
	}
	if req.GoodStatus != nil {
		g := *req.GoodStatus
		if g == dao.GoodStatusOnSale || g == dao.GoodStatusOffShelf || g == dao.GoodStatusSold {
			updates["good_status"] = g
		}
	}
	if req.GoodsType != nil {
		gt := *req.GoodsType
		if gt == constant.GoodsTypeDelivery || gt == constant.GoodsTypePickup || gt == constant.GoodsTypeOnline {
			updates["goods_type"] = gt
		}
	}
	if req.GoodsAddr != nil || req.PickupAddr != nil {
		addr := ""
		if req.GoodsAddr != nil {
			addr = strings.TrimSpace(*req.GoodsAddr)
		}
		if addr == "" && req.PickupAddr != nil {
			addr = strings.TrimSpace(*req.PickupAddr)
		}
		updates["goods_addr"] = addr
		updates["pickup_addr"] = addr
	}
	if req.GoodsLat != nil && req.GoodsLng != nil {
		updates["goods_lat"] = *req.GoodsLat
		updates["goods_lng"] = *req.GoodsLng
	}
	if req.Status != nil && (*req.Status == constant.StatusValid || *req.Status == constant.StatusInvalid) {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, updates)
}

func (s *goodService) AdminPublish(ctx *gin.Context, id uint) error {
	if _, err := dao.Good().GetByIDAdmin(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrGoodNotFoundOrNoPermission
		}
		return err
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, map[string]interface{}{"good_status": dao.GoodStatusOnSale})
}

func (s *goodService) AdminOffShelf(ctx *gin.Context, id uint) error {
	if _, err := dao.Good().GetByIDAdmin(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrGoodNotFoundOrNoPermission
		}
		return err
	}
	return dao.Good().UpdateColumns(ctx.Request.Context(), id, map[string]interface{}{"good_status": dao.GoodStatusOffShelf})
}

func (s *goodService) AdminUploadImages(ctx *gin.Context, id uint, files []*multipart.FileHeader) ([]string, error) {
	if _, err := dao.Good().GetByIDAdmin(ctx.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errno.ErrGoodNotFoundOrNoPermission
		}
		return nil, err
	}
	return s.uploadGoodImages(ctx, id, files)
}
