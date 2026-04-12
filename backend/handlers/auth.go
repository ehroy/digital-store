package handlers

import (
	"digistore/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username dan password wajib diisi"})
		return
	}

	if req.Username != config.App.AdminUser || req.Password != config.App.AdminPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "username atau password salah"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  req.Username,
		"role": "admin",
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(config.App.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    signed,
		"username": req.Username,
		"expires":  time.Now().Add(24 * time.Hour).Unix(),
	})
}
