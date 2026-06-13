package service

import "errors"

var (
	ErrInvalidURL   = errors.New("invalid url")
	ErrInactiveLink = errors.New("link is inactive")
)
