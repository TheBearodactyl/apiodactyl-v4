package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thebearodactyl/apiodactyl/internal/database"
	"github.com/thebearodactyl/apiodactyl/internal/models"
	"github.com/thebearodactyl/apiodactyl/internal/utils"
)

type GameHandler struct {
	db *database.DB
}

func NewGameHandler(db *database.DB) *GameHandler {
	return &GameHandler{db: db}
}

func (h *GameHandler) saveCoverImage(c *gin.Context, fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("open error: %v", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash error: %v", err)
	}
	hashSum := hex.EncodeToString(hasher.Sum(nil))
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	file.Seek(0, io.SeekStart)

	saveDir := "./files"
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return "", fmt.Errorf("dir create error: %v", err)
	}

	hashFilename := hashSum + ext
	savePath := filepath.Join(saveDir, hashFilename)

	if _, err := os.Stat(savePath); err == nil {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		permalink := fmt.Sprintf("%s://%s/files/%s", scheme, host, hashFilename)
		return permalink, nil
	}

	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("save error: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", fmt.Errorf("write error: %v", err)
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	permalink := fmt.Sprintf("%s://%s/files/%s", scheme, host, hashFilename)
	return permalink, nil
}

func (h *GameHandler) saveCoverFromURL(c *gin.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: 30 & time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GameHandler/1.0)")
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GameHandler/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", fmt.Errorf("read body error: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	exts, _ := mime.ExtensionsByType(contentType)
	ext := ".bin"
	if len(exts) > 0 {
		ext = exts[0]
	}

	hasher := sha256.New()
	hasher.Write(buf.Bytes())
	hashSum := hex.EncodeToString(hasher.Sum(nil))

	saveDir := "./files"
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	hashFilename := hashSum + ext
	savePath := filepath.Join(saveDir, hashFilename)

	if _, err := os.Stat(savePath); err == nil {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s/files/%s", scheme, c.Request.Host, hashFilename), nil
	}

	if err := os.WriteFile(savePath, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("save error: %v", err)
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/files/%s", scheme, c.Request.Host, hashFilename), nil
}

func (h *GameHandler) GetGames(c *gin.Context) {
	userID, _ := c.Get("user_id")

	query := `
		SELECT id, title, developer, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, percent, bad, user_id, 
		       created_at, updated_at 
		FROM games 
		WHERE user_id = ? 
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}
	defer rows.Close()

	games := []models.Game{}
	for rows.Next() {
		var g models.Game
		var genresJSON, tagsJSON string
		if err := rows.Scan(&g.ID, &g.Title, &g.Developer, &genresJSON, &tagsJSON,
			&g.Rating, &g.Status, &g.Description, &g.MyThoughts, &g.CoverImage,
			&g.Explicit, &g.Color, &g.Percent, &g.Bad, &g.UserID,
			&g.CreatedAt, &g.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan game"})
			return
		}

		json.Unmarshal([]byte(genresJSON), &g.Genres)
		json.Unmarshal([]byte(tagsJSON), &g.Tags)

		games = append(games, g)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating games"})
		return
	}

	for i := range games {
		links, err := h.getGameLinks(c, games[i].ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch game links"})
			return
		}
		games[i].Links = links
	}

	c.JSON(http.StatusOK, games)
}

func (h *GameHandler) GetGame(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")

	query := `
		SELECT id, title, developer, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, percent, bad, user_id, 
		       created_at, updated_at 
		FROM games 
		WHERE id = ? AND user_id = ?
	`

	var g models.Game
	var genresJSON, tagsJSON string
	err = h.db.QueryRowContext(c.Request.Context(), query, id, userID).Scan(
		&g.ID, &g.Title, &g.Developer, &genresJSON, &tagsJSON,
		&g.Rating, &g.Status, &g.Description, &g.MyThoughts, &g.CoverImage,
		&g.Explicit, &g.Color, &g.Percent, &g.Bad, &g.UserID,
		&g.CreatedAt, &g.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.GenErr("Failed to fetch game", err))
		return
	}

	json.Unmarshal([]byte(genresJSON), &g.Genres)
	json.Unmarshal([]byte(tagsJSON), &g.Tags)

	links, err := h.getGameLinks(c, g.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.GenErr("Failed to fetch game links", err))
		return
	}
	g.Links = links

	c.JSON(http.StatusOK, g)
}

func (h *GameHandler) CreateGame(c *gin.Context) {
	var req models.CreateGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.GenErr("Bad request body", err))
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, utils.GenErr("Unauthorized", fmt.Errorf("no user id in context")))
		return
	}

	var coverImageURL string
	var err error

	// Check if a file was uploaded
	fileHeader, fileErr := c.FormFile("cover_image")
	if fileErr == nil && fileHeader != nil {
		coverImageURL, err = h.saveCoverImage(c, fileHeader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.CoverImageURL != "" {
		// If no file uploaded, check for URL in JSON
		coverImageURL, err = h.saveCoverFromURL(c, req.CoverImageURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to download cover image: %v", err)})
			return
		}
	} else if req.CoverImage != "" {
		// Use provided cover_image string directly
		coverImageURL = req.CoverImage
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cover_image, cover_image_url, or cover_image file upload is required"})
		return
	}

	genresJSON, _ := json.Marshal(req.Genres)
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO games (title, developer, genres, tags, rating, status, description, 
		                   my_thoughts, cover_image, explicit, color, percent, bad, user_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
		RETURNING id, created_at, updated_at
	`

	var id int64
	var createdAt, updatedAt string
	err = h.db.QueryRowContext(c.Request.Context(), query,
		req.Title, req.Developer, string(genresJSON), string(tagsJSON),
		req.Rating, req.Status, req.Description, req.MyThoughts,
		coverImageURL, req.Explicit, req.Color, req.Percent, req.Bad, userID,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.GenErr("Failed to create game", err))
		return
	}

	if err := h.insertGameLinks(c, id, req.Links); err != nil {
		c.JSON(http.StatusInternalServerError, utils.GenErr("Failed to create game links", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"cover_image": coverImageURL,
		"message":     "Game created successfully",
	})
}

func (h *GameHandler) UpdateGame(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.GenErr("Invalid ID", err))
		return
	}

	var req models.UpdateGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.GenErr("Invalid JSON", err))
		return
	}

	userID, _ := c.Get("user_id")

	updates := []string{}
	args := []any{}

	if req.Title != "" {
		updates = append(updates, "title = ?")
		args = append(args, req.Title)
	}
	if req.Developer != "" {
		updates = append(updates, "developer = ?")
		args = append(args, req.Developer)
	}
	if req.Genres != nil {
		genresJSON, _ := json.Marshal(req.Genres)
		updates = append(updates, "genres = ?")
		args = append(args, string(genresJSON))
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		updates = append(updates, "tags = ?")
		args = append(args, string(tagsJSON))
	}
	if req.Rating > 0 {
		updates = append(updates, "rating = ?")
		args = append(args, req.Rating)
	}
	if req.Status != "" {
		updates = append(updates, "status = ?")
		args = append(args, req.Status)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.MyThoughts != "" {
		updates = append(updates, "my_thoughts = ?")
		args = append(args, req.MyThoughts)
	}
	if req.CoverImage != "" {
		updates = append(updates, "cover_image = ?")
		args = append(args, req.CoverImage)
	}
	if req.Explicit != nil {
		updates = append(updates, "explicit = ?")
		args = append(args, *req.Explicit)
	}
	if req.Color != "" {
		updates = append(updates, "color = ?")
		args = append(args, req.Color)
	}
	if req.Percent >= 0 {
		updates = append(updates, "percent = ?")
		args = append(args, req.Percent)
	}
	if req.Bad != nil {
		updates = append(updates, "bad = ?")
		args = append(args, *req.Bad)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id, userID)

	query := fmt.Sprintf("UPDATE games SET %s WHERE id = ? AND user_id = ?", strings.Join(updates, ", "))

	result, err := h.db.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	if req.Links != nil {
		_, err := h.db.ExecContext(c.Request.Context(), "DELETE FROM game_links WHERE game_id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game links"})
			return
		}

		if err := h.insertGameLinks(c, id, req.Links); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game links"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Game updated successfully"})
}

func (h *GameHandler) DeleteGame(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")

	query := `DELETE FROM games WHERE id = ? AND user_id = ?`

	result, err := h.db.ExecContext(c.Request.Context(), query, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete game"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Game deleted successfully"})
}

func (h *GameHandler) SearchGames(c *gin.Context) {
	var params models.GameSearchParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	whereClauses := []string{"user_id = ?"}
	args := []any{userID}

	if params.Title != "" {
		whereClauses = append(whereClauses, "title LIKE ?")
		args = append(args, "%"+params.Title+"%")
	}

	if params.Developer != "" {
		whereClauses = append(whereClauses, "developer LIKE ?")
		args = append(args, "%"+params.Developer+"%")
	}

	if params.Description != "" {
		whereClauses = append(whereClauses, "description LIKE ?")
		args = append(args, "%"+params.Description+"%")
	}

	if params.MyThoughts != "" {
		whereClauses = append(whereClauses, "my_thoughts LIKE ?")
		args = append(args, "%"+params.MyThoughts+"%")
	}

	if params.Color != "" {
		whereClauses = append(whereClauses, "color = ?")
		args = append(args, params.Color)
	}

	if len(params.Genres) > 0 {
		genreClauses := []string{}
		for _, genre := range params.Genres {
			genreClauses = append(genreClauses, "EXISTS (SELECT 1 FROM json_each(genres) WHERE value = ?)")
			args = append(args, genre)
		}
		whereClauses = append(whereClauses, "("+strings.Join(genreClauses, " OR ")+")")
	}

	if len(params.Tags) > 0 {
		tagClauses := []string{}
		for _, tag := range params.Tags {
			tagClauses = append(tagClauses, "EXISTS (SELECT 1 FROM json_each(tags) WHERE value = ?)")
			args = append(args, tag)
		}
		whereClauses = append(whereClauses, "("+strings.Join(tagClauses, " OR ")+")")
	}

	if params.MinRating > 0 {
		whereClauses = append(whereClauses, "rating >= ?")
		args = append(args, params.MinRating)
	}

	if params.MaxRating > 0 {
		whereClauses = append(whereClauses, "rating <= ?")
		args = append(args, params.MaxRating)
	}

	if params.Rating > 0 {
		whereClauses = append(whereClauses, "rating = ?")
		args = append(args, params.Rating)
	}

	if params.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, params.Status)
	}

	if params.Explicit != nil {
		whereClauses = append(whereClauses, "explicit = ?")
		args = append(args, *params.Explicit)
	}

	if params.Bad != nil {
		whereClauses = append(whereClauses, "bad = ?")
		args = append(args, *params.Bad)
	}

	if params.MinPercent > 0 {
		whereClauses = append(whereClauses, "percent >= ?")
		args = append(args, params.MinPercent)
	}

	if params.MaxPercent > 0 {
		whereClauses = append(whereClauses, "percent <= ?")
		args = append(args, params.MaxPercent)
	}

	if params.CreatedAfter != "" {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, params.CreatedAfter+" 00:00:00")
	}

	if params.CreatedBefore != "" {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, params.CreatedBefore+" 23:59:59")
	}

	orderBy := "created_at DESC"
	if params.SortBy != "" {
		allowedSortFields := map[string]bool{
			"title": true, "developer": true, "rating": true,
			"status": true, "percent": true, "created_at": true, "updated_at": true,
		}
		if allowedSortFields[params.SortBy] {
			sortOrder := "ASC"
			if params.SortOrder == "desc" {
				sortOrder = "DESC"
			}
			orderBy = params.SortBy + " " + sortOrder
		}
	}

	limit := 50
	if params.Limit > 0 && params.Limit <= 100 {
		limit = params.Limit
	}

	offset := max(params.Offset, 0)

	query := fmt.Sprintf(`
		SELECT id, title, developer, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, percent, bad, user_id, 
		       created_at, updated_at 
		FROM games 
		WHERE %s 
		ORDER BY %s 
		LIMIT ? OFFSET ?
	`, strings.Join(whereClauses, " AND "), orderBy)

	args = append(args, limit, offset)

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search games"})
		return
	}
	defer rows.Close()

	games := []models.Game{}
	for rows.Next() {
		var g models.Game
		var genresJSON, tagsJSON string
		if err := rows.Scan(&g.ID, &g.Title, &g.Developer, &genresJSON, &tagsJSON,
			&g.Rating, &g.Status, &g.Description, &g.MyThoughts, &g.CoverImage,
			&g.Explicit, &g.Color, &g.Percent, &g.Bad, &g.UserID,
			&g.CreatedAt, &g.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan game"})
			return
		}

		json.Unmarshal([]byte(genresJSON), &g.Genres)
		json.Unmarshal([]byte(tagsJSON), &g.Tags)

		games = append(games, g)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating search results"})
		return
	}

	for i := range games {
		links, err := h.getGameLinks(c, games[i].ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch game links"})
			return
		}
		games[i].Links = links
	}

	c.JSON(http.StatusOK, gin.H{
		"results": games,
		"limit":   limit,
		"offset":  offset,
		"count":   len(games),
	})
}

func (h *GameHandler) getGameLinks(c *gin.Context, gameID int64) ([]models.GameLink, error) {
	query := `SELECT id, key, value, game_id FROM game_links WHERE game_id = ?`
	rows, err := h.db.QueryContext(c.Request.Context(), query, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []models.GameLink{}
	for rows.Next() {
		var link models.GameLink
		if err := rows.Scan(&link.ID, &link.Key, &link.Value, &link.GameID); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}

func (h *GameHandler) insertGameLinks(c *gin.Context, gameID int64, links []models.GameLink) error {
	if len(links) == 0 {
		return nil
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO game_links (key, value, game_id) VALUES (?, ?, ?)`
	stmt, err := tx.PrepareContext(c.Request.Context(), query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, link := range links {
		if _, err := stmt.ExecContext(c.Request.Context(), link.Key, link.Value, gameID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
