package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

type UserCertStore struct{}

func (s *UserCertStore) Upsert(ctx context.Context, cert *model.UserCert) error {
	if cert.Status == 0 {
		cert.Status = constant.StatusValid
	}
	return pgsql.DB.WithContext(ctx).Where("user_id = ? AND school_id = ?", cert.UserID, cert.SchoolID).
		Assign(map[string]interface{}{"cert_info": cert.CertInfo, "status": cert.Status}).
		FirstOrCreate(cert).Error
}

func (s *UserCertStore) GetByUserAndSchool(ctx context.Context, userID, schoolID uint) (*model.UserCert, error) {
	cert := &model.UserCert{}
	err := pgsql.DB.WithContext(ctx).Where("user_id = ? AND school_id = ? AND status = ?", userID, schoolID, constant.StatusValid).First(cert).Error
	return cert, err
}

func (s *UserCertStore) UpdateStatus(ctx context.Context, userID, schoolID uint, status int16) error {
	return pgsql.DB.WithContext(ctx).Model(&model.UserCert{}).
		Where("user_id = ? AND school_id = ?", userID, schoolID).
		Update("status", status).Error
}
