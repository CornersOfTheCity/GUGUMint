package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"GUGUMint/internal/service"
)

type MintRequestBody struct {
	Hash    string `json:"hash" binding:"required"`
	Address string `json:"address" binding:"required"`
}

func NewRouter(svc *service.MintService) *gin.Engine {
	r := gin.Default()

	r.POST("/api/mint", func(c *gin.Context) {
		var body MintRequestBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		txHash, err := svc.ProcessMint(c.Request.Context(), body.Hash, body.Address)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"txHash": txHash,
			"status": "ok",
		})
	})

	return r
}
