package handler

import (
	"net/http"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/gin-gonic/gin"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) ListUsers(c *gin.Context, params httpserver.ListUsersParams) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented"})
}

func (h *Handler) CreateUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented"})
}

func (h *Handler) DeleteUser(c *gin.Context, id int64) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented"})
}

func (h *Handler) GetUser(c *gin.Context, id int64) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented"})
}

func (h *Handler) UpdateUser(c *gin.Context, id int64) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": "not_implemented"})
}

var _ httpserver.ServerInterface = (*Handler)(nil)
