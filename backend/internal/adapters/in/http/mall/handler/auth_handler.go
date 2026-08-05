// backend/internal/adapters/in/http/mall/handler/auth_handler.go
package mallHandler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"

	"narratives/internal/adapters/in/http/middleware"
)

const (
	authEmailVerificationSendPath = "/auth/email-verification/send"
	defaultAuthActionBaseURL      = "https://amol.jp"
)

type AuthVerificationMailer interface {
	SendVerificationEmail(
		ctx context.Context,
		toEmail string,
		verifyURL string,
	) error
}

type AuthHandler struct {
	FirebaseAuth  *auth.Client
	Mailer        AuthVerificationMailer
	ActionBaseURL string
}

func NewAuthHandler(
	firebaseAuth *auth.Client,
	mailer AuthVerificationMailer,
	actionBaseURL string,
) http.Handler {
	return &AuthHandler{
		FirebaseAuth: firebaseAuth,
		Mailer:       mailer,
		ActionBaseURL: strings.TrimRight(
			strings.TrimSpace(actionBaseURL),
			"/",
		),
	}
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodPost &&
		path0 == authEmailVerificationSendPath:
		h.handleSendEmailVerification(w, r)
		return

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})
		return
	}
}

func (h *AuthHandler) handleSendEmailVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil {
		writeAuthErr(w, errors.New("auth handler is nil"))
		return
	}

	if h.FirebaseAuth == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "firebase_auth_not_configured",
		})
		return
	}

	if h.Mailer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "auth_mailer_not_configured",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized: missing uid",
		})
		return
	}

	userRecord, err := h.FirebaseAuth.GetUser(r.Context(), uid)
	if err != nil {
		writeAuthErr(w, fmt.Errorf("get firebase user: %w", err))
		return
	}

	if userRecord.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "user_email_empty",
		})
		return
	}

	if userRecord.EmailVerified {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"message": "email_already_verified",
		})
		return
	}

	verifyURL, err := h.buildEmailVerificationLink(
		r.Context(),
		userRecord.Email,
	)
	if err != nil {
		writeAuthErr(w, err)
		return
	}

	if err := h.Mailer.SendVerificationEmail(
		r.Context(),
		userRecord.Email,
		verifyURL,
	); err != nil {
		writeAuthErr(w, fmt.Errorf("send verification email: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}

func (h *AuthHandler) buildEmailVerificationLink(
	ctx context.Context,
	email string,
) (string, error) {
	if h == nil || h.FirebaseAuth == nil {
		return "", errors.New("firebase auth is not configured")
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return "", errors.New("email is empty")
	}

	settings := &auth.ActionCodeSettings{
		URL:             h.authSignInURL(),
		HandleCodeInApp: false,
	}

	link, err := h.FirebaseAuth.EmailVerificationLinkWithSettings(
		ctx,
		email,
		settings,
	)
	if err != nil {
		return "", fmt.Errorf(
			"generate email verification link: %w",
			err,
		)
	}

	return link, nil
}

func (h *AuthHandler) authSignInURL() string {
	baseURL := defaultAuthActionBaseURL

	if h != nil && strings.TrimSpace(h.ActionBaseURL) != "" {
		baseURL = strings.TrimRight(
			strings.TrimSpace(h.ActionBaseURL),
			"/",
		)
	}

	baseURL = strings.TrimSuffix(baseURL, "/auth/action")

	if strings.HasSuffix(baseURL, "/signin") {
		return baseURL
	}

	return baseURL + "/signin"
}

func writeAuthErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	errorCode := "internal_error"

	switch {
	case err == nil:
		// Default response values are used.

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		code = http.StatusRequestTimeout
		errorCode = "request_timeout"

	default:
		log.Printf("[auth_handler] error: %v", err)
	}

	writeJSON(w, code, map[string]string{
		"error": errorCode,
	})
}
