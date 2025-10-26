package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thebearodactyl/apiodactyl/internal/database"
	"github.com/thebearodactyl/apiodactyl/internal/middleware"
	"github.com/thebearodactyl/apiodactyl/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db              *database.DB
	jwtSecret       string
	expirationHours int
}

func NewAuthHandler(db *database.DB, jwtSecret string, expirationHours int) *AuthHandler {
	return &AuthHandler{
		db:              db,
		jwtSecret:       jwtSecret,
		expirationHours: expirationHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	query := `INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?) RETURNING id`
	var userID int64
	err = h.db.QueryRowContext(c.Request.Context(), query, req.Username, req.Email, string(hashedPassword), models.RoleNormal).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	token, expiresAt, err := middleware.GenerateToken(userID, req.Username, models.RoleNormal, h.jwtSecret, h.expirationHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, models.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: models.UserInfo{
			ID:       userID,
			Username: req.Username,
			Email:    req.Email,
			Role:     models.RoleNormal,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `SELECT id, username, email, password_hash, role FROM users WHERE username = ?`
	var user models.User
	err := h.db.QueryRowContext(c.Request.Context(), query, req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, expiresAt, err := middleware.GenerateToken(user.ID, user.Username, user.Role, h.jwtSecret, h.expirationHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: models.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := `SELECT id, username, email, role, created_at FROM users WHERE id = ?`
	var user models.User
	err := h.db.QueryRowContext(c.Request.Context(), query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	c.JSON(http.StatusOK, user)
}
