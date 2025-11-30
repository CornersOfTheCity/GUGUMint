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

type MintTxBody struct {
	Hash    string `json:"hash" binding:"required"`
	Address string `json:"address" binding:"required"`
	TxHash  string `json:"txHash" binding:"required"`
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

	r.POST("/api/mint/tx", func(c *gin.Context) {
		var body MintTxBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mr, err := svc.SaveTxHash(c.Request.Context(), body.Hash, body.Address, body.TxHash)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"hash":    mr.Hash,
			"address": mr.Address,
			"txHash":  mr.TxHash,
			"status":  mr.Status,
		})
	})

	r.GET("/api/mint/status", func(c *gin.Context) {
		txHash := c.Query("txHash")
		if txHash == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "txHash is required"})
			return
		}

		mr, err := svc.GetStatusByTxHash(c.Request.Context(), txHash)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"hash":      mr.Hash,
			"address":   mr.Address,
			"txHash":    mr.TxHash,
			"status":    mr.Status,
			"updatedAt": mr.UpdatedAt,
		})
	})

	return r
}
