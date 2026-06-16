package service

import (
	"net/http"

	"github.com/ekideno/zaplink/internal/apperror"
)

var (
	ErrInvalidURL   = apperror.New(http.StatusBadRequest, "invalid_url", "invalid url")
	ErrInactiveLink = apperror.New(http.StatusGone, "inactive_link", "link is inactive")
)
