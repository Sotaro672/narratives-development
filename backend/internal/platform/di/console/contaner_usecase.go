// backend/internal/platform/di/console/contaner_usecase.go
package console

import (
	"context"
	"errors"
	"os"
	"strings"

	fsrepo "narratives/internal/adapters/out/firestore"
	mailadp "narratives/internal/adapters/out/mail"
	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	pbuc "narratives/internal/application/productBlueprint/usecase"
	productionapp "narratives/internal/application/production"
	tokenblueprintapp "narratives/internal/application/tokenBlueprint"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	avataruc "narratives/internal/application/usecase/avatar"
	listuc "narratives/internal/application/usecase/list"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	"narratives/internal/infra/arweave"
	solanainfra "narratives/internal/infra/solana"
)

type usecases struct {
	tokenUC *uc.TokenUsecase

	accountUC       *uc.AccountUsecase
	announcementUC  *uc.AnnouncementUsecase
	avatarUC        *avataruc.AvatarUsecase
	paymentMethodUC *uc.PaymentMethodUsecase
	brandUC         *uc.BrandUsecase
	companyUC       *uc.CompanyUsecase
	inquiryUC       *uc.InquiryUsecase
	inventoryUC     *uc.InventoryUsecase
	listUC          *listuc.ListUsecase
	memberUC        *uc.MemberUsecase
	modelUC         *uc.ModelUsecase
	orderUC         *uc.OrderUsecase
	paymentUC       *uc.PaymentUsecase
	permissionUC    *uc.PermissionUsecase
	printUC         *uc.PrintUsecase

	productionUC       *productionapp.ProductionUsecase
	productBlueprintUC *pbuc.ProductBlueprintUsecase

	inspectionUC *inspectionapp.InspectionUsecase
	productUC    *uc.ProductUsecase
	mintUC       *mintapp.MintUsecase

	shippingAddressUC *uc.ShippingAddressUsecase

	tokenBlueprintUC      *tokenblueprintapp.TokenBlueprintUsecase
	tokenBlueprintQueryUC *tokenblueprintapp.TokenBlueprintQueryUsecase

	tokenBlueprintReviewUC *uc.TokenBlueprintReviewUsecase

	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase

	userUC   *uc.UserUsecase
	walletUC *uc.WalletUsecase
	cartUC   *uc.CartUsecase

	invitationQueryUC    uc.InvitationQueryPort
	invitationCommandUC  uc.InvitationCommandPort
	invitationCompleteUC uc.InvitationCompletePort

	authBootstrapSvc *authuc.BootstrapService
}

func buildUsecases(c *clients, r *repos, s *services, res *resolvers) *usecases {
	var tokenUC *uc.TokenUsecase
	if c.infra.MintAuthorityKey != nil {
		solanaClient := solanainfra.NewMintClient(c.infra.MintAuthorityKey)
		tokenUC = uc.NewTokenUsecase(solanaClient, r.mintRequestPort)
	} else {
		tokenUC = uc.NewTokenUsecase(nil, r.mintRequestPort)
	}

	accountUC := uc.NewAccountUsecase(r.accountRepo)
	announcementUC := uc.NewAnnouncementUsecase(r.announcementRepo, nil, nil)

	brandWalletSvc := solanainfra.NewBrandWalletService(c.firestoreProjectID)
	avatarWalletSvc := solanainfra.NewAvatarWalletService(c.firestoreProjectID)

	avatarUC := avataruc.NewAvatarUsecase(
		r.avatarRepo,
		r.avatarStateRepo,
	).
		WithWalletService(avatarWalletSvc).
		WithWalletRepo(r.walletRepo)

	callOptionalMethod(avatarUC, "WithCartRepo", r.cartRepo)

	paymentMethodUC := uc.NewPaymentMethodUsecase(
		r.paymentMethodRepo,
		c.infra.PaymentMethodGateway,
	)

	brandUC := uc.NewBrandUsecase(
		r.brandRepo,
		r.memberRepo,
		uc.WithBrandWalletService(brandWalletSvc),
	)

	companyUC := uc.NewCompanyUsecase(r.companyRepo)
	inquiryUC := uc.NewInquiryUsecase(r.inquiryRepo, nil, nil)

	inventoryUC := uc.NewInventoryUsecase(r.inventoryRepo)
	paymentUC := uc.NewPaymentUsecase(r.paymentRepo)

	var listReader listuc.ListReader = r.listRepoFS
	var listCreator listuc.ListCreator = r.listRepoFS

	var listPatcher listuc.ListPatcher
	if r.listRepoFS != nil {
		listPatcher = r.listRepoFS
	} else {
		listPatcher = nil
	}

	// Firebase Storage 移行後:
	// - frontend が Firebase Storage へ直接 upload する
	// - backend は GCS signed URL / GCS object / bucket を扱わない
	// - list image は Firestore record repository のみを usecase に渡す
	listUC := listuc.NewListUsecase(
		listReader,
		listCreator,
		listPatcher,
		r.listImageRecordRepo,
		r.listImageRecordRepo,
	)

	modelUC := uc.NewModelUsecase(r.modelRepo)

	orderUC := uc.NewOrderUsecase(r.orderRepo, r.cartRepo)

	permissionUC := uc.NewPermissionUsecase(r.permissionRepo)

	printUC := uc.NewPrintUsecase(
		r.productRepo,
		r.printLogRepo,
		r.inspectionRepo,
		res.nameResolver,
		r.productBlueprintRepo,
	)

	productionUC := productionapp.NewProductionUsecase(
		r.productionRepo,
		s.pbSvc,
		res.nameResolver,
	)

	productBlueprintUC := pbuc.NewProductBlueprintUsecase(
		r.productBlueprintRepo,
		r.productBlueprintReviewRepo,
	)

	inspectionUC := inspectionapp.NewInspectionUsecase(
		r.inspectionRepo,
		r.productRepo,
		r.mintRepo,
		r.modelRepo,
	)

	productUC := uc.NewProductUsecase(
		r.productRepo,
		r.modelRepo,
		r.productionRepo,
		r.productBlueprintRepo,
		s.brandSvc,
		s.companySvc,
	)

	mintUC := mintapp.NewMintUsecase(
		r.productBlueprintRepo,
		r.productionRepo,
		r.inspectionRepo,
		r.modelRepo,
		r.tokenBlueprintRepo,
		s.brandSvc,
		r.mintRepo,
		r.inspectionRepo,
		tokenUC,
	)
	mintUC.SetNameResolver(res.nameResolver)
	mintUC.SetInventoryUsecase(inventoryUC)

	// GCS bucket ensurer は廃止。
	// tokenBlueprint icon / contents は frontend が Firebase Storage へ直接 upload し、
	// backend は Firestore に保存された downloadURL / objectPath を扱う。

	baseURL := os.Getenv("ARWEAVE_BASE_URL")
	apiKey := os.Getenv("IRYS_SERVICE_API_KEY")
	uploader := arweave.NewHTTPUploader(baseURL, apiKey)

	tbMetadataUC := tokenblueprintapp.NewTokenBlueprintMetadataUsecase(r.tokenBlueprintRepo, uploader)
	mintUC.SetTokenBlueprintMetadataEnsurer(tbMetadataUC)

	shippingAddressUC := uc.NewShippingAddressUsecase(r.shippingAddressRepo)

	tbReviewRepo := fsrepo.NewTokenBlueprintReviewRepositoryFS(c.fsClient)

	tokenBlueprintUC := tokenblueprintapp.NewTokenBlueprintUsecase(
		r.tokenBlueprintRepo,
		tbReviewRepo,
		res.nameResolver,
	)

	tokenBlueprintQueryUC := tokenblueprintapp.NewTokenBlueprintQueryUsecase(
		r.tokenBlueprintRepo,
		res.nameResolver,
	)

	tokenBlueprintReviewUC := uc.NewTokenBlueprintReviewUsecase(
		tbReviewRepo,
		r.avatarRepo,
		r.tokenBlueprintRepo,
		s.brandSvc,
	)

	var productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase
	if r.productBlueprintReviewRepo != nil && r.productBlueprintRepo != nil && r.walletRepo != nil {
		memberSvc := memdom.NewService(r.memberRepo)

		productBlueprintReviewUC = uc.NewProductBlueprintReviewUsecase(
			r.productBlueprintReviewRepo,
			r.walletRepo,
		).
			WithProductBlueprintRepo(r.productBlueprintRepo).
			WithBrandService(s.brandSvc).
			WithMemberService(memberSvc)
	}

	userUC := uc.NewUserUsecase(r.userRepo)
	walletUC := uc.NewWalletUsecase(r.walletRepo)
	cartUC := uc.NewCartUsecase(r.cartRepo)

	invitationMailer := mailadp.NewInvitationMailerWithResend(s.companySvc, s.brandSvc)

	var invitationTokenRepo uc.InvitationTokenRepository
	if repo, ok := r.invitationTokenUCRepo.(uc.InvitationTokenRepository); ok {
		invitationTokenRepo = repo
	} else {
		invitationTokenRepo = &invitationTokenRepositoryAdapter{
			repo: r.invitationTokenUCRepo,
		}
	}

	invitationQueryUC := uc.NewInvitationService(
		invitationTokenRepo,
		r.memberRepo,
	)

	invitationCommandUC := uc.NewInvitationCommandService(
		invitationTokenRepo,
		r.memberRepo,
		invitationMailer,
	)

	invitationCompleteUC := uc.NewInvitationCompleteService(
		invitationTokenRepo,
		r.memberRepo,
	)

	memberUC := uc.NewMemberUsecaseWithInvitationCommand(
		r.memberRepo,
		invitationCommandUC,
	)

	authBootstrapSvc := &authuc.BootstrapService{
		Members: &authMemberRepoAdapter{repo: r.memberRepo},
		Companies: &authCompanyRepoAdapter{
			repo: r.companyRepo,
		},
	}

	return &usecases{
		tokenUC: tokenUC,

		accountUC:       accountUC,
		announcementUC:  announcementUC,
		avatarUC:        avatarUC,
		paymentMethodUC: paymentMethodUC,
		brandUC:         brandUC,
		companyUC:       companyUC,
		inquiryUC:       inquiryUC,
		inventoryUC:     inventoryUC,
		listUC:          listUC,
		memberUC:        memberUC,
		modelUC:         modelUC,
		orderUC:         orderUC,
		paymentUC:       paymentUC,
		permissionUC:    permissionUC,
		printUC:         printUC,

		productionUC:       productionUC,
		productBlueprintUC: productBlueprintUC,

		inspectionUC: inspectionUC,
		productUC:    productUC,
		mintUC:       mintUC,

		shippingAddressUC: shippingAddressUC,

		tokenBlueprintUC:         tokenBlueprintUC,
		tokenBlueprintQueryUC:    tokenBlueprintQueryUC,
		tokenBlueprintReviewUC:   tokenBlueprintReviewUC,
		productBlueprintReviewUC: productBlueprintReviewUC,

		userUC:   userUC,
		walletUC: walletUC,
		cartUC:   cartUC,

		invitationQueryUC:    invitationQueryUC,
		invitationCommandUC:  invitationCommandUC,
		invitationCompleteUC: invitationCompleteUC,

		authBootstrapSvc: authBootstrapSvc,
	}
}

// ========================================
// invitation token repository adapter
// ========================================

type invitationTokenRepositoryAdapter struct {
	repo memdom.InvitationTokenRepository
}

func (a *invitationTokenRepositoryAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	if a == nil || a.repo == nil {
		return memdom.InvitationInfo{}, errors.New("invitationTokenRepositoryAdapter.ResolveInvitationInfoByToken: repo is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.InvitationInfo{}, memdom.ErrInvitationTokenNotFound
	}

	return a.repo.ResolveInvitationInfoByToken(ctx, token)
}

func (a *invitationTokenRepositoryAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	if a == nil || a.repo == nil {
		return "", errors.New("invitationTokenRepositoryAdapter.CreateInvitationToken: repo is nil")
	}

	info.MemberID = strings.TrimSpace(info.MemberID)
	info.CompanyID = strings.TrimSpace(info.CompanyID)
	info.Email = strings.TrimSpace(info.Email)

	return a.repo.CreateInvitationToken(ctx, info)
}

func (a *invitationTokenRepositoryAdapter) ConsumeInvitationToken(
	ctx context.Context,
	token string,
) error {
	if a == nil || a.repo == nil {
		return errors.New("invitationTokenRepositoryAdapter.ConsumeInvitationToken: repo is nil")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return memdom.ErrInvitationTokenNotFound
	}

	return a.repo.ConsumeInvitationToken(ctx, token)
}

// ========================================
// auth.BootstrapService 用アダプタ
// adapter_auth.go を廃止する前提でこのファイルに集約
// ========================================

// memdom.Repository -> auth.MemberRepository
type authMemberRepoAdapter struct {
	repo memdom.Repository
}

func (a *authMemberRepoAdapter) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*memdom.Member, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("authMemberRepoAdapter.GetByFirebaseUID: repo is nil")
	}

	firebaseUID = strings.TrimSpace(firebaseUID)
	if firebaseUID == "" {
		return nil, memdom.ErrNotFound
	}

	v, err := a.repo.GetByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (a *authMemberRepoAdapter) Create(ctx context.Context, m *memdom.Member) error {
	if a == nil || a.repo == nil {
		return errors.New("authMemberRepoAdapter.Create: repo is nil")
	}
	if m == nil {
		return errors.New("authMemberRepoAdapter.Create: nil member")
	}

	m.UID = strings.TrimSpace(m.UID)
	m.Email = strings.TrimSpace(m.Email)
	m.FirstName = strings.TrimSpace(m.FirstName)
	m.LastName = strings.TrimSpace(m.LastName)
	m.FirstNameKana = strings.TrimSpace(m.FirstNameKana)
	m.LastNameKana = strings.TrimSpace(m.LastNameKana)
	m.CompanyID = strings.TrimSpace(m.CompanyID)
	m.Status = strings.TrimSpace(m.Status)

	saved, err := a.repo.Create(ctx, *m)
	if err != nil {
		return err
	}

	*m = saved
	return nil
}

// CompanyRepositoryFS -> auth.CompanyRepository
type authCompanyRepoAdapter struct {
	repo *fsrepo.CompanyRepositoryFS
}

func (a *authCompanyRepoAdapter) NewID(ctx context.Context) (string, error) {
	if a == nil || a.repo == nil || a.repo.Client == nil {
		return "", errors.New("authCompanyRepoAdapter.NewID: repo or client is nil")
	}

	doc := a.repo.Client.Collection("companies").NewDoc()
	return doc.ID, nil
}

func (a *authCompanyRepoAdapter) Save(ctx context.Context, c *companydom.Company) error {
	if a == nil || a.repo == nil {
		return errors.New("authCompanyRepoAdapter.Save: repo is nil")
	}
	if c == nil {
		return errors.New("authCompanyRepoAdapter.Save: nil company")
	}

	saved, err := a.repo.Save(ctx, *c, nil)
	if err != nil {
		return err
	}

	*c = saved
	return nil
}
