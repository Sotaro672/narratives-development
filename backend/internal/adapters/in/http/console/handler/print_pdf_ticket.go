// backend/internal/adapters/in/http/console/handler/print_pdf_ticket.go
package consoleHandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	printPDFTicketVersion = 1

	defaultPrintPDFTicketTTL = 5 * time.Minute
	maxPrintPDFTicketTTL     = 10 * time.Minute

	minPrintPDFTicketSecretBytes = 32
)

var (
	ErrPrintPDFTicketSecretRequired = errors.New("print pdf ticket: signing secret is required")
	ErrPrintPDFTicketSecretTooShort = errors.New("print pdf ticket: signing secret is too short")
	ErrPrintPDFTicketInvalidTTL     = errors.New("print pdf ticket: invalid ttl")
	ErrPrintPDFTicketInvalid        = errors.New("print pdf ticket: invalid ticket")
	ErrPrintPDFTicketExpired        = errors.New("print pdf ticket: expired")
	ErrPrintPDFTicketProductionID   = errors.New("print pdf ticket: productionId is required")
	ErrPrintPDFTicketCompanyID      = errors.New("print pdf ticket: companyId is required")
)

type PrintPDFTicketClaims struct {
	Version      int    `json:"v"`
	ProductionID string `json:"productionId"`
	CompanyID    string `json:"companyId"`
	ExpiresAt    int64  `json:"expiresAt"`
}

type PrintPDFTicketSigner struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

/**
 * NewPrintPDFTicketSigner は QR PDF 表示用の署名チケット発行器を生成する。
 *
 * secret:
 * - HMAC-SHA256 用の秘密鍵
 * - 32 bytes 以上を必須とする
 * - リポジトリへハードコードせず、Cloud Run の secret/env から渡す
 *
 * ttl:
 * - 0 の場合は 5 分
 * - 最大 10 分
 */
func NewPrintPDFTicketSigner(
	secret string,
	ttl time.Duration,
) (*PrintPDFTicketSigner, error) {
	normalizedSecret := strings.TrimSpace(secret)
	if normalizedSecret == "" {
		return nil, ErrPrintPDFTicketSecretRequired
	}

	if len([]byte(normalizedSecret)) < minPrintPDFTicketSecretBytes {
		return nil, ErrPrintPDFTicketSecretTooShort
	}

	if ttl == 0 {
		ttl = defaultPrintPDFTicketTTL
	}

	if ttl <= 0 || ttl > maxPrintPDFTicketTTL {
		return nil, ErrPrintPDFTicketInvalidTTL
	}

	return &PrintPDFTicketSigner{
		secret: []byte(normalizedSecret),
		ttl:    ttl,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

/**
 * Issue は productionId / companyId を含む短時間有効の署名チケットを発行する。
 *
 * 戻り値は URL query にそのまま使用できる Base64URL 文字列。
 *
 * 形式:
 *
 *   <base64url(payload)>.<base64url(hmac-sha256)>
 */
func (s *PrintPDFTicketSigner) Issue(
	productionID string,
	companyID string,
) (string, error) {
	if s == nil || len(s.secret) == 0 {
		return "", ErrPrintPDFTicketSecretRequired
	}

	productionID = normalizePrintPDFTicketValue(productionID)
	if productionID == "" {
		return "", ErrPrintPDFTicketProductionID
	}

	companyID = normalizePrintPDFTicketValue(companyID)
	if companyID == "" {
		return "", ErrPrintPDFTicketCompanyID
	}

	now := s.currentTime()

	claims := PrintPDFTicketClaims{
		Version:      printPDFTicketVersion,
		ProductionID: productionID,
		CompanyID:    companyID,
		ExpiresAt:    now.Add(s.ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf(
			"print pdf ticket: marshal claims: %w",
			err,
		)
	}

	encodedPayload :=
		base64.RawURLEncoding.EncodeToString(payload)

	signature :=
		s.sign(encodedPayload)

	encodedSignature :=
		base64.RawURLEncoding.EncodeToString(signature)

	return encodedPayload + "." + encodedSignature, nil
}

/**
 * Verify はチケットの署名・形式・有効期限を検証し、
 * 正常な場合のみ claims を返す。
 *
 * productionId / companyId は署名対象なので、
 * クライアント側で改ざんすると検証に失敗する。
 */
func (s *PrintPDFTicketSigner) Verify(
	ticket string,
) (PrintPDFTicketClaims, error) {
	var empty PrintPDFTicketClaims

	if s == nil || len(s.secret) == 0 {
		return empty, ErrPrintPDFTicketSecretRequired
	}

	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return empty, ErrPrintPDFTicketInvalid
	}

	parts := strings.Split(ticket, ".")
	if len(parts) != 2 {
		return empty, ErrPrintPDFTicketInvalid
	}

	encodedPayload :=
		strings.TrimSpace(parts[0])

	encodedSignature :=
		strings.TrimSpace(parts[1])

	if encodedPayload == "" || encodedSignature == "" {
		return empty, ErrPrintPDFTicketInvalid
	}

	signature, err :=
		base64.RawURLEncoding.DecodeString(
			encodedSignature,
		)
	if err != nil {
		return empty, ErrPrintPDFTicketInvalid
	}

	expectedSignature :=
		s.sign(encodedPayload)

	if !hmac.Equal(
		signature,
		expectedSignature,
	) {
		return empty, ErrPrintPDFTicketInvalid
	}

	payload, err :=
		base64.RawURLEncoding.DecodeString(
			encodedPayload,
		)
	if err != nil {
		return empty, ErrPrintPDFTicketInvalid
	}

	var claims PrintPDFTicketClaims

	if err := json.Unmarshal(
		payload,
		&claims,
	); err != nil {
		return empty, ErrPrintPDFTicketInvalid
	}

	if claims.Version != printPDFTicketVersion {
		return empty, ErrPrintPDFTicketInvalid
	}

	claims.ProductionID =
		normalizePrintPDFTicketValue(
			claims.ProductionID,
		)

	if claims.ProductionID == "" {
		return empty, ErrPrintPDFTicketInvalid
	}

	claims.CompanyID =
		normalizePrintPDFTicketValue(
			claims.CompanyID,
		)

	if claims.CompanyID == "" {
		return empty, ErrPrintPDFTicketInvalid
	}

	if claims.ExpiresAt <= 0 {
		return empty, ErrPrintPDFTicketInvalid
	}

	now := s.currentTime()

	expiresAt :=
		time.Unix(
			claims.ExpiresAt,
			0,
		).UTC()

	if !expiresAt.After(now) {
		return empty, ErrPrintPDFTicketExpired
	}

	// signer が発行可能な最大TTLを大幅に超える有効期限を持つ
	// チケットは、署名済みであっても設定不整合として拒否する。
	if expiresAt.After(
		now.Add(maxPrintPDFTicketTTL),
	) {
		return empty, ErrPrintPDFTicketInvalid
	}

	return claims, nil
}

func (s *PrintPDFTicketSigner) sign(
	encodedPayload string,
) []byte {
	mac :=
		hmac.New(
			sha256.New,
			s.secret,
		)

	_, _ = mac.Write(
		[]byte(encodedPayload),
	)

	return mac.Sum(nil)
}

func (s *PrintPDFTicketSigner) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}

	return time.Now().UTC()
}

func normalizePrintPDFTicketValue(
	value string,
) string {
	return strings.Trim(
		value,
		" \t\r\n/",
	)
}
