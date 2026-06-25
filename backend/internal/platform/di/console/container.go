// backend/internal/platform/di/console/container.go
package console

import (
	"context"
	"errors"

	query "narratives/internal/application/query/console"

	inspectorquery "narratives/internal/application/query/inspector"
	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"

	uc "narratives/internal/application/usecase"

	shared "narratives/internal/platform/di/shared"

	avatar "narratives/internal/domain/avatar"
	avatarstate "narratives/internal/domain/avatarState"
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

	MemberRepo  memdom.Repository
	BrandRepo   branddom.Repository
	CompanyRepo companydom.Repository

	TokenBlueprintRepo         tokenBlueprint.RepositoryPort
	TokenBlueprintReviewRepo   tbReview.RepositoryPort
	ProductBlueprintRepo       pbdomain.Repository
	ProductBlueprintReviewRepo pbReview.Repository

	AvatarRepo      avatar.Repository
	AvatarStateRepo avatarstate.Repository

	MemberService *memdom.Service

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

	TokenBlueprintUC *uc.TokenBlueprintUsecase

	UserUC   *uc.UserUsecase
	WalletUC *uc.WalletUsecase

	CartUC *uc.CartUsecase

	CompanyProductionQueryService *query.CompanyProductionQueryService
	MintRequestQueryService       *query.MintRequestQueryService

	BrandManagementQuery *query.BrandManagementQuery
	BrandDetailQuery     *query.BrandDetailQuery

	ProductBlueprintManagementQuery *query.ProductBlueprintManagementQuery
	ProductBlueprintDetailQuery     *query.ProductBlueprintDetailQuery

	TokenBlueprintManagementQuery *query.TokenBlueprintManagementQuery
	TokenBlueprintDetailQuery     *query.TokenBlueprintDetailQuery

	InquiryManagementQuery *query.InquiryManagementQuery
	InquiryDetailQuery     *query.InquiryDetailQuery

	InventoryManagementQuery *query.InventoryManagementQuery
	InventoryDetailQuery     *query.InventoryDetailQuery

	ListCreateQuery             *query.ListCreateQuery
	SalesQuery                  *query.SalesQuery
	AnnouncementManagementQuery *query.AnnouncementManagementQuery
	AnnouncementDetailQuery     *query.AnnouncementDetailQuery

	PrintQueryService *query.PrintQueryService

	ListManagementQuery *query.ListManagementQuery
	ListDetailQuery     *query.ListDetailQuery

	OrderManagementQuery *query.OrderManagementQuery
	OrderDetailQuery     *query.OrderDetailQuery

	InspectorQuery *inspectorquery.QueryService

	InventoryBlueprintResolver query.InventoryBlueprintResolver

	OwnerResolveQ *sharedquery.OwnerResolveQuery

	InspectionUC *uc.InspectionUsecase
	MintUC       *uc.MintUsecase

	InvitationUC  uc.InvitationUsecasePort
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
	res := buildResolvers(clients, repos)
	u := buildUsecases(clients, repos, services, res)
	q := buildQueries(clients.infra, repos, res, u, services)

	if clients == nil || clients.infra == nil {
		return nil, errors.New("clients/infra is nil")
	}

	var invBlueprint query.InventoryBlueprintResolver
	if repos.inventoryRepo != nil {
		invBlueprint = repos.inventoryRepo
	}

	var orderMgmtQ *query.OrderManagementQuery
	if repos.orderConsoleLister != nil && q.inventoryManagementQuery != nil && invBlueprint != nil {
		orderMgmtQ = query.NewOrderManagementQuery(query.NewOrderManagementQueryParams{
			Lister:       repos.orderConsoleLister,
			InvRows:      q.inventoryManagementQuery,
			InvBlueprint: invBlueprint,

			PBName:           repos.productBlueprintRepo,
			ProductBlueprint: repos.productBlueprintRepo,
			TBName:           repos.tokenBlueprintRepo,
			AvatarName:       repos.avatarRepo,

			ListReadable:  repos.listRepoFS,
			ModelResolver: res.nameResolver,
		})
	}

	announcementManagementQuery := query.NewAnnouncementManagementQuery(
		repos.tokenBlueprintRepo,
		repos.announcementRepo,
	)

	announcementDetailQuery := query.NewAnnouncementDetailQuery(
		repos.announcementRepo,
		repos.announcementAttachmentRepo,
		repos.memberRepo,
		repos.avatarRepo,
		repos.avatarStateRepo,
		repos.tokenReaderRepo,
		res.mintProductBlueprintResolver,
	)

	return &Container{
		Infra: clients.infra,

		MemberRepo:  repos.memberRepo,
		BrandRepo:   repos.brandRepo,
		CompanyRepo: repos.companyRepo,

		TokenBlueprintRepo:         repos.tokenBlueprintRepo,
		TokenBlueprintReviewRepo:   repos.tokenBlueprintReviewRepo,
		ProductBlueprintRepo:       repos.productBlueprintRepo,
		ProductBlueprintReviewRepo: repos.productBlueprintReviewRepo,

		AvatarRepo:      repos.avatarRepo,
		AvatarStateRepo: repos.avatarStateRepo,

		MemberService: services.memberSvc,

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

		TokenBlueprintUC: u.tokenBlueprintUC,

		UserUC:   u.userUC,
		WalletUC: u.walletUC,

		CartUC: u.cartUC,

		CompanyProductionQueryService: q.companyProductionQueryService,
		MintRequestQueryService:       q.mintRequestQueryService,

		BrandManagementQuery: q.brandManagementQuery,
		BrandDetailQuery:     q.brandDetailQuery,

		ProductBlueprintManagementQuery: q.productBlueprintManagementQuery,
		ProductBlueprintDetailQuery:     q.productBlueprintDetailQuery,

		TokenBlueprintManagementQuery: q.tokenBlueprintManagementQuery,
		TokenBlueprintDetailQuery:     q.tokenBlueprintDetailQuery,

		InquiryManagementQuery: q.inquiryManagementQuery,
		InquiryDetailQuery:     q.inquiryDetailQuery,

		InventoryManagementQuery: q.inventoryManagementQuery,
		InventoryDetailQuery:     q.inventoryDetailQuery,

		ListCreateQuery: q.listCreateQuery,
		SalesQuery:      q.salesQuery,

		AnnouncementManagementQuery: announcementManagementQuery,
		AnnouncementDetailQuery:     announcementDetailQuery,

		PrintQueryService: q.printQueryService,

		ListManagementQuery: q.listManagementQuery,
		ListDetailQuery:     q.listDetailQuery,

		OrderManagementQuery: orderMgmtQ,
		OrderDetailQuery:     q.orderDetailQuery,

		InspectorQuery: q.inspectorQuery,

		InventoryBlueprintResolver: invBlueprint,

		OwnerResolveQ: res.ownerResolveQuery,

		InspectionUC: u.inspectionUC,
		MintUC:       u.mintUC,

		InvitationUC: u.invitationUC,

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
