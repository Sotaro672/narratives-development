// backend/internal/platform/di/console/container.go
package console

import (
	"context"
	"errors"
	"log"

	fs "narratives/internal/adapters/out/firestore"

	inspectionapp "narratives/internal/application/inspection"
	mintapp "narratives/internal/application/mint"
	pbuc "narratives/internal/application/productBlueprint/usecase"
	productionapp "narratives/internal/application/production"
	companyquery "narratives/internal/application/query/console"
	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"
	tokenblueprintapp "narratives/internal/application/tokenBlueprint"
	uc "narratives/internal/application/usecase"
	authuc "narratives/internal/application/usecase/auth"

	shared "narratives/internal/platform/di/shared"

	branddom "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
)

// ========================================
// Container (Console DI)
// ========================================
type Container struct {
	Infra *shared.Infra

	// Repositories (AuthMiddleware 用に memberRepo だけ保持)
	MemberRepo  memdom.Repository
	MessageRepo *fs.MessageRepositoryFS

	// member.Service / brand.Service (表示名解決用)
	MemberService *memdom.Service
	BrandService  *branddom.Service

	// History Repositories
	ProductBlueprintHistoryRepo *fs.ProductBlueprintHistoryRepositoryFS
	ModelHistoryRepo            *fs.ModelHistoryRepositoryFS

	// Application-layer usecases
	AccountUC          *uc.AccountUsecase
	AnnouncementUC     *uc.AnnouncementUsecase
	AvatarUC           *uc.AvatarUsecase
	BillingAddressUC   *uc.BillingAddressUsecase
	BrandUC            *uc.BrandUsecase
	CampaignUC         *uc.CampaignUsecase
	CompanyUC          *uc.CompanyUsecase
	InquiryUC          *uc.InquiryUsecase
	InventoryUC        *uc.InventoryUsecase
	InvoiceUC          *uc.InvoiceUsecase
	ListUC             *uc.ListUsecase
	MemberUC           *uc.MemberUsecase
	MessageUC          *uc.MessageUsecase
	ModelUC            *uc.ModelUsecase
	OrderUC            *uc.OrderUsecase
	PaymentUC          *uc.PaymentUsecase
	PermissionUC       *uc.PermissionUsecase
	PrintUC            *uc.PrintUsecase
	ProductionUC       *productionapp.ProductionUsecase
	ProductBlueprintUC *pbuc.ProductBlueprintUsecase
	ShippingAddressUC  *uc.ShippingAddressUsecase
	TokenUC            *uc.TokenUsecase

	TokenBlueprintUC      *tokenblueprintapp.TokenBlueprintUsecase
	TokenBlueprintQueryUC *tokenblueprintapp.TokenBlueprintQueryUsecase

	TokenOperationUC *uc.TokenOperationUsecase
	TrackingUC       *uc.TrackingUsecase
	UserUC           *uc.UserUsecase
	WalletUC         *uc.WalletUsecase

	CartUC *uc.CartUsecase
	PostUC *uc.PostUsecase

	CompanyProductionQueryService *companyquery.CompanyProductionQueryService
	MintRequestQueryService       *companyquery.MintRequestQueryService
	InventoryQuery                *companyquery.InventoryQuery
	ListCreateQuery               *companyquery.ListCreateQuery
	ListManagementQuery           *companyquery.ListManagementQuery
	ListDetailQuery               *companyquery.ListDetailQuery

	OwnerResolveQ *sharedquery.OwnerResolveQuery

	ProductUC    *uc.ProductUsecase
	InspectionUC *inspectionapp.InspectionUsecase
	MintUC       *mintapp.MintUsecase

	InvitationQuery   uc.InvitationQueryPort
	InvitationCommand uc.InvitationCommandPort

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
	q := buildQueries(repos, res, u)

	if clients == nil || clients.infra == nil {
		return nil, errors.New("clients/infra is nil")
	}

	return &Container{
		Infra: clients.infra,

		MemberRepo:  repos.memberRepo,
		MessageRepo: repos.messageRepo,

		MemberService: services.memberSvc,
		BrandService:  services.brandSvc,

		ProductBlueprintHistoryRepo: repos.productBlueprintHistoryRepo,
		ModelHistoryRepo:            repos.modelHistoryRepo,

		AccountUC:          u.accountUC,
		AnnouncementUC:     u.announcementUC,
		AvatarUC:           u.avatarUC,
		BillingAddressUC:   u.billingAddressUC,
		BrandUC:            u.brandUC,
		CampaignUC:         u.campaignUC,
		CompanyUC:          u.companyUC,
		InquiryUC:          u.inquiryUC,
		InventoryUC:        u.inventoryUC,
		InvoiceUC:          u.invoiceUC,
		ListUC:             u.listUC,
		MemberUC:           u.memberUC,
		MessageUC:          u.messageUC,
		ModelUC:            u.modelUC,
		OrderUC:            u.orderUC,
		PaymentUC:          u.paymentUC,
		PermissionUC:       u.permissionUC,
		PrintUC:            u.printUC,
		ProductionUC:       u.productionUC,
		ProductBlueprintUC: u.productBlueprintUC,
		ShippingAddressUC:  u.shippingAddressUC,
		TokenUC:            u.tokenUC,

		TokenBlueprintUC:      u.tokenBlueprintUC,
		TokenBlueprintQueryUC: u.tokenBlueprintQueryUC,

		TokenOperationUC: u.tokenOperationUC,
		TrackingUC:       u.trackingUC,
		UserUC:           u.userUC,
		WalletUC:         u.walletUC,

		CartUC: u.cartUC,

		CompanyProductionQueryService: q.companyProductionQueryService,
		MintRequestQueryService:       q.mintRequestQueryService,
		InventoryQuery:                q.inventoryQuery,
		ListCreateQuery:               q.listCreateQuery,
		ListManagementQuery:           q.listManagementQuery,
		ListDetailQuery:               q.listDetailQuery,

		OwnerResolveQ: res.ownerResolveQ,

		ProductUC:    u.productUC,
		InspectionUC: u.inspectionUC,
		MintUC:       u.mintUC,

		InvitationQuery:   u.invitationQueryUC,
		InvitationCommand: u.invitationCommandUC,

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
