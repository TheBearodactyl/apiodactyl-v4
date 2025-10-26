package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func GenErr(title string, e error) gin.H {
	return gin.H{
		"error": gin.H{
			"title":   title,
			"message": e.Error(),
		},
	}
}

func hasAllowedExtension(filename string) bool {
	allowedExtensions := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".webm": true,
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".mp3":  true,
		".ogg":  true,
		".avif": true,
	}
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedExtensions[ext]
}

func UploadFile(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("failed to upload file: %v", err))
		return
	}

	if !hasAllowedExtension(fileHeader.Filename) {
		c.String(http.StatusForbidden, "file MUST end with one of the following extensions:\nmp4, mkv, webm, png, jpg, jpeg, gif, mp3, ogg, avif")
		return
	}

	allowedMimes := map[string]string{
		".mp4":  "video/mp4",
		".mkv":  "video/x-matroska",
		".webm": "video/webm",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".avif": "image/avif",
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	expectedMime := allowedMimes[ext]

	file, err := fileHeader.Open()
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("file open error: %v", err))
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("hash compute error: %v", err))
		return
	}
	hashSum := hex.EncodeToString(hasher.Sum(nil))
	file.Seek(0, io.SeekStart)

	hashFilename := hashSum + ext
	savePath := filepath.Join("./files", hashFilename)
	if _, err := os.Stat(savePath); err == nil {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		permalink := fmt.Sprintf("%s://%s/files/%s", scheme, host, hashFilename)
		c.JSON(http.StatusOK, gin.H{
			"message":   "duplicate detected, returning existing file",
			"filename":  hashFilename,
			"permalink": permalink,
		})
		return
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		c.String(http.StatusInternalServerError, fmt.Sprintf("file read error: %v", err))
		return
	}
	mimeType := http.DetectContentType(buf[:n])
	if mimeType != expectedMime {
		c.String(http.StatusBadRequest, fmt.Sprintf("mimetype %s does not match expected %s", mimeType, expectedMime))
		return
	}
	file.Seek(0, io.SeekStart)

	if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to save file: %v", err))
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	permalink := fmt.Sprintf("%s://%s/files/%s", scheme, host, hashFilename)

	c.JSON(http.StatusOK, gin.H{
		"message":   "file uploaded successfully",
		"filename":  hashFilename,
		"permalink": permalink,
	})
}
