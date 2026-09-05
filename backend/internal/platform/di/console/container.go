// backend/internal/platform/di/console/container.go
package console

import (
	"context"
	"errors"

	listcloudtasksadp "narratives/internal/adapters/out/cloudtasks"
	query "narratives/internal/application/query/console"
	inspectorquery "narratives/internal/application/query/inspector"
	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"
	uc "narratives/internal/application/usecase"
	avatar "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	pbdomain "narratives/internal/domain/productBlueprint"
	pbReview "narratives/internal/domain/productBlueprintReview"
	tokenBlueprint "narratives/internal/domain/tokenBlueprint"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
	shared "narratives/internal/platform/di/shared"
)

type Container struct {
	Infra                           *shared.Infra
	resources                       *containerResources
	MemberRepo                      memdom.Repository
	BrandRepo                       branddom.Repository
	CompanyRepo                     companydom.Repository
	TokenBlueprintRepo              tokenBlueprint.RepositoryPort
	TokenBlueprintReviewRepo        tbReview.RepositoryPort
	ProductBlueprintRepo            pbdomain.Repository
	ProductBlueprintReviewRepo      pbReview.Repository
	AvatarRepo                      avatar.Repository
	MemberService                   *memdom.Service
	AccountUC                       *uc.AccountUsecase
	AnnouncementUC                  *uc.AnnouncementUsecase
	AvatarUC                        *uc.AvatarUsecase
	PaymentMethodUC                 *uc.PaymentMethodUsecase
	BrandUC                         *uc.BrandUsecase
	CompanyUC                       *uc.CompanyUsecase
	CompanyQuery                    *query.CompanyQuery
	InquiryUC                       *uc.InquiryUsecase
	ReturnReceiptUC                 *uc.ReturnReceiptUsecase
	OpenedReturnReceiptUC           *uc.OpenedReturnReceiptUsecase
	InventoryUC                     *uc.InventoryUsecase
	ListUC                          *uc.ListUsecase
	ListSaveOperationUC             *uc.ListSaveOperationUsecase
	MemberUC                        *uc.MemberUsecase
	ModelUC                         *uc.ModelUsecase
	OrderUC                         *uc.OrderUsecase
	OrderDispatchNotificationUC     uc.OrderDispatchNotificationUsecasePort
	RefundCompletionNotificationUC  uc.RefundCompletionNotificationUsecasePort
	PaymentUC                       *uc.PaymentUsecase
	PaymentFlowUC                   *uc.PaymentFlowUsecase
	SettlementUC                    *uc.SettlementUsecase
	SettlementQueue                 uc.SettlementTransferQueue
	RefundUC                        *uc.RefundUsecase
	PermissionUC                    *uc.PermissionUsecase
	PrintUC                         *uc.PrintUsecase
	ProductionUC                    *uc.ProductionUsecase
	ProductBlueprintUC              *uc.ProductBlueprintUsecase
	ProductBlueprintCategoryUC      *uc.ProductBlueprintCategoryUsecase
	ProductBlueprintReviewUC        *uc.ProductBlueprintReviewUsecase
	ReviewReportUC                  *uc.ReviewReportUsecase
	ShippingAddressUC               *uc.ShippingAddressUsecase
	TransportationUC                *uc.TransportationUsecase
	TokenUC                         *uc.TokenUsecase
	TokenBlueprintUC                *uc.TokenBlueprintUsecase
	TokenBlueprintCreateOperationUC *uc.TokenBlueprintCreateOperationUsecase
	TokenBlueprintReviewUC          *uc.TokenBlueprintReviewUsecase
	UserUC                          *uc.UserUsecase
	WalletUC                        *uc.WalletUsecase
	CartUC                          *uc.CartUsecase
	CompanyProductionQueryService   *query.CompanyProductionQueryService
	MintRequestQueryService         *query.MintRequestQueryService
	MintFundingEstimateQuery        *query.MintFundingEstimateQuery
	BrandManagementQuery            *query.BrandManagementQuery
	BrandDetailQuery                *query.BrandDetailQuery
	ProductBlueprintManagementQuery *query.ProductBlueprintManagementQuery
	ProductBlueprintDetailQuery     *query.ProductBlueprintDetailQuery
	TokenBlueprintManagementQuery   *query.TokenBlueprintManagementQuery
	TokenBlueprintDetailQuery       *query.TokenBlueprintDetailQuery
	LocationManagementQuery         *query.LocationManagementQuery
	LocationDetailQuery             *query.LocationDetailQuery
	TransportationManagementQuery   *query.TransportationManagementQuery
	TransportationDetailQuery       *query.TransportationDetailQuery
	InquiryManagementQuery          *query.InquiryManagementQuery
	InquiryDetailQuery              *query.InquiryDetailQuery
	InventoryManagementQuery        *query.InventoryManagementQuery
	InventoryDetailQuery            *query.InventoryDetailQuery
	ListCreateQuery                 *query.ListCreateQuery
	SalesQuery                      *query.SalesQuery
	AnnouncementManagementQuery     *query.AnnouncementManagementQuery
	AnnouncementDetailQuery         *query.AnnouncementDetailQuery
	PrintQueryService               *query.PrintQueryService
	ListManagementQuery             *query.ListManagementQuery
	ListDetailQuery                 *query.ListDetailQuery
	OrderManagementQuery            *query.OrderManagementQuery
	OrderDetailQuery                *query.OrderDetailQuery
	TransactionManagementQuery      *query.TransactionManagementQuery
	InspectorQuery                  *inspectorquery.QueryService
	InventoryBlueprintResolver      query.InventoryBlueprintResolver
	OwnerResolveQ                   *sharedquery.OwnerResolveQuery
	InspectionUC                    *uc.InspectionUsecase
	MintUC                          *uc.MintUsecase
	InvitationUC                    uc.InvitationUsecasePort
	InvitationDeliveryUC            uc.InvitationDeliveryUsecasePort
	AuthBootstrap                   *uc.BootstrapService
	NameResolver                    *resolver.NameResolver
}

func NewContainer(
	ctx context.Context,
	infra *shared.Infra,
) (*Container, error) {
	clients, err := ensureClients(ctx, infra)
	if err != nil {
		return nil, err
	}

	if clients == nil || clients.infra == nil {
		return nil, errors.New("di.console: clients/infra is nil")
	}

	repos := buildRepos(clients)
	services := buildDomainServices(repos)
	res := buildResolvers(clients, repos)

	u, err := buildUsecases(
		ctx,
		clients,
		repos,
		services,
		res,
	)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("di.console: usecases is nil")
	}
	if u.resources == nil {
		return nil, errors.New("di.console: container resources is nil")
	}

	resources := u.resources

	settlementQueue, err := listcloudtasksadp.NewSettlementQueueFromEnv(ctx)
	if err != nil {
		return nil, resources.CloseWithError(err)
	}
	if settlementQueue == nil {
		return nil, resources.CloseWithError(
			errors.New("di.console: settlement queue is nil"),
		)
	}

	resources.Add("settlement queue", settlementQueue)

	q := buildQueries(
		clients.infra,
		repos,
		res,
		u,
		services,
	)

	var invBlueprint query.InventoryBlueprintResolver
	if repos.inventoryRepo != nil {
		invBlueprint = repos.inventoryRepo
	}

	var orderMgmtQ *query.OrderManagementQuery
	if repos.orderConsoleLister != nil &&
		q.inventoryManagementQuery != nil &&
		invBlueprint != nil {
		orderMgmtQ = query.NewOrderManagementQuery(
			query.NewOrderManagementQueryParams{
				Lister:           repos.orderConsoleLister,
				InvRows:          q.inventoryManagementQuery,
				InvBlueprint:     invBlueprint,
				ProductBlueprint: repos.productBlueprintRepo,
				TBName:           repos.tokenBlueprintRepo,
				UserName:         res.nameResolver,
				ListReadable:     repos.listRepoFS,
				ModelResolver:    res.nameResolver,
			},
		)
	}

	announcementManagementQuery := query.NewAnnouncementManagementQuery(
		repos.tokenBlueprintRepo,
		repos.announcementRepo,
	)

	announcementDetailQuery := query.NewAnnouncementDetailQuery(
		repos.announcementRepo,
		repos.announcementAttachmentRepo,
		repos.memberRepo,
	)

	return &Container{
		Infra:                           clients.infra,
		resources:                       resources,
		MemberRepo:                      repos.memberRepo,
		BrandRepo:                       repos.brandRepo,
		CompanyRepo:                     repos.companyRepo,
		TokenBlueprintRepo:              repos.tokenBlueprintRepo,
		TokenBlueprintReviewRepo:        repos.tokenBlueprintReviewRepo,
		ProductBlueprintRepo:            repos.productBlueprintRepo,
		ProductBlueprintReviewRepo:      repos.productBlueprintReviewRepo,
		AvatarRepo:                      repos.avatarRepo,
		MemberService:                   services.memberSvc,
		AccountUC:                       u.accountUC,
		AnnouncementUC:                  u.announcementUC,
		AvatarUC:                        u.avatarUC,
		PaymentMethodUC:                 u.paymentMethodUC,
		BrandUC:                         u.brandUC,
		CompanyUC:                       u.companyUC,
		CompanyQuery:                    q.companyQuery,
		InquiryUC:                       u.inquiryUC,
		ReturnReceiptUC:                 u.returnReceiptUC,
		OpenedReturnReceiptUC:           u.openedReturnReceiptUC,
		InventoryUC:                     u.inventoryUC,
		ListUC:                          u.listUC,
		ListSaveOperationUC:             u.listSaveOperationUC,
		MemberUC:                        u.memberUC,
		ModelUC:                         u.modelUC,
		OrderUC:                         u.orderUC,
		OrderDispatchNotificationUC:     u.orderDispatchNotificationUC,
		RefundCompletionNotificationUC:  u.refundCompletionNotificationUC,
		PaymentUC:                       u.paymentUC,
		PaymentFlowUC:                   u.paymentFlowUC,
		SettlementUC:                    u.settlementUC,
		SettlementQueue:                 settlementQueue,
		RefundUC:                        u.refundUC,
		PermissionUC:                    u.permissionUC,
		PrintUC:                         u.printUC,
		ProductionUC:                    u.productionUC,
		ProductBlueprintUC:              u.productBlueprintUC,
		ProductBlueprintCategoryUC:      u.productBlueprintCategoryUC,
		ProductBlueprintReviewUC:        u.productBlueprintReviewUC,
		ReviewReportUC:                  u.reviewReportUC,
		ShippingAddressUC:               u.shippingAddressUC,
		TransportationUC:                u.transportationUC,
		TokenUC:                         u.tokenUC,
		TokenBlueprintUC:                u.tokenBlueprintUC,
		TokenBlueprintCreateOperationUC: u.tokenBlueprintCreateOperationUC,
		TokenBlueprintReviewUC:          u.tokenBlueprintReviewUC,
		UserUC:                          u.userUC,
		WalletUC:                        u.walletUC,
		CartUC:                          u.cartUC,
		CompanyProductionQueryService:   q.companyProductionQueryService,
		MintRequestQueryService:         q.mintRequestQueryService,
		MintFundingEstimateQuery:        q.mintFundingEstimateQuery,
		BrandManagementQuery:            q.brandManagementQuery,
		BrandDetailQuery:                q.brandDetailQuery,
		ProductBlueprintManagementQuery: q.productBlueprintManagementQuery,
		ProductBlueprintDetailQuery:     q.productBlueprintDetailQuery,
		TokenBlueprintManagementQuery:   q.tokenBlueprintManagementQuery,
		TokenBlueprintDetailQuery:       q.tokenBlueprintDetailQuery,
		LocationManagementQuery:         q.locationManagementQuery,
		LocationDetailQuery:             q.locationDetailQuery,
		TransportationManagementQuery:   q.transportationManagementQuery,
		TransportationDetailQuery:       q.transportationDetailQuery,
		InquiryManagementQuery:          q.inquiryManagementQuery,
		InquiryDetailQuery:              q.inquiryDetailQuery,
		InventoryManagementQuery:        q.inventoryManagementQuery,
		InventoryDetailQuery:            q.inventoryDetailQuery,
		ListCreateQuery:                 q.listCreateQuery,
		SalesQuery:                      q.salesQuery,
		AnnouncementManagementQuery:     announcementManagementQuery,
		AnnouncementDetailQuery:         announcementDetailQuery,
		PrintQueryService:               q.printQueryService,
		ListManagementQuery:             q.listManagementQuery,
		ListDetailQuery:                 q.listDetailQuery,
		OrderManagementQuery:            orderMgmtQ,
		OrderDetailQuery:                q.orderDetailQuery,
		TransactionManagementQuery:      q.transactionManagementQuery,
		InspectorQuery:                  q.inspectorQuery,
		InventoryBlueprintResolver:      invBlueprint,
		OwnerResolveQ:                   res.ownerResolveQuery,
		InspectionUC:                    u.inspectionUC,
		MintUC:                          u.mintUC,
		InvitationUC:                    u.invitationUC,
		InvitationDeliveryUC:            u.invitationDeliveryUC,
		AuthBootstrap:                   u.authBootstrapSvc,
		NameResolver:                    res.nameResolver,
	}, nil
}

func (c *Container) Close() error {
	if c == nil {
		return nil
	}

	var resourcesErr error
	if c.resources != nil {
		resourcesErr = c.resources.Close()
	}

	var infraErr error
	if c.Infra != nil {
		infraErr = c.Infra.Close()
	}

	return errors.Join(
		resourcesErr,
		infraErr,
	)
}
