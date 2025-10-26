package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thebearodactyl/apiodactyl/internal/models"
)

type RouteHandler struct{}

func NewRouteHandler() *RouteHandler {
	return &RouteHandler{}
}

func (h *RouteHandler) GetAllRoutes(c *gin.Context) {
	routes := []models.RouteGroup{
		{
			Name:        "Authentication",
			Description: "User authentication and profile management",
			BasePath:    "/api/v1",
			Routes: []models.RouteInfo{
				{
					Method:      "POST",
					Path:        "/auth/register",
					Description: "Register a new user account",
					Protected:   false,
					Group:       "auth",
				},
				{
					Method:      "POST",
					Path:        "/auth/login",
					Description: "Login with username and password",
					Protected:   false,
					Group:       "auth",
				},
				{
					Method:      "GET",
					Path:        "/me",
					Description: "Get current user profile",
					Protected:   true,
					Group:       "auth",
				},
			},
		},
		{
			Name:        "Resources",
			Description: "Manage generic resources",
			BasePath:    "/api/v1/resources",
			Routes: []models.RouteInfo{
				{
					Method:      "GET",
					Path:        "",
					Description: "Get all resources for the current user",
					Protected:   true,
					Group:       "resources",
				},
				{
					Method:      "GET",
					Path:        "/:id",
					Description: "Get a specific resource by ID",
					Protected:   true,
					Group:       "resources",
					Params:      []string{"id"},
				},
				{
					Method:       "POST",
					Path:         "",
					Description:  "Create a new resource",
					Protected:    true,
					Group:        "resources",
					RequiredRole: "admin",
				},
				{
					Method:       "PUT",
					Path:         "/:id",
					Description:  "Update a resource by ID",
					Protected:    true,
					Group:        "resources",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
				{
					Method:       "DELETE",
					Path:         "/:id",
					Description:  "Delete a resource by ID",
					Protected:    true,
					Group:        "resources",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
			},
		},
		{
			Name:        "Games",
			Description: "Manage game entries",
			BasePath:    "/api/v1/games",
			Routes: []models.RouteInfo{
				{
					Method:      "GET",
					Path:        "",
					Description: "Get all games for the current user",
					Protected:   true,
					Group:       "games",
				},
				{
					Method:      "GET",
					Path:        "/:id",
					Description: "Get a specific game by ID",
					Protected:   true,
					Group:       "games",
					Params:      []string{"id"},
				},
				{
					Method:       "POST",
					Path:         "",
					Description:  "Create a new game entry (supports cover_image file upload or cover_image_url)",
					Protected:    true,
					Group:        "games",
					RequiredRole: "admin",
				},
				{
					Method:       "PUT",
					Path:         "/:id",
					Description:  "Update a game entry by ID",
					Protected:    true,
					Group:        "games",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
				{
					Method:       "DELETE",
					Path:         "/:id",
					Description:  "Delete a game entry by ID",
					Protected:    true,
					Group:        "games",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
				{
					Method:      "GET",
					Path:        "/search",
					Description: "Search games with various filters",
					Protected:   true,
					Group:       "games",
				},
			},
		},
		{
			Name:        "Books",
			Description: "Manage book entries",
			BasePath:    "/api/v1/books",
			Routes: []models.RouteInfo{
				{
					Method:      "GET",
					Path:        "",
					Description: "Get all books for the current user",
					Protected:   true,
					Group:       "books",
				},
				{
					Method:      "GET",
					Path:        "/:id",
					Description: "Get a specific book by ID",
					Protected:   true,
					Group:       "books",
					Params:      []string{"id"},
				},
				{
					Method:       "POST",
					Path:         "",
					Description:  "Create a new book entry (supports cover_image file upload or cover_image_url)",
					Protected:    true,
					Group:        "books",
					RequiredRole: "admin",
				},
				{
					Method:       "PUT",
					Path:         "/:id",
					Description:  "Update a book entry by ID",
					Protected:    true,
					Group:        "books",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
				{
					Method:       "DELETE",
					Path:         "/:id",
					Description:  "Delete a book entry by ID",
					Protected:    true,
					Group:        "books",
					Params:       []string{"id"},
					RequiredRole: "admin",
				},
				{
					Method:      "GET",
					Path:        "/search",
					Description: "Search books with various filters",
					Protected:   true,
					Group:       "books",
				},
			},
		},
		{
			Name:        "Comments",
			Description: "Manage comments on games and books",
			BasePath:    "/api/v1/comments",
			Routes: []models.RouteInfo{
				{
					Method:      "POST",
					Path:        "",
					Description: "Create a new comment on a game or book",
					Protected:   true,
					Group:       "comments",
				},
				{
					Method:      "GET",
					Path:        "",
					Description: "Get comments for a game or book (requires game_id or book_id query param)",
					Protected:   true,
					Group:       "comments",
				},
				{
					Method:      "PUT",
					Path:        "/:id",
					Description: "Update your own comment (admins can update any comment)",
					Protected:   true,
					Group:       "comments",
					Params:      []string{"id"},
				},
				{
					Method:      "DELETE",
					Path:        "/:id",
					Description: "Delete your own comment (admins can delete any comment)",
					Protected:   true,
					Group:       "comments",
					Params:      []string{"id"},
				},
			},
		},
		{
			Name:        "Files",
			Description: "File upload management",
			BasePath:    "/api/v1",
			Routes: []models.RouteInfo{
				{
					Method:       "POST",
					Path:         "/upload",
					Description:  "Upload a file (images, videos, audio)",
					Protected:    true,
					Group:        "files",
					RequiredRole: "admin",
				},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"route_groups": routes,
		"total_groups": len(routes),
	})
}

func (h *RouteHandler) GetResourcesRoutes(c *gin.Context) {
	routes := models.RouteGroup{
		Name:        "Resources",
		Description: "Manage generic resources",
		BasePath:    "/api/v1/resources",
		Routes: []models.RouteInfo{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get all resources for the current user",
				Protected:   true,
				Group:       "resources",
			},
			{
				Method:      "GET",
				Path:        "/:id",
				Description: "Get a specific resource by ID",
				Protected:   true,
				Group:       "resources",
				Params:      []string{"id"},
			},
			{
				Method:       "POST",
				Path:         "",
				Description:  "Create a new resource",
				Protected:    true,
				Group:        "resources",
				RequiredRole: "admin",
			},
			{
				Method:       "PUT",
				Path:         "/:id",
				Description:  "Update a resource by ID",
				Protected:    true,
				Group:        "resources",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
			{
				Method:       "DELETE",
				Path:         "/:id",
				Description:  "Delete a resource by ID",
				Protected:    true,
				Group:        "resources",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
		},
	}

	c.JSON(http.StatusOK, routes)
}

func (h *RouteHandler) GetGamesRoutes(c *gin.Context) {
	routes := models.RouteGroup{
		Name:        "Games",
		Description: "Manage game entries",
		BasePath:    "/api/v1/games",
		Routes: []models.RouteInfo{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get all games for the current user",
				Protected:   true,
				Group:       "games",
			},
			{
				Method:      "GET",
				Path:        "/:id",
				Description: "Get a specific game by ID",
				Protected:   true,
				Group:       "games",
				Params:      []string{"id"},
			},
			{
				Method:       "POST",
				Path:         "",
				Description:  "Create a new game entry (supports cover_image file upload or cover_image_url in JSON)",
				Protected:    true,
				Group:        "games",
				RequiredRole: "admin",
			},
			{
				Method:       "PUT",
				Path:         "/:id",
				Description:  "Update a game entry by ID",
				Protected:    true,
				Group:        "games",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
			{
				Method:       "DELETE",
				Path:         "/:id",
				Description:  "Delete a game entry by ID",
				Protected:    true,
				Group:        "games",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
			{
				Method:      "GET",
				Path:        "/search",
				Description: "Search games with various filters",
				Protected:   true,
				Group:       "games",
			},
		},
	}

	c.JSON(http.StatusOK, routes)
}

func (h *RouteHandler) GetBooksRoutes(c *gin.Context) {
	routes := models.RouteGroup{
		Name:        "Books",
		Description: "Manage book entries",
		BasePath:    "/api/v1/books",
		Routes: []models.RouteInfo{
			{
				Method:      "GET",
				Path:        "",
				Description: "Get all books for the current user",
				Protected:   true,
				Group:       "books",
			},
			{
				Method:      "GET",
				Path:        "/:id",
				Description: "Get a specific book by ID",
				Protected:   true,
				Group:       "books",
				Params:      []string{"id"},
			},
			{
				Method:       "POST",
				Path:         "",
				Description:  "Create a new book entry (supports cover_image file upload or cover_image_url in JSON)",
				Protected:    true,
				Group:        "books",
				RequiredRole: "admin",
			},
			{
				Method:       "PUT",
				Path:         "/:id",
				Description:  "Update a book entry by ID",
				Protected:    true,
				Group:        "books",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
			{
				Method:       "DELETE",
				Path:         "/:id",
				Description:  "Delete a book entry by ID",
				Protected:    true,
				Group:        "books",
				Params:       []string{"id"},
				RequiredRole: "admin",
			},
			{
				Method:      "GET",
				Path:        "/search",
				Description: "Search books with various filters",
				Protected:   true,
				Group:       "books",
			},
		},
	}

	c.JSON(http.StatusOK, routes)
}

func (h *RouteHandler) GetCommentsRoutes(c *gin.Context) {
	routes := models.RouteGroup{
		Name:        "Comments",
		Description: "Manage comments on games and books",
		BasePath:    "/api/v1/comments",
		Routes: []models.RouteInfo{
			{
				Method:      "POST",
				Path:        "",
				Description: "Create a new comment on a game or book",
				Protected:   true,
				Group:       "comments",
			},
			{
				Method:      "GET",
				Path:        "",
				Description: "Get comments for a game or book (requires game_id or book_id query param)",
				Protected:   true,
				Group:       "comments",
			},
			{
				Method:      "PUT",
				Path:        "/:id",
				Description: "Update your own comment (admins can update any comment)",
				Protected:   true,
				Group:       "comments",
				Params:      []string{"id"},
			},
			{
				Method:      "DELETE",
				Path:        "/:id",
				Description: "Delete your own comment (admins can delete any comment)",
				Protected:   true,
				Group:       "comments",
				Params:      []string{"id"},
			},
		},
	}

	c.JSON(http.StatusOK, routes)
}
