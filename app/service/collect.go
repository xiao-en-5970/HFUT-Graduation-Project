package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

var (
	ErrCollectFolderNotFound   = errors.New("收藏夹不存在")
	ErrCollectFolderNotOwned   = errors.New("无权限操作该收藏夹")
	ErrCollectArticleNotFound  = errors.New("文章不存在")
	ErrCollectAlreadyCollected = errors.New("已收藏")
)

type collectService struct{}

// GetOrCreateDefaultFolder 获取或创建用户的默认收藏夹
// collectID=0 表示使用默认收藏夹
func (s *collectService) GetOrCreateDefaultFolder(ctx *gin.Context, userID uint) (uint, error) {
	folder, err := dao.Collect().GetDefaultByUserID(ctx.Request.Context(), userID)
	if err != nil {
		return 0, err
	}
	if folder != nil {
		return folder.ID, nil
	}
	uid := int(userID)
	c := &model.Collect{
		UserID:    &uid,
		Name:      "默认",
		IsDefault: true,
		Status:    constant.StatusValid,
	}
	return dao.Collect().Create(ctx.Request.Context(), c)
}

// CreateFolder 创建收藏夹
func (s *collectService) CreateFolder(ctx *gin.Context, userID uint, name string) (uint, error) {
	uid := int(userID)
	c := &model.Collect{
		UserID:    &uid,
		Name:      name,
		IsDefault: false,
		Status:    constant.StatusValid,
	}
	return dao.Collect().Create(ctx.Request.Context(), c)
}

// ListFolders 列出用户收藏夹
func (s *collectService) ListFolders(ctx *gin.Context, userID uint) ([]*model.Collect, error) {
	return dao.Collect().ListByUserID(ctx.Request.Context(), userID)
}

// resolveCollectID collectID=0 时解析为默认收藏夹
func (s *collectService) resolveCollectID(ctx *gin.Context, userID uint, collectID uint) (uint, error) {
	if collectID > 0 {
		folder, err := dao.Collect().GetByIDAndUser(ctx.Request.Context(), collectID, userID)
		if err != nil || folder == nil {
			return 0, ErrCollectFolderNotFound
		}
		return collectID, nil
	}
	return s.GetOrCreateDefaultFolder(ctx, userID)
}

// AddArticle 收藏文章到收藏夹
// collectID=0 表示默认收藏夹；extType 标明收藏类型（1帖子 2提问 3回答），便于筛选展示
func (s *collectService) AddArticle(ctx *gin.Context, userID uint, schoolID uint, collectID uint, articleID uint, extType int) error {
	cid, err := s.resolveCollectID(ctx, userID, collectID)
	if err != nil {
		return err
	}
	art, err := dao.Article().GetByIDWithSchoolAndType(ctx.Request.Context(), articleID, schoolID, extType)
	if err != nil || art == nil {
		return ErrCollectArticleNotFound
	}
	if art.PublishStatus == 1 {
		ok, _ := dao.Article().ExistsAndOwnedByWithSchoolAndType(ctx.Request.Context(), articleID, userID, schoolID, extType)
		if !ok {
			return ErrCollectArticleNotFound
		}
	}
	ok, _ := dao.CollectItem().Exists(ctx.Request.Context(), cid, int(articleID), extType)
	if ok {
		return ErrCollectAlreadyCollected
	}
	item := &model.CollectItem{
		CollectID: cid,
		ExtID:     int(articleID),
		ExtType:   extType,
		Status:    constant.StatusValid,
	}
	_, err = dao.CollectItem().Create(ctx.Request.Context(), item)
	if err != nil {
		return err
	}
	// 更新文章 collect_count
	return dao.Article().UpdateCollectCount(ctx.Request.Context(), articleID, 1)
}

// RemoveArticle 取消收藏文章
func (s *collectService) RemoveArticle(ctx *gin.Context, userID uint, collectID uint, articleID uint, extType int) error {
	cid, err := s.resolveCollectID(ctx, userID, collectID)
	if err != nil {
		return err
	}
	err = dao.CollectItem().Delete(ctx.Request.Context(), cid, int(articleID), extType)
	if err != nil {
		return err
	}
	return dao.Article().UpdateCollectCount(ctx.Request.Context(), articleID, -1)
}

// ListItems 列出收藏夹中的收藏项
// extType=0 全部混合展示；>0 按类型筛选（1帖子 2提问 3回答 4商品）
func (s *collectService) ListItems(ctx *gin.Context, userID uint, collectID uint, extType int, page, pageSize int) ([]*model.CollectItem, int64, error) {
	cid, err := s.resolveCollectID(ctx, userID, collectID)
	if err != nil {
		return nil, 0, err
	}
	return dao.CollectItem().ListByCollect(ctx.Request.Context(), cid, extType, page, pageSize)
}
