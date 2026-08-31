// backend/internal/adapters/out/kms/payout_account_cipher.go
package kms

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	cloudkms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"

	applicationport "narratives/internal/application/port"
)

const payoutAccountAADPrefix = "amol:payout-account:"

var (
	ErrPayoutAccountCipherClientMissing = errors.New(
		"payoutAccount cipher: KMS client is not configured",
	)
	ErrPayoutAccountCipherKeyMissing = errors.New(
		"payoutAccount cipher: KMS key name is not configured",
	)
	ErrPayoutAccountCipherInvalidUserID = errors.New(
		"payoutAccount cipher: invalid userId",
	)
	ErrPayoutAccountCipherInvalidPlaintext = errors.New(
		"payoutAccount cipher: invalid plaintext",
	)
	ErrPayoutAccountCipherInvalidCiphertext = errors.New(
		"payoutAccount cipher: invalid ciphertext",
	)
	ErrPayoutAccountCipherEncryptResultEmpty = errors.New(
		"payoutAccount cipher: KMS encrypt result is empty",
	)
	ErrPayoutAccountCipherDecryptResultEmpty = errors.New(
		"payoutAccount cipher: KMS decrypt result is empty",
	)
)

// PayoutAccountCipher encrypts and decrypts payout bank account numbers with
// Google Cloud KMS.
//
// The KMS CryptoKey name must have the form:
//
//	projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{key}
//
// userID is bound to each ciphertext through Additional Authenticated Data.
// Therefore a ciphertext encrypted for one user cannot be successfully
// decrypted with another user's userID.
//
// The plaintext account number must never be logged, persisted, included in an
// error message, or otherwise leave this adapter except as the return value of
// Decrypt for an authorized backend payout operation.
type PayoutAccountCipher struct {
	client  *cloudkms.KeyManagementClient
	keyName string
}

var _ applicationport.PayoutAccountCipher = (*PayoutAccountCipher)(nil)

// NewPayoutAccountCipher creates a Google Cloud KMS-backed payout account
// cipher.
//
// The caller owns the KMS client lifecycle. PayoutAccountCipher does not close
// the supplied client.
func NewPayoutAccountCipher(
	client *cloudkms.KeyManagementClient,
	keyName string,
) (*PayoutAccountCipher, error) {
	if client == nil {
		return nil, ErrPayoutAccountCipherClientMissing
	}
	if keyName == "" {
		return nil, ErrPayoutAccountCipherKeyMissing
	}

	return &PayoutAccountCipher{
		client:  client,
		keyName: keyName,
	}, nil
}

// Encrypt encrypts one plaintext bank account number.
//
// The returned value is standard Base64-encoded KMS ciphertext and is safe to
// persist as AccountNumberCiphertext.
func (c *PayoutAccountCipher) Encrypt(
	ctx context.Context,
	userID string,
	accountNumber string,
) (string, error) {
	if err := c.validateReady(); err != nil {
		return "", err
	}
	if !isValidCipherUserID(userID) {
		return "", ErrPayoutAccountCipherInvalidUserID
	}
	if accountNumber == "" {
		return "", ErrPayoutAccountCipherInvalidPlaintext
	}

	result, err := c.client.Encrypt(
		ctx,
		&kmspb.EncryptRequest{
			Name:                        c.keyName,
			Plaintext:                   []byte(accountNumber),
			AdditionalAuthenticatedData: payoutAccountAAD(userID),
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"payoutAccount cipher: KMS encrypt failed: %w",
			err,
		)
	}
	if result == nil || len(result.Ciphertext) == 0 {
		return "", ErrPayoutAccountCipherEncryptResultEmpty
	}

	return base64.StdEncoding.EncodeToString(result.Ciphertext), nil
}

// Decrypt decrypts one persisted Base64-encoded KMS ciphertext.
//
// The same userID supplied during Encrypt must be supplied here. A different
// userID changes the Additional Authenticated Data and KMS rejects decryption.
func (c *PayoutAccountCipher) Decrypt(
	ctx context.Context,
	userID string,
	accountNumberCiphertext string,
) (string, error) {
	if err := c.validateReady(); err != nil {
		return "", err
	}
	if !isValidCipherUserID(userID) {
		return "", ErrPayoutAccountCipherInvalidUserID
	}
	if accountNumberCiphertext == "" {
		return "", ErrPayoutAccountCipherInvalidCiphertext
	}

	ciphertext, err := base64.StdEncoding.DecodeString(
		accountNumberCiphertext,
	)
	if err != nil || len(ciphertext) == 0 {
		return "", ErrPayoutAccountCipherInvalidCiphertext
	}

	result, err := c.client.Decrypt(
		ctx,
		&kmspb.DecryptRequest{
			Name:                        c.keyName,
			Ciphertext:                  ciphertext,
			AdditionalAuthenticatedData: payoutAccountAAD(userID),
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"payoutAccount cipher: KMS decrypt failed: %w",
			err,
		)
	}
	if result == nil || len(result.Plaintext) == 0 {
		return "", ErrPayoutAccountCipherDecryptResultEmpty
	}

	return string(result.Plaintext), nil
}

func (c *PayoutAccountCipher) validateReady() error {
	if c == nil || c.client == nil {
		return ErrPayoutAccountCipherClientMissing
	}
	if c.keyName == "" {
		return ErrPayoutAccountCipherKeyMissing
	}

	return nil
}

func payoutAccountAAD(userID string) []byte {
	return []byte(payoutAccountAADPrefix + userID)
}

func isValidCipherUserID(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		switch r {
		case ' ', '\t', '\r', '\n':
			return false
		}
	}

	return true
}
