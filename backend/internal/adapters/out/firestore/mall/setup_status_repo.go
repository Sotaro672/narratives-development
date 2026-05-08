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
// - First try "document id == uid" (fast path)
// - If not found, fallback to query where field "userId" == uid (limit 1)
// - For payment method readiness, also check paymentMethodCustomers
type SetupStatusRepoFirestore struct {
	Client *firestore.Client

	// Collection names (top-level collections by default)
	UserCollection                  string
	ShippingAddressCollection       string
	PaymentMethodCollection         string
	PaymentMethodCustomerCollection string
	AvatarCollection                string

	// Field name used in fallback query
	UIDField string
}

const (
	defaultUserCollection                  = "users"
	defaultShippingAddressCollection       = "shippingAddresses"
	defaultPaymentMethodCollection         = "paymentMethods"
	defaultPaymentMethodCustomerCollection = "paymentMethodCustomers"
	defaultAvatarCollection                = "avatars"

	// uid field is "userId"
	defaultUIDField = "userId"
)

func NewSetupStatusRepoFirestore(client *firestore.Client) *SetupStatusRepoFirestore {
	return &SetupStatusRepoFirestore{
		Client:                          client,
		UserCollection:                  defaultUserCollection,
		ShippingAddressCollection:       defaultShippingAddressCollection,
		PaymentMethodCollection:         defaultPaymentMethodCollection,
		PaymentMethodCustomerCollection: defaultPaymentMethodCustomerCollection,
		AvatarCollection:                defaultAvatarCollection,
		UIDField:                        defaultUIDField,
	}
}

func (r *SetupStatusRepoFirestore) HasUser(ctx context.Context, uid string) (bool, error) {
	return r.existsByDocIDOrUIDField(ctx, r.UserCollection, uid)
}

func (r *SetupStatusRepoFirestore) HasShippingAddress(ctx context.Context, uid string) (bool, error) {
	return r.existsByDocIDOrUIDField(ctx, r.ShippingAddressCollection, uid)
}

func (r *SetupStatusRepoFirestore) HasPaymentMethod(ctx context.Context, uid string) (bool, error) {
	if r == nil || r.Client == nil {
		return false, nil
	}
	if uid == "" {
		return false, nil
	}

	// 1) actual saved payment method docs
	hasPaymentMethodDoc, err := r.existsByDocIDOrUIDField(ctx, r.PaymentMethodCollection, uid)
	if err != nil {
		return false, err
	}
	if hasPaymentMethodDoc {
		return true, nil
	}

	// 2) stripe customer mapping for setup-intent flow
	hasPaymentMethodCustomer, err := r.existsByDocIDOrUIDField(ctx, r.PaymentMethodCustomerCollection, uid)
	if err != nil {
		return false, err
	}
	if hasPaymentMethodCustomer {
		return true, nil
	}

	return false, nil
}

func (r *SetupStatusRepoFirestore) HasAvatar(ctx context.Context, uid string) (bool, error) {
	return r.existsByDocIDOrUIDField(ctx, r.AvatarCollection, uid)
}

// ------------------------------------------------------------
// Helpers

func (r *SetupStatusRepoFirestore) existsByDocIDOrUIDField(
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

	// 1) Fast path: doc ID == uid
	_, err := r.Client.Collection(collection).Doc(uid).Get(ctx)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		// fallthrough to query
	} else {
		// other errors (permission, unavailable, etc.)
		return false, err
	}

	// 2) Fallback: query where userId field matches
	uidField := r.UIDField
	if uidField == "" {
		uidField = defaultUIDField
	}

	it := r.Client.Collection(collection).
		Where(uidField, "==", uid).
		Limit(1).
		Documents(ctx)

	docs, err := it.GetAll()
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
