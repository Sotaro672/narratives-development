// backend/internal/adapters/out/firestore/user_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	udom "narratives/internal/domain/user"
)

var _ udom.RepositoryPort = (*UserRepositoryFS)(nil)

// UserRepositoryFS implements the User persistence port with Firestore.
// The document ID is the authenticated user ID.
type UserRepositoryFS struct {
	Client *firestore.Client
}

func NewUserRepositoryFS(client *firestore.Client) *UserRepositoryFS {
	return &UserRepositoryFS{Client: client}
}

func (r *UserRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("users")
}

// optionalUserName converts an optional input field to the value expected by
// the User constructor. Validation remains the domain constructor's concern.
func optionalUserName(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func (r *UserRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, udom.ErrInvalidID
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}

		return nil, err
	}

	u, err := docToUser(snap)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetEmailByID returns users/{userID}.email for payment notifications.
// This method is used by the narrow payment-side reader contract.
func (r *UserRepositoryFS) GetEmailByID(
	ctx context.Context,
	userID string,
) (string, error) {
	if r == nil || r.Client == nil {
		return "", errors.New("firestore client is nil")
	}
	if userID == "" {
		return "", udom.ErrInvalidID
	}

	snap, err := r.col().Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", udom.ErrNotFound
		}

		return "", err
	}

	data := snap.Data()
	if data == nil {
		return "", udom.ErrNotFound
	}

	value, exists := data["email"]
	if !exists {
		return "", nil
	}

	email, ok := value.(string)
	if !ok {
		return "", nil
	}

	return email, nil
}

// Create creates users/{id}. Optional name fields are converted to empty
// strings and validated by the User domain constructor before persistence.
func (r *UserRepositoryFS) Create(
	ctx context.Context,
	id string,
	in udom.CreateUserInput,
) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, udom.ErrInvalidID
	}

	firstName := optionalUserName(in.FirstName)
	firstNameKana := optionalUserName(in.FirstNameKana)
	lastNameKana := optionalUserName(in.LastNameKana)
	lastName := optionalUserName(in.LastName)

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	u, err := udom.New(
		id,
		firstName,
		firstNameKana,
		lastNameKana,
		lastName,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"first_name":      u.FirstName,
		"first_name_kana": u.FirstNameKana,
		"last_name_kana":  u.LastNameKana,
		"last_name":       u.LastName,
		"createdAt":       u.CreatedAt,
		"updatedAt":       u.UpdatedAt,
	}

	ref := r.col().Doc(id)
	if _, err := ref.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, udom.ErrConflict
		}

		return nil, err
	}

	return r.GetByID(ctx, id)
}

// Update partially updates users/{id}. A nil name field means no change.
func (r *UserRepositoryFS) Update(
	ctx context.Context,
	id string,
	in udom.UpdateUserInput,
) (*udom.User, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}
	if id == "" {
		return nil, udom.ErrInvalidID
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}

		return nil, err
	}

	updates := make([]firestore.Update, 0, 5)

	setStringUpdate := func(
		path string,
		value *string,
		invalidErr error,
	) error {
		if value == nil {
			return nil
		}

		if len([]rune(*value)) > udom.MaxNameLength {
			return invalidErr
		}

		updates = append(updates, firestore.Update{
			Path:  path,
			Value: *value,
		})

		return nil
	}

	if err := setStringUpdate(
		"first_name",
		in.FirstName,
		udom.ErrInvalidFirstName,
	); err != nil {
		return nil, err
	}
	if err := setStringUpdate(
		"first_name_kana",
		in.FirstNameKana,
		udom.ErrInvalidFirstNameKana,
	); err != nil {
		return nil, err
	}
	if err := setStringUpdate(
		"last_name_kana",
		in.LastNameKana,
		udom.ErrInvalidLastNameKana,
	); err != nil {
		return nil, err
	}
	if err := setStringUpdate(
		"last_name",
		in.LastName,
		udom.ErrInvalidLastName,
	); err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC()
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: updatedAt,
	})

	if _, err := ref.Update(ctx, updates); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, udom.ErrNotFound
		}

		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	if r == nil || r.Client == nil {
		return errors.New("firestore client is nil")
	}
	if id == "" {
		return udom.ErrInvalidID
	}

	ref := r.col().Doc(id)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return udom.ErrNotFound
		}

		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return udom.ErrNotFound
		}

		return err
	}

	return nil
}

func docToUser(
	doc *firestore.DocumentSnapshot,
) (udom.User, error) {
	if doc == nil || doc.Ref == nil || !doc.Exists() {
		return udom.User{}, udom.ErrNotFound
	}

	data := doc.Data()
	if data == nil {
		return udom.User{}, udom.ErrNotFound
	}

	getString := func(key string) string {
		value, exists := data[key]
		if !exists {
			return ""
		}

		result, ok := value.(string)
		if !ok {
			return ""
		}

		return result
	}

	getTime := func(key string) time.Time {
		value, exists := data[key]
		if !exists {
			return time.Time{}
		}

		result, ok := value.(time.Time)
		if !ok {
			return time.Time{}
		}

		return result.UTC()
	}

	return udom.New(
		doc.Ref.ID,
		getString("first_name"),
		getString("first_name_kana"),
		getString("last_name_kana"),
		getString("last_name"),
		getTime("createdAt"),
		getTime("updatedAt"),
	)
}
