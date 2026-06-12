package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/internal/domain/errs"
)

type errorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func respondError(c *gin.Context, err error) {
	var ve *errs.ValidationError
	switch {
	case errors.Is(err, errs.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Code: "not_found", Message: err.Error()})
	case errors.Is(err, errs.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse{Code: "conflict", Message: err.Error()})
	case errors.Is(err, errs.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse{Code: "forbidden", Message: err.Error()})
	case errors.Is(err, errs.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, errorResponse{Code: "unauthorized", Message: err.Error()})
	case errors.As(err, &ve):
		details := make(map[string]interface{}, len(ve.Fields))
		for k, v := range ve.Fields {
			details[k] = v
		}
		c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Code:    "validation_error",
			Message: ve.Error(),
			Details: details,
		})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Code: "internal_error", Message: "internal server error"})
	}
}

func bindJSON[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			_ = syntaxErr
			c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Code:    "validation_error",
				Message: "invalid JSON",
			})
			return req, false
		}
		c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Code:    "validation_error",
			Message: "request validation failed",
		})
		return req, false
	}
	return req, true
}
