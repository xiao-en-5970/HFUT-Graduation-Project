package service

import (
	"errors"
	"mime/multipart"

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
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	Price       int      `json:"price"`        // 价格（分）
	MarkedPrice int      `json:"marked_price"` // 标价（分）
	Stock       int      `json:"stock"`        // 库存
	Images      []string `json:"images"`       // 图片 URL 列表
}

type UpdateGoodReq struct {
	Title       *string   `json:"title"`
	Content     *string   `json:"content"`
	Price       *int      `json:"price"`
	MarkedPrice *int      `json:"marked_price"`
	Stock       *int      `json:"stock"`
	Images      *[]string `json:"images"`
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
	uid := int(userID)
	sid := int(schoolID)
	g := &model.Good{
		UserID:      &uid,
		SchoolID:    &sid,
		Title:       req.Title,
		Content:     req.Content,
		Price:       req.Price,
		MarkedPrice: req.MarkedPrice,
		Stock:       req.Stock,
		GoodStatus:  dao.GoodStatusOffShelf, // 新建为下架，需上架后才可见
		Status:      constant.StatusValid,
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

func (s *goodService) List(ctx *gin.Context, schoolID uint, page, pageSize int) ([]*model.Good, int64, error) {
	return dao.Good().List(ctx.Request.Context(), schoolID, page, pageSize)
}

func (s *goodService) ListByUserID(ctx *gin.Context, targetUserID uint, viewerSchoolID uint, includeOffShelf bool, page, pageSize int) ([]*model.Good, int64, error) {
	return dao.Good().ListByUserID(ctx.Request.Context(), targetUserID, viewerSchoolID, includeOffShelf, page, pageSize)
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
	Price       int      `json:"price"`
	MarkedPrice int      `json:"marked_price"`
	Stock       int      `json:"stock"`
	Images      []string `json:"images"`
	GoodStatus  int      `json:"good_status"` // 可选 1在售 2下架 3已售出，默认下架
}

// AdminUpdateGoodReq 管理端更新商品
type AdminUpdateGoodReq struct {
	UserID      *uint     `json:"user_id"`
	SchoolID    *uint     `json:"school_id"`
	Title       *string   `json:"title"`
	Content     *string   `json:"content"`
	Price       *int      `json:"price"`
	MarkedPrice *int      `json:"marked_price"`
	Stock       *int      `json:"stock"`
	Images      *[]string `json:"images"`
	GoodStatus  *int      `json:"good_status"`
	Status      *int16    `json:"status"` // 1正常 2禁用
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
	uid := int(req.UserID)
	sid := int(req.SchoolID)
	g := &model.Good{
		UserID:      &uid,
		SchoolID:    &sid,
		Title:       req.Title,
		Content:     req.Content,
		Price:       req.Price,
		MarkedPrice: req.MarkedPrice,
		Stock:       req.Stock,
		GoodStatus:  gs,
		Status:      constant.StatusValid,
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
