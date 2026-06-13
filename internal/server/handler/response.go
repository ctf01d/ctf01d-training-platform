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

const (
	codeNotFound        = "not_found"
	codeConflict        = "conflict"
	codeForbidden       = "forbidden"
	codeUnauthorized    = "unauthorized"
	codeValidationError = "validation_error"
	codeBadRequest      = "bad_request"
	codeInternalError   = "internal_error"

	msgMustFitInt32     = "must fit int32"
	msgNotAuthenticated = "not authenticated"
)

func respondError(c *gin.Context, err error) {
	var ve *errs.ValidationError
	switch {
	case errors.Is(err, errs.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Code: codeNotFound, Message: err.Error()})
	case errors.Is(err, errs.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse{Code: codeConflict, Message: err.Error()})
	case errors.Is(err, errs.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse{Code: codeForbidden, Message: err.Error()})
	case errors.Is(err, errs.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: err.Error()})
	case errors.As(err, &ve):
		details := make(map[string]interface{}, len(ve.Fields))
		for k, v := range ve.Fields {
			details[k] = v
		}
		c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Code:    codeValidationError,
			Message: ve.Error(),
			Details: details,
		})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Code: codeInternalError, Message: "internal server error"})
	}
}

func bindJSON[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			_ = syntaxErr
			c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Code:    codeValidationError,
				Message: "invalid JSON",
			})
			return req, false
		}
		c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Code:    codeValidationError,
			Message: "request validation failed",
		})
		return req, false
	}
	return req, true
}
