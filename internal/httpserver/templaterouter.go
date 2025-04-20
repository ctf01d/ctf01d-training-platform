package httpserver

import (
	"net/http"
	"path/filepath"
	"strings"

	"html/template"

	"github.com/gin-gonic/gin"
)

var tmplDir = "./templates/"

func NewSSRRouter() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// Не трогаем API и статику
		if strings.HasPrefix(reqPath, "/api/") || strings.HasPrefix(reqPath, "/static/") {
			c.Next()
			return
		}

		// Определяем шаблон по пути
		var pageTmpl string
		switch reqPath {
		case "/", "/index", "/index.html":
			pageTmpl = "index.html"
		case "/games", "/games/":
			pageTmpl = "games.html"
		case "/teams", "/teams/":
			pageTmpl = "teams.html"
		case "/users", "/users/":
			pageTmpl = "users.html"
		default:
			pageTmpl = "404.html"
		}

		// Парсим base.html + нужный шаблон
		tmpl, err := template.ParseFiles(
			filepath.Join(tmplDir, "base.html"),
			filepath.Join(tmplDir, pageTmpl),
		)
		if err != nil {
			c.String(http.StatusInternalServerError, "Template error: %v", err)
			return
		}

		c.Status(http.StatusOK)
		// Рендерим всегда через base.html, чтобы работали блоки
		tmpl.ExecuteTemplate(c.Writer, "base.html", nil)
	}
}
