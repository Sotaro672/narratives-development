// backend/internal/adapters/out/firebase/auth_user_reader.go

package firebase

import (
	"context"
	"errors"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type AuthUserReader struct {
	client *firebaseauth.Client
}

func NewAuthUserReader(client *firebaseauth.Client) *AuthUserReader {
	return &AuthUserReader{
		client: client,
	}
}

func (r *AuthUserReader) GetEmailByUID(ctx context.Context, uid string) (string, error) {
	userRecord, err := r.getUserByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	return userRecord.Email, nil
}

func (r *AuthUserReader) GetDisplayNameByUID(ctx context.Context, uid string) (string, error) {
	userRecord, err := r.getUserByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	if userRecord.DisplayName != "" {
		return userRecord.DisplayName, nil
	}

	return userRecord.Email, nil
}

func (r *AuthUserReader) getUserByUID(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("firebase auth user reader is not configured")
	}

	if uid == "" {
		return nil, errors.New("firebase auth uid is empty")
	}

	userRecord, err := r.client.GetUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	if userRecord == nil {
		return nil, errors.New("firebase auth user record is nil")
	}

	return userRecord, nil
}
