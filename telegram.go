package main

import (
	"context"
	"errors"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"strings"
)

var ErrTelegramAuthTimeout = errors.New("authentication operation timed out")

// noSignUp can be embedded to prevent signing up.
type noSignUp struct{}

func (c noSignUp) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("not implemented")
}

func (c noSignUp) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

// HTTPAuthenticator implements authentication steps.
type HTTPAuthenticator struct {
	noSignUp
	phoneNumber  string
	passwordChan <-chan string
	codeChan     <-chan string
}

func (h *HTTPAuthenticator) Phone(_ context.Context) (string, error) {
	return h.phoneNumber, nil
}

func (h *HTTPAuthenticator) Password(ctx context.Context) (string, error) {
	select {
	case password := <-h.passwordChan:
		return strings.TrimSpace(password), nil
	case <-ctx.Done():
		return "", ErrTelegramAuthTimeout
	}
}

func (h *HTTPAuthenticator) Code(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	select {
	case code := <-h.codeChan:
		return strings.TrimSpace(code), nil
	case <-ctx.Done():
		return "", ErrTelegramAuthTimeout
	}
}
