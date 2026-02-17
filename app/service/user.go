package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/util"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
}

func User() *userService {
	return &userService{}
}

func (s *userService) Register(ctx *gin.Context, username, password string) error {
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return dao.User().Create(ctx, &model.User{
		Username: username,
		Password: string(passwordHash),
	})
}

func (s *userService) Login(ctx *gin.Context, username, password string) (token string, err error) {
	user, err := dao.User().GetByUsername(ctx, username)
	if err != nil {
		return token, errors.New("用户名或密码错误")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", errors.New("用户名或密码错误")
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

func (s *userService) Info(ctx *gin.Context, userID uint) (*model.User, error) {
	return dao.User().GetByID(ctx, userID)
}

func (s *userService) BindSchool(ctx *gin.Context, schoolId uint) (err error) {
	return dao.User().UpdateColumn(ctx, "school_id", schoolId)
}

func (s *userService) Update(ctx *gin.Context, user *model.User) error {
	return dao.User().Update(ctx, user)
}
