// backend/internal/application/port/payout_account_cipher.go
package port

import "context"

// PayoutAccountCipher encrypts and decrypts payout bank account numbers.
//
// Plaintext account numbers are sensitive transient data and must never be:
//   - persisted in Firestore
//   - written to logs
//   - included in error messages
//   - returned to the browser
//
// userID is supplied as part of the cryptographic context so that ciphertext
// belongs to the user for whom it was created.
type PayoutAccountCipher interface {
	Encrypt(
		ctx context.Context,
		userID string,
		accountNumber string,
	) (string, error)

	Decrypt(
		ctx context.Context,
		userID string,
		accountNumberCiphertext string,
	) (string, error)
}
