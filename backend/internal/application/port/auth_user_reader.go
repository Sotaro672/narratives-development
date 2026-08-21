// backend\internal\application\port\auth_user_reader.go
package port

import "context"

type AuthUserReader interface {
	GetEmailByUID(
		ctx context.Context,
		uid string,
	) (string, error)
}
