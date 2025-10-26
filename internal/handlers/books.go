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
)

type BookHandler struct {
	db *database.DB
}

func NewBookHandler(db *database.DB) *BookHandler {
	return &BookHandler{db: db}
}

func (h *BookHandler) saveCoverImage(c *gin.Context, fileheader *multipart.FileHeader) (string, error) {
	file, err := fileheader.Open()
	if err != nil {
		return "", fmt.Errorf("open error: %v", err)
	}

	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash error: %v", err)
	}
	hashSum := hex.EncodeToString(hasher.Sum(nil))
	ext := strings.ToLower(filepath.Ext(fileheader.Filename))
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

func (h *BookHandler) saveCoverFromURL(c *gin.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: 30 & time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BookHandler/1.0)")
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BookHandler/1.0)")

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

func (h *BookHandler) GetBooks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	query := `
		SELECT id, title, author, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, user_id, 
		       created_at, updated_at 
		FROM books
		WHERE user_id = ? 
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch books"})
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var b models.Book
		var genresJSON, tagsJSON string
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &genresJSON, &tagsJSON,
			&b.Rating, &b.Status, &b.Description, &b.MyThoughts, &b.CoverImage,
			&b.Explicit, &b.Color, &b.UserID,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan book"})
			return
		}

		json.Unmarshal([]byte(genresJSON), &b.Genres)
		json.Unmarshal([]byte(tagsJSON), &b.Tags)

		books = append(books, b)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating books"})
		return
	}

	for i := range books {
		links, err := h.getBookLinks(c, books[i].ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch book links"})
			return
		}
		books[i].Links = links
	}

	c.JSON(http.StatusOK, books)
}

func (h *BookHandler) GetBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")

	query := `
		SELECT id, title, author, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, user_id, 
		       created_at, updated_at 
		FROM books
		WHERE id = ? AND user_id = ?
	`

	var b models.Book
	var genresJSON, tagsJSON string
	err = h.db.QueryRowContext(c.Request.Context(), query, id, userID).Scan(
		&b.ID, &b.Title, &b.Author, &genresJSON, &tagsJSON,
		&b.Rating, &b.Status, &b.Description, &b.MyThoughts, &b.CoverImage,
		&b.Explicit, &b.Color, &b.UserID,
		&b.CreatedAt, &b.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch book"})
		return
	}

	json.Unmarshal([]byte(genresJSON), &b.Genres)
	json.Unmarshal([]byte(tagsJSON), &b.Tags)

	links, err := h.getBookLinks(c, b.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch book links"})
		return
	}
	b.Links = links

	c.JSON(http.StatusOK, b)
}

func (h *BookHandler) CreateBook(c *gin.Context) {
	var req models.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	var coverImageURL string
	var err error

	fileHeader, fileErr := c.FormFile("cover_image")
	if fileErr == nil && fileHeader != nil {
		coverImageURL, err = h.saveCoverImage(c, fileHeader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.CoverImageURL != "" {
		coverImageURL, err = h.saveCoverFromURL(c, req.CoverImageURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to download cover image: %v", err)})
			return
		}
	} else if req.CoverImage != "" {
		coverImageURL = req.CoverImage
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cover_image, cover_image_url, or cover_image file upload is required"})
		return
	}

	genresJSON, _ := json.Marshal(req.Genres)
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO books (title, author, genres, tags, rating, status, description, 
		                   my_thoughts, cover_image, explicit, color, user_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
		RETURNING id, created_at, updated_at
	`

	var id int64
	var createdAt, updatedAt string
	err = h.db.QueryRowContext(c.Request.Context(), query,
		req.Title, req.Author, string(genresJSON), string(tagsJSON),
		req.Rating, req.Status, req.Description, req.MyThoughts,
		coverImageURL, req.Explicit, req.Color, userID,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create book"})
		return
	}

	if err := h.insertBookLinks(c, id, req.Links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create book links"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"cover_image": coverImageURL,
		"message":     "Book created successfully",
	})
}

func (h *BookHandler) UpdateBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req models.UpdateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	updates := []string{}
	args := []any{}

	if req.Title != "" {
		updates = append(updates, "title = ?")
		args = append(args, req.Title)
	}
	if req.Author != "" {
		updates = append(updates, "author = ?")
		args = append(args, req.Author)
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

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id, userID)

	query := fmt.Sprintf("UPDATE books SET %s WHERE id = ? AND user_id = ?", strings.Join(updates, ", "))

	result, err := h.db.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	if req.Links != nil {
		_, err := h.db.ExecContext(c.Request.Context(), "DELETE FROM book_links WHERE book_id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book links"})
			return
		}

		if err := h.insertBookLinks(c, id, req.Links); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book links"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book updated successfully"})
}

func (h *BookHandler) DeleteBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")

	query := `DELETE FROM books WHERE id = ? AND user_id = ?`

	result, err := h.db.ExecContext(c.Request.Context(), query, id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete book"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book deleted successfully"})
}

func (h *BookHandler) SearchBooks(c *gin.Context) {
	var params models.BookSearchParams
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

	if params.Author != "" {
		whereClauses = append(whereClauses, "author LIKE ?")
		args = append(args, "%"+params.Author+"%")
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
			"title": true, "author": true, "rating": true,
			"status": true, "created_at": true, "updated_at": true,
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
		SELECT id, title, author, genres, tags, rating, status, description, 
		       my_thoughts, cover_image, explicit, color, user_id, 
		       created_at, updated_at 
		FROM books
		WHERE %s 
		ORDER BY %s 
		LIMIT ? OFFSET ?
	`, strings.Join(whereClauses, " AND "), orderBy)

	args = append(args, limit, offset)

	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search books"})
		return
	}
	defer rows.Close()

	books := []models.Book{}
	for rows.Next() {
		var b models.Book
		var genresJSON, tagsJSON string
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &genresJSON, &tagsJSON,
			&b.Rating, &b.Status, &b.Description, &b.MyThoughts, &b.CoverImage,
			&b.Explicit, &b.Color, &b.UserID,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan book"})
			return
		}

		json.Unmarshal([]byte(genresJSON), &b.Genres)
		json.Unmarshal([]byte(tagsJSON), &b.Tags)

		books = append(books, b)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating search results"})
		return
	}

	for i := range books {
		links, err := h.getBookLinks(c, books[i].ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch book links"})
			return
		}
		books[i].Links = links
	}

	c.JSON(http.StatusOK, gin.H{
		"results": books,
		"limit":   limit,
		"offset":  offset,
		"count":   len(books),
	})
}

func (h *BookHandler) getBookLinks(c *gin.Context, bookID int64) ([]models.BookLink, error) {
	query := `SELECT id, key, value, book_id FROM book_links WHERE book_id = ?`
	rows, err := h.db.QueryContext(c.Request.Context(), query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []models.BookLink{}
	for rows.Next() {
		var link models.BookLink
		if err := rows.Scan(&link.ID, &link.Key, &link.Value, &link.BookID); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}

func (h *BookHandler) insertBookLinks(c *gin.Context, bookID int64, links []models.BookLink) error {
	if len(links) == 0 {
		return nil
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO book_links (key, value, book_id) VALUES (?, ?, ?)`
	stmt, err := tx.PrepareContext(c.Request.Context(), query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, link := range links {
		if _, err := stmt.ExecContext(c.Request.Context(), link.Key, link.Value, bookID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
