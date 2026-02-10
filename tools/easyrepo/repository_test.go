package easyrepo

import (
	"testing"

	"github.com/raywall/fast-service-lab/tools/dyndb"
	"github.com/raywall/fast-service-lab/tools/easyrepo/models"
	"github.com/stretchr/testify/assert"
)

func TestNewRepository(t *testing.T) {
	config := dyndb.TableConfig[models.TestItem]{
		TableName: "TestTable",
		HashKey:   "ID",
	}

	repo := NewRepository(nil, config)

	assert.NotNil(t, repo)
	assert.Equal(t, "TestTable", repo.Config.TableName)
	assert.NotNil(t, repo.Store)
}
