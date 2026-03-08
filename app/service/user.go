package service

import (
	"errors"
	"mime/multipart"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
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
			Role:     constant.RoleUser,
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

// GetProfile 获取指定用户的公开身份信息，仅限 status=1 的正常用户
func (s *userService) GetProfile(ctx *gin.Context, id uint) (*response.UserProfile, error) {
	user, err := dao.User().GetByIDIfValid(ctx.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	return &response.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		Avatar:      oss.ToFullURL(user.Avatar),
		Background:  oss.ToFullURL(user.Background),
		FollowCount: user.FollowCount,
		FansCount:   user.FansCount,
		CreatedAt:   user.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *userService) Info(ctx *gin.Context, userID uint) (*response.UserInfo, error) {
	userDao, err := dao.User().GetByID(ctx.Request.Context(), userID)
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
		BindQQ:      userDao.BindQQ,
		BindWX:      userDao.BindWX,
		BindPhone:   userDao.BindPhone,
		CreatedAt:   userDao.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   userDao.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if userDao.SchoolID > 0 {
		if school, err := dao.School().GetByID(ctx.Request.Context(), userDao.SchoolID); err == nil && school != nil && school.Name != nil {
			userInfo.SchoolName = *school.Name
		}
	}
	return userInfo, nil
}

// BindSchoolReq 绑定学校请求（需学校端账号密码验证）
type BindSchoolReq struct {
	SchoolID uint   `json:"school_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *userService) BindSchool(ctx *gin.Context, userID uint, req BindSchoolReq) error {
	school, err := dao.School().GetByIDValid(ctx.Request.Context(), req.SchoolID)
	if err != nil {
		return errors.New("学校不存在")
	}
	if school.Code == nil || *school.Code == "" {
		return errors.New("该学校暂不支持在线认证绑定")
	}

	res, err := schools.Login(ctx.Request.Context(), *school.Code, req.Username, req.Password)
	if err != nil {
		return err
	}
	if !res.Success {
		return errors.New(res.Message)
	}

	certInfo := make(map[string]interface{})
	if res.CertInfo != nil {
		certInfo = res.CertInfo
	}
	certInfo["student_id"] = res.StudentID
	if res.Name != "" {
		certInfo["name"] = res.Name
	}

	cert := &model.UserCert{
		UserID:   userID,
		SchoolID: req.SchoolID,
		CertInfo: certInfo,
	}
	if err := dao.UserCert().Upsert(ctx.Request.Context(), cert); err != nil {
		return err
	}
	return dao.User().UpdateSchoolByID(ctx.Request.Context(), userID, req.SchoolID)
}

// UpdateProfileReq 用户更新资料请求（仅允许更新这些字段，不含 username/role/status/password）
type UpdateProfileReq struct {
	Avatar     *string `json:"avatar"`
	Background *string `json:"background"`
	BindQQ     *string `json:"bind_qq"`
	BindWX     *string `json:"bind_wx"`
	BindPhone  *string `json:"bind_phone"`
}

// UpdateProfile 根据当前登录用户 ID 部分更新资料
func (s *userService) UpdateProfile(ctx *gin.Context, userID uint, req UpdateProfileReq) error {
	updates := make(map[string]interface{})
	if req.Avatar != nil {
		storagePath, err := oss.ExtractUserAssetPath(*req.Avatar, userID)
		if err != nil {
			return errors.New("头像路径无效或无权使用")
		}
		updates["avatar"] = oss.PathForStorage(storagePath)
	}
	if req.Background != nil {
		storagePath, err := oss.ExtractUserAssetPath(*req.Background, userID)
		if err != nil {
			return errors.New("背景路径无效或无权使用")
		}
		updates["background"] = oss.PathForStorage(storagePath)
	}
	if req.BindQQ != nil {
		v := strings.TrimSpace(*req.BindQQ)
		if len(v) > 128 {
			return errors.New("QQ 长度不能超过 128 字符")
		}
		updates["bind_qq"] = v
	}
	if req.BindWX != nil {
		v := strings.TrimSpace(*req.BindWX)
		if len(v) > 128 {
			return errors.New("微信长度不能超过 128 字符")
		}
		updates["bind_wx"] = v
	}
	if req.BindPhone != nil {
		v := strings.TrimSpace(*req.BindPhone)
		if len(v) > 20 {
			return errors.New("手机号长度不能超过 20 字符")
		}
		updates["bind_phone"] = v
	}
	if len(updates) == 0 {
		return nil
	}
	return dao.User().UpdateColumns(ctx.Request.Context(), userID, updates)
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
