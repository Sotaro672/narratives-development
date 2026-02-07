// backend\internal\application\usecase\avatar\usecase.go
package avatar

import "time"

type AvatarUsecase struct {
	avRepo AvatarRepo
	stRepo AvatarStateRepo
	icRepo AvatarIconRepo

	// object storage (icons)
	objStore AvatarIconObjectStoragePort

	// wallet
	walletSvc  AvatarWalletService
	walletRepo WalletRepo // ✅ required for Create

	// cart
	cartRepo CartRepo

	now func() time.Time
}

func NewAvatarUsecase(
	avRepo AvatarRepo,
	stRepo AvatarStateRepo,
	icRepo AvatarIconRepo,
	objStore AvatarIconObjectStoragePort,
) *AvatarUsecase {
	return &AvatarUsecase{
		avRepo:   avRepo,
		stRepo:   stRepo,
		icRepo:   icRepo,
		objStore: objStore,
		now:      time.Now,
	}
}

func (u *AvatarUsecase) WithNow(now func() time.Time) *AvatarUsecase {
	u.now = now
	return u
}

// WithWalletService injects wallet opener.
func (u *AvatarUsecase) WithWalletService(svc AvatarWalletService) *AvatarUsecase {
	u.walletSvc = svc
	return u
}

// ✅ WithWalletRepo injects wallet persistence.
func (u *AvatarUsecase) WithWalletRepo(r WalletRepo) *AvatarUsecase {
	u.walletRepo = r
	return u
}

// ✅ WithCartRepo injects cart persistence.
func (u *AvatarUsecase) WithCartRepo(r CartRepo) *AvatarUsecase {
	u.cartRepo = r
	return u
}
