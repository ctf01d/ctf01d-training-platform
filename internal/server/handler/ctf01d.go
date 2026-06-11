package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	ctf01dsvc "github.com/ctf01d/ctf01d-training-platform/internal/service/ctf01d"
	"github.com/gin-gonic/gin"
)

func (h *Handler) HandleGetCtf01dExportOptions(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	opts, err := h.ctf01dBuilder.BuildOptions(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	resp := httpserver.Ctf01dExportOptions{
		FlagTtlMin:      &opts.FlagTtlMin,
		BasicAttackCost: &opts.BasicAttackCost,
		IncludeHtml:     &opts.IncludeHtml,
		IncludeCompose:  &opts.IncludeCompose,
	}
	defenceCost := float32(opts.DefenceCost)
	resp.DefenceCost = &defenceCost
	if opts.Port > 0 {
		resp.Port = &opts.Port
	}
	if opts.HtmlSourcePath != "" {
		resp.HtmlSourcePath = &opts.HtmlSourcePath
	}
	if opts.CoffeeBreakStart != nil {
		resp.CoffeeBreakStart = opts.CoffeeBreakStart
	}
	if opts.CoffeeBreakEnd != nil {
		resp.CoffeeBreakEnd = opts.CoffeeBreakEnd
	}
	if len(opts.Warnings) > 0 {
		resp.Warnings = &opts.Warnings
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) HandleExportCtf01d(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.Ctf01dExportRequest](c)
	if !ok {
		return
	}

	if req.HtmlSourcePath != nil && *req.HtmlSourcePath != "" {
		abs, err := filepath.Abs(*req.HtmlSourcePath)
		if err != nil {
			respondError(c, fmt.Errorf("invalid html_source_path"))
			return
		}
		abs = filepath.Clean(abs)
		allowedBase, err := filepath.Abs(h.storageDir)
		if err != nil {
			respondError(c, fmt.Errorf("resolving storage dir: %w", err))
			return
		}
		if !strings.HasPrefix(abs, allowedBase+string(filepath.Separator)) && abs != allowedBase {
			respondError(c, fmt.Errorf("html_source_path must be within the storage directory"))
			return
		}
		*req.HtmlSourcePath = abs
	}

	builderReq := ctf01dsvc.Ctf01dExportRequest{
		Prefix:           req.Prefix,
		IncludeHtml:      req.IncludeHtml,
		HtmlSourcePath:   req.HtmlSourcePath,
		IncludeCompose:   req.IncludeCompose,
		ComposeProject:   req.ComposeProject,
		Port:             req.Port,
		Htmlfolder:       req.Htmlfolder,
		Random:           req.Random,
		FlagTtlMin:       req.FlagTtlMin,
		BasicAttackCost:  req.BasicAttackCost,
		CoffeeBreakStart: req.CoffeeBreakStart,
		CoffeeBreakEnd:   req.CoffeeBreakEnd,
	}
	if req.DefenceCost != nil {
		dc := float64(*req.DefenceCost)
		builderReq.DefenceCost = &dc
	}

	result, err := h.ctf01dBuilder.BuildParams(c.Request.Context(), id, builderReq)
	if err != nil {
		respondError(c, err)
		return
	}

	opts := result.Options
	opts.Warnings = result.Warnings

	exportResult, err := ctf01dsvc.Export(result.Game, result.Scoreboard, result.Teams, result.Checkers, opts)
	if err != nil {
		if exportErr, ok := err.(*ctf01dsvc.ExportError); ok {
			c.JSON(http.StatusUnprocessableEntity, httpserver.Ctf01dExportError{
				Code:    "validation_error",
				Errors:  exportErr.Errors,
				Message: strPtr("export validation failed"),
			})
			return
		}
		respondError(c, err)
		return
	}

	c.Header("Content-Type", "application/zip")
	safeName := sanitizeFilename(exportResult.Filename)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, safeName))
	c.Header("Content-Length", fmt.Sprintf("%d", len(exportResult.Data)))
	c.Data(http.StatusOK, "application/zip", exportResult.Data)
}

func strPtr(s string) *string {
	return &s
}
