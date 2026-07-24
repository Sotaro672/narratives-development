// backend/internal/adapters/in/http/console/handler/invitation_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	compdom "narratives/internal/domain/company"
	invdom "narratives/internal/domain/invitation"
	memdom "narratives/internal/domain/member"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

/*
InvitationHandler
- POST /invitations
- POST /invitations/validate
- POST /invitations/complete
招待メールの送信はPOST /invitationsのみに固定する。
招待完了時のUIDとemailはclient bodyから受け取らず、
検証済みFirebase ID tokenから取得する。
*/
type InvitationHandler struct {
	InvitationUC usecase.InvitationUsecasePort
	CompanyRepo  compdom.Repository
	BrandRepo    branddom.Repository
	FirebaseAuth *firebaseauth.Client
}

func NewInvitationHandler(
	invitationUC usecase.InvitationUsecasePort,
	companyRepo compdom.Repository,
	brandRepo branddom.Repository,
	firebaseAuth *firebaseauth.Client,
) *InvitationHandler {
	return &InvitationHandler{
		InvitationUC: invitationUC,
		CompanyRepo:  companyRepo,
		BrandRepo:    brandRepo,
		FirebaseAuth: firebaseAuth,
	}
}

type invitationValidateResponse struct {
	CompanyName string   `json:"companyName,omitempty"`
	BrandNames  []string `json:"brandNames,omitempty"`
}
type invitationValidateRequest struct {
	Token string `json:"token"`
}
type createInvitationRequest struct {
	MemberID string `json:"memberId"`
}
type createInvitationResponse struct {
	MemberID string `json:"memberId"`
}
type invitationCompleteRequest struct {
	Token         string `json:"token"`
	LastName      string `json:"lastName"`
	LastNameKana  string `json:"lastNameKana"`
	FirstName     string `json:"firstName"`
	FirstNameKana string `json:"firstNameKana"`
}
type invitationCompleteResponse struct {
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}
type invitationIdentity struct {
	UID   string
	Email string
}
type invitationIdentityContextKey struct{}

func (h *InvitationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	switch r.URL.Path {
	case "/invitations":
		h.handleCreateInvitation(w, r)
	case "/invitations/validate":
		h.handleResolveInfo(w, r)
	case "/invitations/complete":
		h.handleComplete(w, r)
	default:
		writeInvitationJSONError(
			w,
			http.StatusNotFound,
			"not_found",
		)
	}
}
func (h *InvitationHandler) handleCreateInvitation(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}
	if h.InvitationUC == nil {
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"invitation_usecase_not_configured",
		)
		return
	}
	var req createInvitationRequest
	if err := decodeInvitationJSON(r, &req); err != nil {
		writeInvitationJSONError(
			w,
			http.StatusBadRequest,
			"invalid_body",
		)
		return
	}
	memberID := strings.TrimSpace(req.MemberID)
	if memberID == "" {
		writeInvitationJSONError(
			w,
			http.StatusBadRequest,
			"memberId_required",
		)
		return
	}
	err := h.InvitationUC.CreateInvitationAndSend(
		r.Context(),
		memberID,
	)
	if err != nil {
		if errors.Is(err, memdom.ErrNotFound) {
			writeInvitationJSONError(
				w,
				http.StatusNotFound,
				"member_not_found",
			)
			return
		}
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"cannot_send_invitation",
		)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(
		createInvitationResponse{
			MemberID: memberID,
		},
	)
}
func (h *InvitationHandler) handleResolveInfo(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}
	if h.InvitationUC == nil {
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"invitation_usecase_not_configured",
		)
		return
	}
	var req invitationValidateRequest
	if err := decodeInvitationJSON(r, &req); err != nil {
		writeInvitationJSONError(
			w,
			http.StatusBadRequest,
			"invalid_body",
		)
		return
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeInvitationJSONError(
			w,
			http.StatusBadRequest,
			"token_required",
		)
		return
	}
	ctx := r.Context()
	info, err := h.InvitationUC.GetInvitationInfo(
		ctx,
		token,
	)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrInvitationTokenNotFound,
		) || errors.Is(err, memdom.ErrNotFound) {
			writeInvitationJSONError(
				w,
				http.StatusNotFound,
				"invitation_token_not_found",
			)
			return
		}
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"failed_to_resolve_invitation_token",
		)
		return
	}
	companyName := info.CompanyID
	if h.CompanyRepo != nil && info.CompanyID != "" {
		companyEntity, err := h.CompanyRepo.GetByID(
			ctx,
			info.CompanyID,
		)
		if err != nil {
			if !errors.Is(err, compdom.ErrNotFound) {
				writeInvitationJSONError(
					w,
					http.StatusInternalServerError,
					"failed_to_resolve_company_name",
				)
				return
			}
		} else if companyEntity.Name != "" {
			companyName = companyEntity.Name
		}
	}
	brandNames := info.AssignedBrandIDs
	if h.BrandRepo != nil &&
		len(info.AssignedBrandIDs) > 0 {
		resolved := make(
			[]string,
			0,
			len(info.AssignedBrandIDs),
		)
		for _, rawBrandID := range info.AssignedBrandIDs {
			brandID := strings.TrimSpace(rawBrandID)
			if brandID == "" {
				continue
			}
			brand, err := h.BrandRepo.GetByID(
				ctx,
				brandID,
			)
			if err != nil {
				if errors.Is(
					err,
					branddom.ErrNotFound,
				) || errors.Is(
					err,
					branddom.ErrInvalidID,
				) {
					resolved = append(
						resolved,
						brandID,
					)
					continue
				}
				writeInvitationJSONError(
					w,
					http.StatusInternalServerError,
					"failed_to_resolve_brand_name",
				)
				return
			}
			brandName := strings.TrimSpace(brand.Name)
			if brandName == "" {
				brandName = brandID
			}
			resolved = append(
				resolved,
				brandName,
			)
		}
		brandNames = resolved
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(
		invitationValidateResponse{
			CompanyName: companyName,
			BrandNames:  brandNames,
		},
	)
}
func (h *InvitationHandler) handleComplete(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeInvitationJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}
	h.withInvitationIdentity(
		http.HandlerFunc(h.handleCompleteVerified),
	).ServeHTTP(w, r)
}
func (h *InvitationHandler) handleCompleteVerified(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h.InvitationUC == nil {
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"invitation_complete_usecase_not_configured",
		)
		return
	}
	identity, ok := invitationIdentityFromContext(
		r.Context(),
	)
	if !ok {
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"verified_identity_required",
		)
		return
	}
	var req invitationCompleteRequest
	if err := decodeInvitationJSON(r, &req); err != nil {
		writeInvitationJSONError(
			w,
			http.StatusBadRequest,
			"invalid_body",
		)
		return
	}
	token := strings.TrimSpace(req.Token)
	info, err := h.InvitationUC.GetInvitationInfo(
		r.Context(),
		token,
	)
	if err != nil {
		if errors.Is(
			err,
			invdom.ErrInvitationTokenNotFound,
		) || errors.Is(err, memdom.ErrNotFound) {
			writeInvitationJSONError(
				w,
				http.StatusNotFound,
				"invitation_token_or_member_not_found",
			)
			return
		}
		writeInvitationJSONError(
			w,
			http.StatusInternalServerError,
			"failed_to_resolve_invitation_token",
		)
		return
	}
	input := usecase.CompleteInvitationInput{
		Token: token,
		UID:   identity.UID,
		LastName: strings.TrimSpace(
			req.LastName,
		),
		LastNameKana: strings.TrimSpace(
			req.LastNameKana,
		),
		FirstName: strings.TrimSpace(
			req.FirstName,
		),
		FirstNameKana: strings.TrimSpace(
			req.FirstNameKana,
		),
		Email: identity.Email,
	}
	err = h.InvitationUC.CompleteInvitation(
		r.Context(),
		input,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			invdom.ErrInvitationTokenNotFound,
		),
			errors.Is(err, memdom.ErrNotFound):
			writeInvitationJSONError(
				w,
				http.StatusNotFound,
				"invitation_token_or_member_not_found",
			)
		case err.Error() == "token_or_uid_required":
			writeInvitationJSONError(
				w,
				http.StatusBadRequest,
				"token_or_uid_required",
			)
		case err.Error() == "name_fields_required":
			writeInvitationJSONError(
				w,
				http.StatusBadRequest,
				"name_fields_required",
			)
		case err.Error() == "email_required":
			writeInvitationJSONError(
				w,
				http.StatusBadRequest,
				"email_required",
			)
		case err.Error() == "email_mismatch":
			writeInvitationJSONError(
				w,
				http.StatusForbidden,
				"email_mismatch",
			)
		case err.Error() == "firebase_uid_already_in_use":
			writeInvitationJSONError(
				w,
				http.StatusConflict,
				"firebase_uid_already_in_use",
			)
		default:
			writeInvitationJSONError(
				w,
				http.StatusInternalServerError,
				"failed_to_complete_invitation",
			)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(
		invitationCompleteResponse{
			Email: identity.Email,
			Permissions: append(
				[]string(nil),
				info.Permissions...,
			),
		},
	)
}
func (h *InvitationHandler) withInvitationIdentity(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			if h.FirebaseAuth == nil {
				writeInvitationJSONError(
					w,
					http.StatusInternalServerError,
					"firebase_auth_not_configured",
				)
				return
			}
			identity, err := h.verifyInvitationIdentity(r)
			if err != nil {
				writeInvitationIdentityError(w, err)
				return
			}
			ctx := context.WithValue(
				r.Context(),
				invitationIdentityContextKey{},
				identity,
			)
			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}
func invitationIdentityFromContext(
	ctx context.Context,
) (invitationIdentity, bool) {
	identity, ok := ctx.Value(
		invitationIdentityContextKey{},
	).(invitationIdentity)
	if !ok ||
		strings.TrimSpace(identity.UID) == "" ||
		strings.TrimSpace(identity.Email) == "" {
		return invitationIdentity{}, false
	}
	return identity, true
}
func writeInvitationIdentityError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		errInvitationAuthorizationRequired,
	):
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"authorization_required",
		)
	case errors.Is(
		err,
		errInvitationInvalidIDToken,
	):
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"invalid_id_token",
		)
	case errors.Is(
		err,
		errInvitationUIDRequired,
	):
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"authenticated_uid_required",
		)
	case errors.Is(
		err,
		errInvitationEmailRequired,
	):
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"authenticated_email_required",
		)
	default:
		writeInvitationJSONError(
			w,
			http.StatusUnauthorized,
			"invalid_id_token",
		)
	}
}

var (
	errInvitationAuthorizationRequired = errors.New(
		"invitation authorization required",
	)
	errInvitationInvalidIDToken = errors.New(
		"invitation invalid id token",
	)
	errInvitationUIDRequired = errors.New(
		"invitation authenticated uid required",
	)
	errInvitationEmailRequired = errors.New(
		"invitation authenticated email required",
	)
)

func (h *InvitationHandler) verifyInvitationIdentity(
	r *http.Request,
) (invitationIdentity, error) {
	idToken, err := invitationBearerToken(r)
	if err != nil {
		return invitationIdentity{}, err
	}
	token, err := h.FirebaseAuth.VerifyIDToken(
		r.Context(),
		idToken,
	)
	if err != nil {
		return invitationIdentity{},
			errInvitationInvalidIDToken
	}
	uid := strings.TrimSpace(token.UID)
	if uid == "" {
		return invitationIdentity{},
			errInvitationUIDRequired
	}
	email, ok := token.Claims["email"].(string)
	if !ok {
		return invitationIdentity{},
			errInvitationEmailRequired
	}
	email = strings.ToLower(
		strings.TrimSpace(email),
	)
	if email == "" {
		return invitationIdentity{},
			errInvitationEmailRequired
	}
	return invitationIdentity{
		UID:   uid,
		Email: email,
	}, nil
}
func invitationBearerToken(
	r *http.Request,
) (string, error) {
	const prefix = "Bearer "
	authorization := strings.TrimSpace(
		r.Header.Get("Authorization"),
	)
	if !strings.HasPrefix(
		authorization,
		prefix,
	) {
		return "",
			errInvitationAuthorizationRequired
	}
	idToken := strings.TrimSpace(
		strings.TrimPrefix(
			authorization,
			prefix,
		),
	)
	if idToken == "" {
		return "",
			errInvitationAuthorizationRequired
	}
	return idToken, nil
}
func decodeInvitationJSON(
	r *http.Request,
	destination any,
) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New(
			"multiple JSON values are not allowed",
		)
	}
	return err
}
func writeInvitationJSONError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"error": message,
		},
	)
}
