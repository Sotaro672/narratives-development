// backend/internal/platform/di/console/container.go
package console

import (
	"context"
	"errors"
	"log"

	consoleHandler "narratives/internal/adapters/in/http/console/handler"

	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	pbuc "narratives/internal/application/productBlueprint/usecase"
	productionapp "narratives/internal/application/production"

	query "narratives/internal/application/query/console"

	listdetailquery "narratives/internal/application/query/console/list/detail"
	listmanagementquery "narratives/internal/application/query/console/list/management"

	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"
	tokenblueprintapp "narratives/internal/application/tokenBlueprint"

	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"
	avatarUC "narratives/internal/application/usecase/avatar"

	listuc "narratives/internal/application/usecase/list"

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

// ========================================
// Container (Console DI)
// ========================================
type Container struct {
	Infra *shared.Infra

	// Repositories (AuthMiddleware 用に memberRepo だけ保持)
	MemberRepo memdom.Repository

	// review handler wiring に必要な repo を Container が保持
	TokenBlueprintRepo         tokenBlueprint.RepositoryPort
	TokenBlueprintReviewRepo   tbReview.RepositoryPort
	ProductBlueprintRepo       pbdomain.Repository
	ProductBlueprintReviewRepo pbReview.Repository

	// TokenBlueprintReviewUsecase に渡すため
	AvatarRepo avatar.Repository

	// member.Service / company.Service / brand.Service (表示名解決用)
	MemberService  *memdom.Service
	CompanyService *companydom.Service
	BrandService   *branddom.Service

	// Application-layer usecases
	AccountUC                  *uc.AccountUsecase
	AnnouncementUC             *uc.AnnouncementUsecase
	AvatarUC                   *avatarUC.AvatarUsecase
	PaymentMethodUC            *uc.PaymentMethodUsecase
	BrandUC                    *uc.BrandUsecase
	CompanyUC                  *uc.CompanyUsecase
	InquiryUC                  *uc.InquiryUsecase
	InventoryUC                *uc.InventoryUsecase
	ListUC                     *listuc.ListUsecase
	MemberUC                   *uc.MemberUsecase
	ModelUC                    *uc.ModelUsecase
	OrderUC                    *uc.OrderUsecase
	PaymentUC                  *uc.PaymentUsecase
	PermissionUC               *uc.PermissionUsecase
	PrintUC                    *uc.PrintUsecase
	ProductionUC               *productionapp.ProductionUsecase
	ProductBlueprintUC         *pbuc.ProductBlueprintUsecase
	ProductBlueprintCategoryUC *uc.ProductBlueprintCategoryUsecase
	ShippingAddressUC          *uc.ShippingAddressUsecase
	TokenUC                    *uc.TokenUsecase

	TokenBlueprintUC      *tokenblueprintapp.TokenBlueprintUsecase
	TokenBlueprintQueryUC *tokenblueprintapp.TokenBlueprintQueryUsecase

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

	// order management query instance (for console order endpoints)
	OrderManagementQuery *query.OrderManagementQuery

	// /orders/{id} で item に productBlueprintId/tokenBlueprintId を載せるための resolver
	InventoryBlueprintResolver consoleHandler.InventoryBlueprintResolver

	OwnerResolveQ *sharedquery.OwnerResolveQuery

	ProductUC    *uc.ProductUsecase
	InspectionUC *inspectionapp.InspectionUsecase
	MintUC       *mintapp.MintUsecase

	InvitationQuery    uc.InvitationQueryPort
	InvitationCommand  uc.InvitationCommandPort
	InvitationComplete uc.InvitationCompletePort

	AuthBootstrap *authuc.BootstrapService
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

	log.Printf("[di.console] productBlueprint review initializer skipped (initializer not implemented)")

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

func init() {
	log.Printf("[di.console] container package loaded")
}
