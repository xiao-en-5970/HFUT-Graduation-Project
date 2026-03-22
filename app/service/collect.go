package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"gorm.io/gorm"
)

var (
	ErrCollectFolderNotFound   = errors.New("收藏夹不存在")
	ErrCollectFolderNotOwned   = errors.New("无权限操作该收藏夹")
	ErrCollectArticleNotFound  = errors.New("文章不存在")
	ErrCollectAlreadyCollected = errors.New("已收藏")
	ErrCollectNotCollected     = errors.New("未收藏")
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

// AddArticle 收藏到收藏夹
// collectID=0 表示默认收藏夹；extType 标明收藏类型（1帖子 2提问 3回答 4商品），便于筛选展示
func (s *collectService) AddArticle(ctx *gin.Context, userID uint, schoolID uint, collectID uint, articleID uint, extType int) error {
	cid, err := s.resolveCollectID(ctx, userID, collectID)
	if err != nil {
		return err
	}
	if extType == constant.ExtTypeGoods {
		return s.addGoodCollect(ctx, userID, schoolID, cid, articleID)
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
	// 惰性新建：若存在 status=2 的记录则恢复，否则新建；已收藏则幂等返回成功
	exist, getErr := dao.CollectItem().GetByCollectExt(ctx.Request.Context(), cid, int(articleID), extType)
	if getErr == nil {
		if exist.Status == constant.StatusValid {
			return nil // 幂等：已收藏，多次收藏视为成功
		}
		// status=2，恢复并更新计数
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.CollectItem().RestoreWithDB(tx, cid, int(articleID), extType); err != nil {
				return err
			}
			return dao.Article().UpdateCollectCountDB(tx, articleID, 1)
		})
	}
	if getErr != gorm.ErrRecordNotFound {
		return getErr
	}
	// 记录不存在，新建（唯一约束防止并发重复，若冲突则幂等返回）
	item := &model.CollectItem{
		CollectID: cid,
		ExtID:     int(articleID),
		ExtType:   extType,
		Status:    constant.StatusValid,
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.CollectItem().CreateWithDB(tx, item); err != nil {
			return err
		}
		return dao.Article().UpdateCollectCountDB(tx, articleID, 1)
	})
	if err != nil {
		// 并发时可能触发唯一约束冲突，此时记录已存在，视为幂等成功
		exist2, _ := dao.CollectItem().GetByCollectExt(ctx.Request.Context(), cid, int(articleID), extType)
		if exist2 != nil && exist2.Status == constant.StatusValid {
			return nil
		}
		return err
	}
	return nil
}

// addGoodCollect 收藏商品
func (s *collectService) addGoodCollect(ctx *gin.Context, userID uint, schoolID uint, collectID uint, goodID uint) error {
	g, err := dao.Good().GetByIDWithSchool(ctx.Request.Context(), goodID, schoolID)
	if err != nil || g == nil {
		return ErrCollectArticleNotFound
	}
	exist, getErr := dao.CollectItem().GetByCollectExt(ctx.Request.Context(), collectID, int(goodID), constant.ExtTypeGoods)
	if getErr == nil {
		if exist.Status == constant.StatusValid {
			return nil
		}
		return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := dao.CollectItem().RestoreWithDB(tx, collectID, int(goodID), constant.ExtTypeGoods); err != nil {
				return err
			}
			return dao.Good().UpdateCollectCountDB(tx, goodID, 1)
		})
	}
	if getErr != gorm.ErrRecordNotFound {
		return getErr
	}
	item := &model.CollectItem{
		CollectID: collectID,
		ExtID:     int(goodID),
		ExtType:   constant.ExtTypeGoods,
		Status:    constant.StatusValid,
	}
	err = pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.CollectItem().CreateWithDB(tx, item); err != nil {
			return err
		}
		return dao.Good().UpdateCollectCountDB(tx, goodID, 1)
	})
	if err != nil {
		exist2, _ := dao.CollectItem().GetByCollectExt(ctx.Request.Context(), collectID, int(goodID), constant.ExtTypeGoods)
		if exist2 != nil && exist2.Status == constant.StatusValid {
			return nil
		}
		return err
	}
	return nil
}

// RemoveArticle 取消收藏
func (s *collectService) RemoveArticle(ctx *gin.Context, userID uint, collectID uint, articleID uint, extType int) error {
	cid, err := s.resolveCollectID(ctx, userID, collectID)
	if err != nil {
		return err
	}
	ok, _ := dao.CollectItem().Exists(ctx.Request.Context(), cid, int(articleID), extType)
	if !ok {
		return nil // 幂等：未收藏，多次取消视为成功
	}
	updateCount := func(tx *gorm.DB) error {
		if extType == constant.ExtTypeGoods {
			return dao.Good().UpdateCollectCountDB(tx, articleID, -1)
		}
		return dao.Article().UpdateCollectCountDB(tx, articleID, -1)
	}
	return pgsql.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := dao.CollectItem().SoftDeleteWithDB(tx, cid, int(articleID), extType); err != nil {
			return err
		}
		return updateCount(tx)
	})
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
