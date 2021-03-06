package service

import (
	"github.com/Glintt/gAPI/api/database"
	userModels "github.com/Glintt/gAPI/api/users/models"
)

type ServiceRepositoryInterface interface {
	Update(service Service, serviceExists Service) (int, error)
	CreateService(s Service) (Service, error)
	ListServices(page int, filterQuery string) []Service
	DeleteService(s Service) error
	Find(s Service) (Service, error)
	ListAllAvailableHosts() ([]string, error)
	NormalizeServices() error
}

func GetServicesRepository(user userModels.User) ServiceRepositoryInterface {
	if database.SD_TYPE == "mongo" {
		return &ServiceMongoRepository{
			User: user,
		}
	}
	if database.SD_TYPE == "oracle" {
		return &ServiceOracleRepository{
			User: user,
		}
	}
	return nil
}
