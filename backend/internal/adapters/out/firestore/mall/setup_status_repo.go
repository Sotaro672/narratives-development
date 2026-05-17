// backend\internal\adapters\out\firestore\mall\setup_status_repo.go
package mall

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetupStatusRepoFirestore implements mall.SetupStatusRepo backed by Firestore.
//
// Strategy:
// - One-to-one setup documents are checked by document id == uid.
// - Avatar is checked by owner field because avatar document id is avatarId.
// - For payment method readiness, also check paymentMethodCustomers.
type SetupStatusRepoFirestore struct {
	Client *firestore.Client

	// Collection names (top-level collections by default)
	UserCollection                  string
	ShippingAddressCollection       string
	PaymentMethodCollection         string
	PaymentMethodCustomerCollection string
	AvatarCollection                string

	// Field name used to identify avatar owner.
	AvatarOwnerField string
}

const (
	defaultUserCollection                  = "users"
	defaultShippingAddressCollection       = "shippingAddresses"
	defaultPaymentMethodCollection         = "paymentMethods"
	defaultPaymentMethodCustomerCollection = "paymentMethodCustomers"
	defaultAvatarCollection                = "avatars"

	defaultAvatarOwnerField = "userId"
)

func NewSetupStatusRepoFirestore(client *firestore.Client) *SetupStatusRepoFirestore {
	return &SetupStatusRepoFirestore{
		Client:                          client,
		UserCollection:                  defaultUserCollection,
		ShippingAddressCollection:       defaultShippingAddressCollection,
		PaymentMethodCollection:         defaultPaymentMethodCollection,
		PaymentMethodCustomerCollection: defaultPaymentMethodCustomerCollection,
		AvatarCollection:                defaultAvatarCollection,
		AvatarOwnerField:                defaultAvatarOwnerField,
	}
}

func (r *SetupStatusRepoFirestore) HasUser(ctx context.Context, uid string) (bool, error) {
	return r.existsByDocID(ctx, r.UserCollection, uid)
}

func (r *SetupStatusRepoFirestore) HasShippingAddress(ctx context.Context, uid string) (bool, error) {
	return r.existsByDocID(ctx, r.ShippingAddressCollection, uid)
}

func (r *SetupStatusRepoFirestore) HasPaymentMethod(ctx context.Context, uid string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, nil
	}
	if uid == "" {
		return false, nil
	}

	hasPaymentMethodDoc, err := r.existsByDocID(ctx, r.PaymentMethodCollection, uid)
	if err != nil {
		return false, err
	}
	if hasPaymentMethodDoc {
		return true, nil
	}

	hasPaymentMethodCustomer, err := r.existsByDocID(ctx, r.PaymentMethodCustomerCollection, uid)
	if err != nil {
		return false, err
	}
	if hasPaymentMethodCustomer {
		return true, nil
	}

	return false, nil
}

func (r *SetupStatusRepoFirestore) HasAvatar(ctx context.Context, uid string) (bool, error) {
	return r.existsAvatarByOwner(ctx, uid)
}

// ------------------------------------------------------------
// Helpers

func (r *SetupStatusRepoFirestore) existsByDocID(
	ctx context.Context,
	collection string,
	uid string,
) (bool, error) {
	if r == nil || r.Client == nil {
		return false, nil
	}
	if collection == "" || uid == "" {
		return false, nil
	}

	_, err := r.Client.Collection(collection).Doc(uid).Get(ctx)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}

	return false, err
}

func (r *SetupStatusRepoFirestore) existsAvatarByOwner(
	ctx context.Context,
	uid string,
) (bool, error) {
	if r == nil || r.Client == nil {
		return false, nil
	}
	if r.AvatarCollection == "" || uid == "" {
		return false, nil
	}

	ownerField := r.AvatarOwnerField
	if ownerField == "" {
		ownerField = defaultAvatarOwnerField
	}

	iter := r.Client.Collection(r.AvatarCollection).
		Where(ownerField, "==", uid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	docs, err := iter.GetAll()
	if err != nil {
		return false, err
	}

	return len(docs) > 0, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.NotFound
}
