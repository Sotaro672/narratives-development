package console

import (
	"context"
	"errors"

	consoleHandler "narratives/internal/adapters/in/http/console/handler"

	query "narratives/internal/application/query/console"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"

	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"

	uc "narratives/internal/application/usecase"

	shared "narratives/internal/platform/di/shared"

	avatar "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	pbdomain "narratives/internal/domain/productBlueprint"
	pbReview "narratives/internal/domain/productBlueprintReview"
	tokenBlueprint "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

type Container struct {
	Infra *shared.Infra

	MemberRepo memdom.Repository

	TokenBlueprintRepo         tokenBlueprint.RepositoryPort
	TokenBlueprintReviewRepo   tbReview.RepositoryPort
	ProductBlueprintRepo       pbdomain.Repository
	ProductBlueprintReviewRepo pbReview.Repository

	AvatarRepo avatar.Repository

	MemberService  *memdom.Service
	CompanyService *companydom.Service
	BrandService   *branddom.Service

	AccountUC                  *uc.AccountUsecase
	AnnouncementUC             *uc.AnnouncementUsecase
	AvatarUC                   *uc.AvatarUsecase
	PaymentMethodUC            *uc.PaymentMethodUsecase
	BrandUC                    *uc.BrandUsecase
	CompanyUC                  *uc.CompanyUsecase
	InquiryUC                  *uc.InquiryUsecase
	InventoryUC                *uc.InventoryUsecase
	ListUC                     *uc.ListUsecase
	MemberUC                   *uc.MemberUsecase
	ModelUC                    *uc.ModelUsecase
	OrderUC                    *uc.OrderUsecase
	PaymentUC                  *uc.PaymentUsecase
	PermissionUC               *uc.PermissionUsecase
	PrintUC                    *uc.PrintUsecase
	ProductionUC               *uc.ProductionUsecase
	ProductBlueprintUC         *uc.ProductBlueprintUsecase
	ProductBlueprintCategoryUC *uc.ProductBlueprintCategoryUsecase
	ShippingAddressUC          *uc.ShippingAddressUsecase
	TokenUC                    *uc.TokenUsecase

	TokenBlueprintUC      *uc.TokenBlueprintUsecase
	TokenBlueprintQueryUC *uc.TokenBlueprintQueryUsecase

	UserUC   *uc.UserUsecase
	WalletUC *uc.WalletUsecase

	CartUC *uc.CartUsecase

	CompanyProductionQueryService *query.CompanyProductionQueryService
	MintRequestQueryService       *query.MintRequestQueryService
	InventoryQuery                *query.InventoryQuery
	ListCreateQuery               *query.ListCreateQuery
	SalesQuery                    *query.SalesQuery

	ListManagementQuery *listmanagementquery.ListManagementQuery
	ListDetailQuery     *listdetailquery.ListDetailQuery

	OrderManagementQuery *query.OrderManagementQuery

	InventoryBlueprintResolver consoleHandler.InventoryBlueprintResolver

	OwnerResolveQ *sharedquery.OwnerResolveQuery

	ProductUC    *uc.ProductUsecase
	InspectionUC *uc.InspectionUsecase
	MintUC       *uc.MintUsecase

	InvitationQuery    uc.InvitationQueryPort
	InvitationCommand  uc.InvitationCommandPort
	InvitationComplete uc.InvitationCompletePort

	AuthBootstrap *uc.BootstrapService
	NameResolver  *resolver.NameResolver
}

func NewContainer(ctx context.Context, infra *shared.Infra) (*Container, error) {
	clients, err := ensureClients(ctx, infra)
	if err != nil {
		return nil, err
	}

	repos := buildRepos(clients)
	services := buildDomainServices(repos)
	res := buildResolvers(clients, repos, services)
	u := buildUsecases(clients, repos, services, res)
	q := buildQueries(clients.infra, repos, res, u)

	if clients == nil || clients.infra == nil {
		return nil, errors.New("clients/infra is nil")
	}

	var invBlueprint consoleHandler.InventoryBlueprintResolver
	if repos.inventoryRepo != nil {
		invBlueprint = repos.inventoryRepo
	}

	var orderMgmtQ *query.OrderManagementQuery
	if repos.orderRepo != nil && q.inventoryQuery != nil && invBlueprint != nil {
		orderMgmtQ = query.NewOrderManagementQuery(query.NewOrderManagementQueryParams{
			Lister:       repos.orderRepo,
			InvRows:      q.inventoryQuery,
			InvBlueprint: invBlueprint,

			PBName:           repos.productBlueprintRepo,
			ProductBlueprint: repos.productBlueprintRepo,
			TBName:           repos.tokenBlueprintRepo,
			AvatarName:       repos.avatarRepo,

			ListReadable:  repos.listRepoFS,
			ModelResolver: res.nameResolver,
		})
	}

	return &Container{
		Infra: clients.infra,

		MemberRepo: repos.memberRepo,

		TokenBlueprintRepo:         repos.tokenBlueprintRepo,
		TokenBlueprintReviewRepo:   repos.tokenBlueprintReviewRepo,
		ProductBlueprintRepo:       repos.productBlueprintRepo,
		ProductBlueprintReviewRepo: repos.productBlueprintReviewRepo,

		AvatarRepo: repos.avatarRepo,

		MemberService:  services.memberSvc,
		CompanyService: services.companySvc,
		BrandService:   services.brandSvc,

		AccountUC:                  u.accountUC,
		AnnouncementUC:             u.announcementUC,
		AvatarUC:                   u.avatarUC,
		PaymentMethodUC:            u.paymentMethodUC,
		BrandUC:                    u.brandUC,
		CompanyUC:                  u.companyUC,
		InquiryUC:                  u.inquiryUC,
		InventoryUC:                u.inventoryUC,
		ListUC:                     u.listUC,
		MemberUC:                   u.memberUC,
		ModelUC:                    u.modelUC,
		OrderUC:                    u.orderUC,
		PaymentUC:                  u.paymentUC,
		PermissionUC:               u.permissionUC,
		PrintUC:                    u.printUC,
		ProductionUC:               u.productionUC,
		ProductBlueprintUC:         u.productBlueprintUC,
		ProductBlueprintCategoryUC: u.productBlueprintCategoryUC,
		ShippingAddressUC:          u.shippingAddressUC,
		TokenUC:                    u.tokenUC,

		TokenBlueprintUC:      u.tokenBlueprintUC,
		TokenBlueprintQueryUC: u.tokenBlueprintQueryUC,

		UserUC:   u.userUC,
		WalletUC: u.walletUC,

		CartUC: u.cartUC,

		CompanyProductionQueryService: q.companyProductionQueryService,
		MintRequestQueryService:       q.mintRequestQueryService,
		InventoryQuery:                q.inventoryQuery,
		ListCreateQuery:               q.listCreateQuery,
		SalesQuery:                    q.salesQuery,

		ListManagementQuery: q.listManagementQuery,
		ListDetailQuery:     q.listDetailQuery,

		OrderManagementQuery:       orderMgmtQ,
		InventoryBlueprintResolver: invBlueprint,

		OwnerResolveQ: res.ownerResolveQuery,

		ProductUC:    u.productUC,
		InspectionUC: u.inspectionUC,
		MintUC:       u.mintUC,

		InvitationQuery:    u.invitationQueryUC,
		InvitationCommand:  u.invitationCommandUC,
		InvitationComplete: u.invitationCompleteUC,

		AuthBootstrap: u.authBootstrapSvc,

		NameResolver: res.nameResolver,
	}, nil
}

func (c *Container) Close() error {
	if c != nil && c.Infra != nil {
		return c.Infra.Close()
	}
	return nil
}
