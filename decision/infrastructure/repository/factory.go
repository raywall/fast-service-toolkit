package repository

import (
	"os"

	"github.com/raywall/fast-service-toolkit/decision/domain"
)

func NewConfigRepository() (domain.ConfigRepository, error) {
	if os.Getenv("GOD_DB_TYPE") != ""{
		return NewDBRepository()
	}
	return NewFileRepository(os.Getenv("GOD_CG_FILEPATH"))
}

func NewConfigRepositoryFromFile(filepath string) (domain.ConfigRepository, error) {
	return NewFileRepository(filepath)
}