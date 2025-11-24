package httpserver

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"GUGUMint/internal/service"
)

type MintRequestBody struct {
	Hash    string `json:"hash" binding:"required"`
	Address string `json:"address" binding:"required"`
}

func NewRouter(svc *service.MintService) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	r.POST("/api/mint", func(c *gin.Context) {
		var body MintRequestBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sig, err := svc.ProcessMint(c.Request.Context(), body.Hash, body.Address)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"hash":    sig.Hash,
			"address": sig.Address,
			"v":       sig.V,
			"r":       sig.R,
			"s":       sig.S,
		})
	})

	return r
}
