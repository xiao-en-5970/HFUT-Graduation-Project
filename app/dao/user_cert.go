package dao

import (
	"context"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
)

type UserCertStore struct{}

func (s *UserCertStore) Upsert(ctx context.Context, cert *model.UserCert) error {
	return pgsql.DB.WithContext(ctx).Where("user_id = ? AND school_id = ?", cert.UserID, cert.SchoolID).
		Assign(map[string]interface{}{"cert_info": cert.CertInfo}).
		FirstOrCreate(cert).Error
}

func (s *UserCertStore) GetByUserAndSchool(ctx context.Context, userID, schoolID uint) (*model.UserCert, error) {
	cert := &model.UserCert{}
	err := pgsql.DB.WithContext(ctx).Where("user_id = ? AND school_id = ?", userID, schoolID).First(cert).Error
	return cert, err
}
