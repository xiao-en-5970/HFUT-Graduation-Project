package service

import (
	"errors"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userService struct{}

func (s *userService) Register(ctx *gin.Context, username, password string) (uint, error) {
	_, err := dao.User().GetByUsername(ctx, username)
	// 用户不存在才能注册
	if errors.Is(err, gorm.ErrRecordNotFound) {
		passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		return dao.User().Create(ctx, &model.User{
			SchoolID: 0,
			Username: username,
			Password: string(passwordHash),
		})
	} else {
		logger.Error(ctx, "用户已存在", zap.Error(err))
		return 0, errors.New("用户已存在")
	}

}

func (s *userService) Login(ctx *gin.Context, username, password string) (token string, err error) {
	user, err := dao.User().GetByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(ctx, "用户不存在", zap.Error(err))
		return token, errors.New("用户不存在")
	}
	if err != nil {
		logger.Error(ctx, "用户登录失败", zap.Error(err))
		return token, err
	}
	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("密码错误")
	}

	// 检查状态
	if user.Status != constant.StatusValid {
		return "", errors.New("账户已被禁用")
	}

	// 生成 JWT token
	token, err = util.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", errors.New("生成 token 失败: " + err.Error())
	}

	return token, nil
}

// AdminLogin 管理员登录：验证账号密码且 role>=2 才返回 token
func (s *userService) AdminLogin(ctx *gin.Context, username, password string) (token string, err error) {
	user, err := dao.User().GetByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errors.New("用户不存在")
	}
	if err != nil {
		return "", err
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("密码错误")
	}
	if user.Status != constant.StatusValid {
		return "", errors.New("账户已被禁用")
	}
	if user.Role < constant.RoleAdmin {
		return "", errors.New("无管理员权限")
	}
	token, err = util.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", errors.New("生成 token 失败: " + err.Error())
	}
	return token, nil
}

func (s *userService) Info(ctx *gin.Context, userID uint) (*response.UserInfo, error) {
	userDao, err := dao.User().GetByID(ctx, userID)
	if err != nil {
		return &response.UserInfo{}, err
	}
	userInfo := &response.UserInfo{
		ID:          userDao.ID,
		Username:    userDao.Username,
		SchoolID:    userDao.SchoolID,
		Role:        userDao.Role,
		Status:      userDao.Status,
		Avatar:      oss.ToFullURL(userDao.Avatar),
		Background:  oss.ToFullURL(userDao.Background),
		FollowCount: userDao.FollowCount,
		FansCount:   userDao.FansCount,
	}
	return userInfo, nil
}

func (s *userService) BindSchool(ctx *gin.Context, userID uint, schoolId uint) error {
	if schoolId > 0 {
		_, err := dao.School().GetByID(ctx.Request.Context(), schoolId)
		if err != nil {
			return errors.New("学校不存在")
		}
	}
	return dao.User().UpdateSchoolByID(ctx.Request.Context(), userID, schoolId)
}

func (s *userService) Update(ctx *gin.Context, user *model.User) error {
	return dao.User().Update(ctx, user)
}

func (s *userService) UploadAvatar(ctx *gin.Context, userID uint, file *multipart.FileHeader) (url string, err error) {
	ext := oss.ExtFromFilename(file.Filename)
	relPath := oss.UserAvatarPath(userID, ext)
	url, err = oss.Save(file, relPath)
	if err != nil {
		return "", err
	}
	if err := dao.User().UpdateAvatarByID(ctx, userID, oss.PathForStorage(url)); err != nil {
		return "", err
	}
	return url, nil
}

func (s *userService) UploadBackground(ctx *gin.Context, userID uint, file *multipart.FileHeader) (url string, err error) {
	ext := oss.ExtFromFilename(file.Filename)
	relPath := oss.UserBackgroundPath(userID, ext)
	url, err = oss.Save(file, relPath)
	if err != nil {
		return "", err
	}
	if err := dao.User().UpdateBackgroundByID(ctx, userID, oss.PathForStorage(url)); err != nil {
		return "", err
	}
	return url, nil
}
