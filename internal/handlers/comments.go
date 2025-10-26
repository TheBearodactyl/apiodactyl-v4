package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thebearodactyl/apiodactyl/internal/database"
	"github.com/thebearodactyl/apiodactyl/internal/models"
)

type CommentHandler struct {
	db *database.DB
}

func NewCommentHandler(db *database.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.GameID == nil && req.BookID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either game_id or book_id must be provided"})
		return
	}

	if req.GameID != nil && req.BookID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot comment on both game and book at the same time"})
		return
	}

	userID, _ := c.Get("user_id")

	query := `INSERT INTO comments (content, game_id, book_id, user_id) VALUES (?, ?, ?, ?) RETURNING id, created_at, updated_at`
	var id int64
	var createdAt, updatedAt string
	err := h.db.QueryRowContext(c.Request.Context(), query, req.Content, req.GameID, req.BookID, userID).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"created_at": createdAt,
		"updated_at": updatedAt,
		"message":    "Comment created successfully",
	})
}

func (h *CommentHandler) GetComments(c *gin.Context) {
	gameID := c.Query("game_id")
	bookID := c.Query("book_id")

	if gameID == "" && bookID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either game_id or book_id query parameter is required"})
		return
	}

	var query string
	var args []any

	if gameID != "" {
		query = `
			SELECT c.id, c.content, c.game_id, c.book_id, c.user_id, u.username, c.created_at, c.updated_at
			FROM comments c
			JOIN users u ON c.user_id = u.id
			WHERE c.game_id = ?
			ORDER BY c.created_at DESC
		`
		args = append(args, gameID)
	} else {
		query = `
			SELECT c.id, c.content, c.game_id, c.book_id, c.user_id, u.username, c.created_at, c.updated_at
			FROM comments c
			JOIN users u ON c.user_id = u.id
			WHERE c.book_id = ?
			ORDER BY c.created_at DESC
		`
		args = append(args, bookID)
	}

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer rows.Close()

	comments := []models.Comment{}
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.Content, &comment.GameID, &comment.BookID,
			&comment.UserID, &comment.Username, &comment.CreatedAt, &comment.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan comment"})
			return
		}
		comments = append(comments, comment)
	}

	c.JSON(http.StatusOK, comments)
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req models.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	var query string
	var args []any

	if userRole == models.RoleAdmin {
		query = `UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []any{req.Content, id}
	} else {
		query = `UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`
		args = []any{req.Content, id, userID}
	}

	result, err := h.db.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found or you don't have permission to update it"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment updated successfully"})
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	var query string
	var args []any

	if userRole == models.RoleAdmin {
		query = `DELETE FROM comments WHERE id = ?`
		args = []any{id}
	} else {
		query = `DELETE FROM comments WHERE id = ? AND user_id = ?`
		args = []any{id, userID}
	}

	result, err := h.db.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found or you don't have permission to delete it"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}
