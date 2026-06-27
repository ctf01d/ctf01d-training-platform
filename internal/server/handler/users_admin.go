package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ctf01d/ctf01d-training-platform/gen/httpserver"
	"github.com/ctf01d/ctf01d-training-platform/internal/imageutil"
	"github.com/ctf01d/ctf01d-training-platform/internal/server/middleware"
	usersvc "github.com/ctf01d/ctf01d-training-platform/internal/service/users"
	"github.com/ctf01d/ctf01d-training-platform/internal/storage"
)

const avatarMaxDimension = imageutil.AvatarMaxDimension

func userAvatarKey(id int64) string {
	return fmt.Sprintf("avatars/%d.png", id)
}

func (h *Handler) HandleUpdateUserProfileAdmin(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.UserProfileUpdate](c)
	if !ok {
		return
	}

	user, err := h.users.UpdateAdmin(c.Request.Context(), id, usersvc.AdminUpdateParams{
		DisplayName: req.DisplayName,
		Password:    req.Password,
		Bio:         req.Bio,
		Telegram:    req.Telegram,
		Github:      req.Github,
		Email:       req.Email,
		Language:    enumStringPtr(req.Language),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, userToHTTPPrivate(*user))
}

func (h *Handler) HandleChangeUserPassword(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.PasswordUpdate](c)
	if !ok {
		return
	}

	if _, err := h.users.ChangePassword(c.Request.Context(), id, req.Password); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleSetUserBlocked(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[httpserver.UserBlockUpdate](c)
	if !ok {
		return
	}

	// Block is authoritative on the next request, so an admin blocking their own
	// account would immediately lock themselves out. Disallow it.
	if req.Blocked {
		if adminID, ok := middleware.CurrentUserID(c); ok && adminID == id {
			c.JSON(http.StatusForbidden, errorResponse{Code: codeForbidden, Message: "you cannot block your own account"})
			return
		}
	}

	user, err := h.users.SetBlocked(c.Request.Context(), id, req.Blocked)
	if err != nil {
		respondError(c, err)
		return
	}

	// Blocking must cut existing access; revoke all of the user's sessions.
	if req.Blocked {
		if err := h.auth.RevokeAllSessions(c.Request.Context(), id); err != nil {
			respondError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, userToHTTPPrivate(*user))
}

func (h *Handler) HandleUploadUserAvatar(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	h.handleUploadUserAvatar(c, id)
}

func (h *Handler) HandleUploadProfileAvatar(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	h.handleUploadUserAvatar(c, userID)
}

func (h *Handler) handleUploadUserAvatar(c *gin.Context, id int64) {
	if _, err := h.users.GetByID(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes+maxBytesReaderOverhead)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "avatar file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		respondError(c, err)
		return
	}
	defer f.Close()

	scaled, err := imageutil.ScaleAvatar(f, avatarMaxDimension)
	if err != nil {
		switch {
		case errors.Is(err, imageutil.ErrInvalidImage):
			c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "uploaded file is not a valid image"})
		case errors.Is(err, imageutil.ErrImageTooLarge):
			c.JSON(http.StatusUnprocessableEntity, errorResponse{Code: codeValidationError, Message: "image dimensions are too large"})
		default:
			respondError(c, err)
		}
		return
	}

	key := userAvatarKey(id)
	if _, err := h.fileStorage.Save(c.Request.Context(), key, bytes.NewReader(scaled)); err != nil {
		respondError(c, err)
		return
	}

	// Append a version so browsers refetch after a re-upload.
	url := fmt.Sprintf("/api/v1/users/%d/avatar?v=%d", id, time.Now().Unix())
	user, err := h.users.SetAvatar(c.Request.Context(), id, &url)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, userToHTTPPrivate(*user))
}

func (h *Handler) HandleGetUserAvatar(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	rc, err := h.fileStorage.Open(c.Request.Context(), userAvatarKey(id))
	if err != nil {
		if errors.Is(err, storage.ErrFileNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Code: codeNotFound, Message: "avatar not found"})
			return
		}
		respondError(c, err)
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "private, max-age=60")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, rc); err != nil {
		_ = c.Error(fmt.Errorf("copy avatar: %w", err))
	}
}

func (h *Handler) HandleListUserSessions(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	h.listUserSessions(c, id)
}

func (h *Handler) HandleListProfileSessions(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Code: codeUnauthorized, Message: msgNotAuthenticated})
		return
	}

	h.listUserSessions(c, userID)
}

func (h *Handler) listUserSessions(c *gin.Context, id int64) {
	if _, err := h.users.GetByID(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	currentJTI, _ := middleware.CurrentSessionJTI(c)
	sessions, err := h.auth.ListSessions(c.Request.Context(), id, currentJTI)
	if err != nil {
		respondError(c, err)
		return
	}

	items := make([]httpserver.UserSession, len(sessions))
	for i, s := range sessions {
		items[i] = httpserver.UserSession{
			Id:         s.ID,
			IpAddress:  s.IPAddress,
			UserAgent:  s.UserAgent,
			CreatedAt:  s.CreatedAt,
			LastSeenAt: s.LastSeenAt,
			ExpiresAt:  s.ExpiresAt,
			Current:    s.Current,
		}
	}
	c.JSON(http.StatusOK, httpserver.UserSessionList{Items: items})
}

// ServerInterface wiring for the admin user-management operations.

func (h *Handler) UpdateUserProfileAdmin(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUpdateUserProfileAdmin(c)
}

func (h *Handler) ChangeUserPassword(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleChangeUserPassword(c)
}

func (h *Handler) SetUserBlocked(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleSetUserBlocked(c)
}

func (h *Handler) UploadUserAvatar(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleUploadUserAvatar(c)
}

func (h *Handler) UploadProfileAvatar(c *gin.Context) {
	h.HandleUploadProfileAvatar(c)
}

func (h *Handler) GetUserAvatar(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleGetUserAvatar(c)
}

func (h *Handler) ListUserSessions(c *gin.Context, id int64) {
	c.Set("id", id)
	h.HandleListUserSessions(c)
}

func (h *Handler) ListProfileSessions(c *gin.Context) {
	h.HandleListProfileSessions(c)
}

func (h *Handler) RevokeUserSession(c *gin.Context, id int64, sessionId int64) {
	if err := h.auth.RevokeUserSession(c.Request.Context(), id, sessionId); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
