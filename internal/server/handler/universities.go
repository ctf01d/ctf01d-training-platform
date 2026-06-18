package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	unisvc "github.com/ctf01d/ctf01d-training-platform/internal/service/universities"
)

func (h *Handler) HandleListUniversities(c *gin.Context) {
	page := 1
	perPage := 20
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			perPage = p
		}
	}

	var q *string
	if v := c.Query("q"); v != "" {
		q = &v
	}

	result, err := h.universities.List(c.Request.Context(), page, perPage, q)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.University, len(result.Items))
	for i, u := range result.Items {
		items[i] = universityToHTTP(u)
	}

	c.JSON(http.StatusOK, httpserver.UniversityList{
		Items: items,
		Pagination: httpserver.Pagination{
			Page:    result.Page,
			PerPage: result.PerPage,
			Total:   int(result.Total),
		},
	})
}

func (h *Handler) HandleCreateUniversity(c *gin.Context) {
	req, ok := bindJSON[httpserver.UniversityCreate](c)
	if !ok {
		return
	}

	params := unisvc.CreateParams{
		Name:      req.Name,
		SiteUrl:   req.SiteUrl,
		AvatarUrl: req.AvatarUrl,
	}

	uni, err := h.universities.Create(c.Request.Context(), params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, universityToHTTP(*uni))
}

func (h *Handler) HandleGetUniversity(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	uni, err := h.universities.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, universityToHTTP(*uni))
}

func (h *Handler) HandleUpdateUniversity(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.UniversityUpdate](c)
	if !ok {
		return
	}

	params := unisvc.UpdateParams{
		Name:      req.Name,
		SiteUrl:   req.SiteUrl,
		AvatarUrl: req.AvatarUrl,
	}

	uni, err := h.universities.Update(c.Request.Context(), id, params)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, universityToHTTP(*uni))
}

func (h *Handler) HandleDeleteUniversity(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.universities.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func universityToHTTP(u unisvc.University) httpserver.University {
	return httpserver.University{
		Id:        u.ID,
		Name:      u.Name,
		SiteUrl:   u.SiteUrl,
		AvatarUrl: u.AvatarUrl,
		CreatedAt: &u.CreatedAt,
		UpdatedAt: &u.UpdatedAt,
	}
}
