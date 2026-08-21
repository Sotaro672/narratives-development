// backend\internal\adapters\out\firebase\auth_user_reader.go
package firebase

import (
	"context"
	"errors"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type AuthUserReader struct {
	client *firebaseauth.Client
}

func NewAuthUserReader(
	client *firebaseauth.Client,
) *AuthUserReader {
	return &AuthUserReader{
		client: client,
	}
}

func (r *AuthUserReader) GetEmailByUID(
	ctx context.Context,
	uid string,
) (string, error) {
	if r == nil || r.client == nil {
		return "",
			errors.New(
				"firebase auth user reader is not configured",
			)
	}

	if uid == "" {
		return "",
			errors.New(
				"firebase auth uid is empty",
			)
	}

	userRecord, err :=
		r.client.GetUser(
			ctx,
			uid,
		)
	if err != nil {
		return "", err
	}

	if userRecord == nil {
		return "",
			errors.New(
				"firebase auth user record is nil",
			)
	}

	return userRecord.Email, nil
}
