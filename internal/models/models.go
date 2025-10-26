package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

const (
	RoleAdmin  = "admin"
	RoleNormal = "normal"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
	Role      string    `json:"role"`
}

type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type Resource struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	UserID      int64     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateResourceRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateResourceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type StringArray []string

func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return errors.New("failed to scan StringArray: invalid type")
		}
		bytes = []byte(str)
	}

	return json.Unmarshal(bytes, s)
}

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	bytes, err := json.Marshal(s)
	return string(bytes), err
}

type Game struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title" binding:"required"`
	Developer   string      `json:"developer" binding:"required"`
	Genres      StringArray `json:"genres" binding:"required"`
	Tags        StringArray `json:"tags" binding:"required"`
	Rating      int         `json:"rating" binding:"required,min=1,max=5"`
	Status      string      `json:"status" binding:"required"`
	Description string      `json:"description" binding:"required"`
	MyThoughts  string      `json:"my_thoughts" binding:"required"`
	Links       []GameLink  `json:"links" binding:"required"`
	CoverImage  string      `json:"cover_image" binding:"required"`
	Explicit    bool        `json:"explicit"`
	Color       string      `json:"color" binding:"required"`
	Percent     int         `json:"percent" binding:"required,min=0,max=100"`
	Bad         bool        `json:"bad"`
	UserID      int64       `json:"user_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Book struct {
	ID          int64       `json:"id"`
	Title       string      `json:"title" binding:"required"`
	Author      string      `json:"author" binding:"required"`
	Genres      StringArray `json:"genres" binding:"required"`
	Tags        StringArray `json:"tags" binding:"required"`
	Rating      int         `json:"rating" binding:"required,min=1,max=5"`
	Status      string      `json:"status" binding:"required"`
	Description string      `json:"description" binding:"required"`
	MyThoughts  string      `json:"my_thoughts" binding:"required"`
	Links       []BookLink  `json:"links" binding:"required"`
	CoverImage  string      `json:"cover_image" binding:"required"`
	Explicit    bool        `json:"explicit"`
	Color       string      `json:"color" binding:"required"`
	UserID      int64       `json:"user_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type GameLink struct {
	ID     int64  `json:"id"`
	Key    string `json:"key" binding:"required"`
	Value  string `json:"value" binding:"required"`
	GameID int64  `json:"game_id,omitempty"`
}

type BookLink struct {
	ID     int64  `json:"id"`
	Key    string `json:"key" binding:"required"`
	Value  string `json:"value" binding:"required"`
	BookID int64  `json:"book_id,omitempty"`
}

type CreateGameRequest struct {
	Title         string      `json:"title" binding:"required"`
	Developer     string      `json:"developer" binding:"required"`
	Genres        StringArray `json:"genres" binding:"required"`
	Tags          StringArray `json:"tags" binding:"required"`
	Rating        int         `json:"rating" binding:"required,min=1,max=5"`
	Status        string      `json:"status" binding:"required"`
	Description   string      `json:"description" binding:"required"`
	MyThoughts    string      `json:"my_thoughts" binding:"required"`
	Links         []GameLink  `json:"links" binding:"required"`
	CoverImage    string      `json:"cover_image"`
	CoverImageURL string      `json:"cover_image_url"`
	Explicit      bool        `json:"explicit"`
	Color         string      `json:"color" binding:"required"`
	Percent       int         `json:"percent" binding:"required,min=0,max=100"`
	Bad           bool        `json:"bad"`
}

type CreateBookRequest struct {
	Title         string      `json:"title" binding:"required"`
	Author        string      `json:"author" binding:"required"`
	Genres        StringArray `json:"genres" binding:"required"`
	Tags          StringArray `json:"tags" binding:"required"`
	Rating        int         `json:"rating" binding:"required,min=1,max=5"`
	Status        string      `json:"status" binding:"required"`
	Description   string      `json:"description" binding:"required"`
	MyThoughts    string      `json:"my_thoughts" binding:"required"`
	Links         []BookLink  `json:"links" binding:"required"`
	CoverImage    string      `json:"cover_image" binding:"required"`
	CoverImageURL string      `json:"cover_image_url" binding:"required"`
	Explicit      bool        `json:"explicit"`
	Color         string      `json:"color" binding:"required"`
}

type UpdateGameRequest struct {
	Title       string      `json:"title"`
	Developer   string      `json:"developer"`
	Genres      StringArray `json:"genres"`
	Tags        StringArray `json:"tags"`
	Rating      int         `json:"rating" binding:"omitempty,min=1,max=5"`
	Status      string      `json:"status"`
	Description string      `json:"description"`
	MyThoughts  string      `json:"my_thoughts"`
	Links       []GameLink  `json:"links"`
	CoverImage  string      `json:"cover_image"`
	Explicit    *bool       `json:"explicit"`
	Color       string      `json:"color"`
	Percent     int         `json:"percent" binding:"omitempty,min=0,max=100"`
	Bad         *bool       `json:"bad"`
}

type UpdateBookRequest struct {
	Title       string      `json:"title" binding:"required"`
	Author      string      `json:"author" binding:"required"`
	Genres      StringArray `json:"genres" binding:"required"`
	Tags        StringArray `json:"tags" binding:"required"`
	Rating      int         `json:"rating" binding:"required,min=1,max=5"`
	Status      string      `json:"status" binding:"required"`
	Description string      `json:"description" binding:"required"`
	MyThoughts  string      `json:"my_thoughts" binding:"required"`
	Links       []BookLink  `json:"links" binding:"required"`
	CoverImage  string      `json:"cover_image" binding:"required"`
	Explicit    *bool       `json:"explicit"`
	Color       string      `json:"color" binding:"required"`
}

type GameSearchParams struct {
	Title         string   `form:"title"`
	Developer     string   `form:"developer"`
	Genres        []string `form:"genres"`
	Tags          []string `form:"tags"`
	Description   string   `form:"description"`
	MyThoughts    string   `form:"my_thoughts"`
	Color         string   `form:"color"`
	Rating        int      `form:"rating"`
	MinRating     int      `form:"min_rating"`
	MaxRating     int      `form:"max_rating"`
	CreatedAfter  string   `form:"created_after"`
	CreatedBefore string   `form:"created_before"`
	Status        string   `form:"status"`
	Explicit      *bool    `form:"explicit"`
	Bad           *bool    `form:"bad"`
	MinPercent    int      `form:"min_percent"`
	MaxPercent    int      `form:"max_percent"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order"`
	Limit         int      `form:"limit"`
	Offset        int      `form:"offset"`
}

type BookSearchParams struct {
	Title         string   `form:"title"`
	Author        string   `form:"author"`
	Genres        []string `form:"genres"`
	Tags          []string `form:"tags"`
	Description   string   `form:"description"`
	MyThoughts    string   `form:"my_thoughts"`
	Color         string   `form:"color"`
	Rating        int      `form:"rating"`
	MinRating     int      `form:"min_rating"`
	MaxRating     int      `form:"max_rating"`
	CreatedAfter  string   `form:"created_after"`
	CreatedBefore string   `form:"created_before"`
	Status        string   `form:"status"`
	Explicit      *bool    `form:"explicit"`
	SortBy        string   `form:"sort_by"`
	SortOrder     string   `form:"sort_order"`
	Limit         int      `form:"limit"`
	Offset        int      `form:"offset"`
}

type RouteInfo struct {
	Method       string   `json:"method"`
	Path         string   `json:"path"`
	Description  string   `json:"description,omitempty"`
	Protected    bool     `json:"protected"`
	Group        string   `json:"group"`
	Params       []string `json:"params,omitempty"`
	RequiredRole string   `json:"required_role,omitempty"`
}

type RouteGroup struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	BasePath    string      `json:"base_path"`
	Routes      []RouteInfo `json:"routes"`
}

type Comment struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content" binding:"required,min=1,max=1000"`
	GameID    *int64    `json:"game_id,omitempty"`
	BookID    *int64    `json:"book_id,omitempty"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=1000"`
	GameID  *int64 `json:"game_id"`
	BookID  *int64 `json:"book_id"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=1000"`
}
