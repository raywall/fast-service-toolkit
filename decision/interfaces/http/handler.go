// Package http em interfaces fornece handlers
package http

import (
	"github.com/gin-gonic/gin"
	"gitrub.com/raywall/fast-service-toolkit/decision/application"
	"gitrub.com/raywall/fast-service-toolkit/decision/domain"
	"gitrub.com/raywall/fast-service-toolkit/decision/infrastructure/adapter"
)

func SetupRouter(repo domain.ConfigRepository) *gin.Engine {
	r := gin.Default()

	cel, _ := adapter.NewCELAdapter()
	api := adapter.NewAPIAdapter(cel)
	dd := adapter.NewDatadogAdapter(repo.GetConfig().Service.Metrics.Datadog.Addr)
	logA := adapter.NewLogAdapter()

	usecase := application.NewProcessRequestUsecase(dd, repo, cel, api, logA)

	r.POST(repo.GetConfig().Service.Route, func(c *gin.Context) {
		body, _ := c.GetRawData()
		resp, code, err := usecase.Execute(c.Request.Context(), body)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
			return
		}
		c.Data(code, "application/json", resp)
	})

	return r
}