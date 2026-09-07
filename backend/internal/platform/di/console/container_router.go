// backend/internal/platform/di/console/container_router.go
package console

import (
	"context"
	"net/http"

	httpin "narratives/internal/adapters/in/http/console"
	consoleHandler "narratives/internal/adapters/in/http/console/handler"
	internalHandler "narratives/internal/adapters/in/http/handler"
	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
)

func (c *Container) RouterDeps() httpin.RouterDeps {
	var authMw *middleware.AuthMiddleware
	if c.Infra.FirebaseAuth != nil && c.MemberRepo != nil {
		authMw = &middleware.AuthMiddleware{
			FirebaseAuth: c.Infra.FirebaseAuth,
			MemberRepo:   c.MemberRepo,
		}
	}

	var bootstrapMw *middleware.BootstrapAuthMiddleware
	if c.Infra.FirebaseAuth != nil {
		bootstrapMw = &middleware.BootstrapAuthMiddleware{
			FirebaseAuth: c.Infra.FirebaseAuth,
		}
	}

	var (
		authBootstrapH                                http.Handler
		accountsH                                     http.Handler
		announcementsH                                http.Handler
		reportDecisionNotificationsH                  http.Handler
		permissionsH                                  http.Handler
		brandsH                                       http.Handler
		companiesH                                    http.Handler
		companyShippingAddressesH                     http.Handler
		transportationH                               http.Handler
		inquiriesH                                    http.Handler
		inventoriesH                                  http.Handler
		listsH                                        http.Handler
		listSaveOperationsH                           http.Handler
		internalListSaveOperationTasksH               http.Handler
		internalTokenBlueprintCreateOperationTasksH   http.Handler
		salesH                                        http.Handler
		productsPrintH                                http.Handler
		productBPH                                    http.Handler
		productBPCategoriesH                          http.Handler
		tokenBPH                                      http.Handler
		tokenBPCreateOperationsH                      http.Handler
		tokenBPReviewH                                http.Handler
		productBPReviewH                              http.Handler
		messagesH                                     http.Handler
		ordersH                                       http.Handler
		transactionsH                                 http.Handler
		walletsH                                      http.Handler
		membersH                                      http.Handler
		productionsH                                  http.Handler
		modelsH                                       http.Handler
		inspectorH                                    http.Handler
		mintH                                         http.Handler
		internalMintTasksH                            http.Handler
		invitationH                                   http.Handler
		internalInvitationDeliveryProcessH            http.Handler
		internalInvitationDeliveryDispatchH           http.Handler
		internalOrderDispatchNotificationProcessH     http.Handler
		internalOrderDispatchNotificationDispatchH    http.Handler
		internalRefundCompletionNotificationProcessH  http.Handler
		internalRefundCompletionNotificationDispatchH http.Handler
		internalSettlementProcessH                    http.Handler
		internalSettlementDispatchH                   http.Handler
		ownerResolveH                                 http.Handler
	)

	if c.AuthBootstrap != nil && bootstrapMw != nil {
		authBootstrapH = consoleHandler.NewAuthBootstrapHandler(c.AuthBootstrap)
	}

	if c.AccountUC != nil {
		accountsH = consoleHandler.NewAccountHandler(c.AccountUC)
	}

	if c.AnnouncementUC != nil &&
		c.AnnouncementManagementQuery != nil &&
		c.AnnouncementDetailQuery != nil {
		announcementsH = consoleHandler.NewAnnouncementHandler(
			c.AnnouncementUC,
			c.AnnouncementManagementQuery,
			c.AnnouncementDetailQuery,
		)
	}

	if c.ReportUC != nil {
		reportDecisionNotificationsH =
			consoleHandler.NewReportDecisionNotificationHandler(
				c.ReportUC,
			)
	}

	if c.PermissionUC != nil {
		permissionsH = consoleHandler.NewPermissionHandler(c.PermissionUC)
	}

	if c.BrandUC != nil &&
		c.BrandManagementQuery != nil &&
		c.BrandDetailQuery != nil {
		brandsH = consoleHandler.NewBrandHandler(
			c.BrandUC,
			c.BrandManagementQuery,
			c.BrandDetailQuery,
		)
	}

	if c.CompanyUC != nil && c.CompanyQuery != nil {
		companiesH = consoleHandler.NewCompanyHandler(
			c.CompanyUC,
			c.CompanyQuery,
		)
	}

	if c.ShippingAddressUC != nil &&
		c.LocationManagementQuery != nil &&
		c.LocationDetailQuery != nil {
		companyShippingAddressesH = consoleHandler.NewCompanyShippingAddressHandler(
			c.ShippingAddressUC,
			c.LocationManagementQuery,
			c.LocationDetailQuery,
		)
	}

	if c.TransportationUC != nil &&
		c.TransportationManagementQuery != nil &&
		c.TransportationDetailQuery != nil {
		transportationH = consoleHandler.NewTransportationHandler(
			c.TransportationUC,
			c.TransportationManagementQuery,
			c.TransportationDetailQuery,
		)
	}

	if c.InquiryUC != nil &&
		c.ReturnReceiptUC != nil &&
		c.OpenedReturnReceiptUC != nil &&
		c.InquiryManagementQuery != nil &&
		c.InquiryDetailQuery != nil {
		inquiriesH = consoleHandler.NewInquiryHandler(
			c.InquiryUC,
			c.ReturnReceiptUC,
			c.OpenedReturnReceiptUC,
			c.InquiryManagementQuery,
			c.InquiryDetailQuery,
		)
	}

	if c.InventoryUC != nil &&
		c.InventoryManagementQuery != nil &&
		c.InventoryDetailQuery != nil &&
		c.ListCreateQuery != nil {
		inventoriesH = consoleHandler.NewInventoryHandlerWithListCreateQuery(
			c.InventoryUC,
			c.InventoryManagementQuery,
			c.InventoryDetailQuery,
			c.ListCreateQuery,
		)
	}

	if c.ListUC != nil {
		listsH = consoleHandler.NewListHandler(
			consoleHandler.NewListHandlerParams{
				UC:      c.ListUC,
				QMgmt:   c.ListManagementQuery,
				QDetail: c.ListDetailQuery,
			},
		)
	}

	if c.ListSaveOperationUC != nil {
		listSaveOperationsH = consoleHandler.NewListSaveOperationHandler(
			consoleHandler.NewListSaveOperationHandlerParams{
				UC: c.ListSaveOperationUC,
			},
		)

		internalListSaveOperationTasksH = consoleHandler.NewListSaveOperationTaskHandler(
			consoleHandler.NewListSaveOperationTaskHandlerParams{
				UC: c.ListSaveOperationUC,
			},
		)
	}

	if c.SalesQuery != nil {
		salesH = &consoleHandler.SalesHandler{
			SalesQuery: c.SalesQuery,
		}
	}

	if c.PrintUC != nil && c.PrintQueryService != nil {
		productsPrintH = consoleHandler.NewPrintHandler(
			c.PrintUC,
			c.PrintQueryService,
		)
	}

	if c.ProductBlueprintUC != nil &&
		c.ProductBlueprintManagementQuery != nil &&
		c.ProductBlueprintDetailQuery != nil {
		productBPH = consoleHandler.NewProductBlueprintHandler(
			c.ProductBlueprintUC,
			c.ProductBlueprintManagementQuery,
			c.ProductBlueprintDetailQuery,
		)
	}

	if c.ProductBlueprintCategoryUC != nil {
		productBPCategoriesH = consoleHandler.NewProductBlueprintCategoryHandler(
			c.ProductBlueprintCategoryUC,
		)
	}

	if c.TokenBlueprintUC != nil &&
		c.TokenBlueprintManagementQuery != nil &&
		c.TokenBlueprintDetailQuery != nil {
		tokenBPH = consoleHandler.NewTokenBlueprintHandler(
			c.TokenBlueprintUC,
			c.TokenBlueprintDetailQuery,
			c.TokenBlueprintManagementQuery,
		)
	}

	if c.TokenBlueprintCreateOperationUC != nil {
		tokenBPCreateOperationsH = consoleHandler.NewTokenBlueprintCreateOperationHandler(
			consoleHandler.NewTokenBlueprintCreateOperationHandlerParams{
				UC: c.TokenBlueprintCreateOperationUC,
			},
		)

		internalTokenBlueprintCreateOperationTasksH = consoleHandler.NewTokenBlueprintCreateOperationTaskHandler(
			consoleHandler.NewTokenBlueprintCreateOperationTaskHandlerParams{
				UC: c.TokenBlueprintCreateOperationUC,
			},
		)
	}

	if c.TokenBlueprintRepo != nil &&
		c.TokenBlueprintReviewRepo != nil &&
		c.BrandRepo != nil {
		tbReviewUC := usecase.NewTokenBlueprintReviewUsecase(
			c.TokenBlueprintReviewRepo,
			c.AvatarRepo,
			c.TokenBlueprintRepo,
			c.BrandRepo,
		)

		tokenBPReviewH = consoleHandler.NewTokenBlueprintReviewHandler(
			tbReviewUC,
			c.ReportUC,
		)
	}

	if c.ProductBlueprintRepo != nil &&
		c.ProductBlueprintReviewRepo != nil &&
		c.BrandRepo != nil &&
		c.WalletUC != nil {
		pbReviewUC := usecase.NewProductBlueprintReviewUsecase(
			c.ProductBlueprintReviewRepo,
			c.ProductBlueprintRepo,
			c.BrandRepo,
			c.MemberRepo,
			c.WalletUC,
			c.AvatarRepo,
			nil,
		)

		productBPReviewH = consoleHandler.NewProductBlueprintReviewHandler(
			pbReviewUC,
			c.ReportUC,
		)
	}

	if c.OrderUC != nil &&
		c.OrderManagementQuery != nil &&
		c.OrderDetailQuery != nil {
		ordersH = consoleHandler.NewOrderHandler(
			c.OrderUC,
			c.PaymentFlowUC,
			c.PaymentUC,
			c.SettlementUC,
			c.RefundUC,
			c.SettlementQueue,
			c.OrderManagementQuery,
			c.OrderDetailQuery,
			c.OrderDispatchNotificationUC,
		)
	}

	if c.TransactionManagementQuery != nil {
		transactionsH = consoleHandler.NewTransactionHandler(
			c.TransactionManagementQuery,
		)
	}

	if c.WalletUC != nil {
		walletsH = consoleHandler.NewWalletHandler(c.WalletUC)
	}

	if c.MemberRepo != nil {
		membersH = consoleHandler.NewMemberHandler(c.MemberRepo)
	}

	if c.ProductionUC != nil && c.CompanyProductionQueryService != nil {
		productionsH = consoleHandler.NewProductionHandler(
			c.CompanyProductionQueryService,
			c.ProductionUC,
		)
	}

	if c.ModelUC != nil && c.ProductBlueprintRepo != nil {
		modelAccessPolicy := consoleHandler.NewModelAccessPolicy(
			func(
				ctx context.Context,
				productBlueprintID string,
			) (consoleHandler.ProductBlueprintAccess, error) {
				productBlueprint, err := c.ProductBlueprintRepo.GetByID(
					ctx,
					productBlueprintID,
				)
				if err != nil {
					return consoleHandler.ProductBlueprintAccess{}, err
				}

				return consoleHandler.ProductBlueprintAccess{
					CompanyID: productBlueprint.CompanyID,
					Printed:   productBlueprint.Printed,
				}, nil
			},
		)

		modelsH = consoleHandler.NewModelHandler(
			c.ModelUC,
			modelAccessPolicy,
		)
	}

	if c.InspectionUC != nil && c.InspectorQuery != nil {
		var pbGetter consoleHandler.ProductBlueprintModelRefGetter

		if c.ProductBlueprintRepo != nil {
			if getter, ok := any(c.ProductBlueprintRepo).(consoleHandler.ProductBlueprintModelRefGetter); ok {
				pbGetter = getter
			}
		}

		inspectorH = consoleHandler.NewInspectorHandler(
			c.InspectionUC,
			c.InspectorQuery,
			c.NameResolver,
			pbGetter,
		)
	}

	if c.MintUC != nil {
		mintH = consoleHandler.NewMintHandler(
			c.MintUC,
			c.MintRequestQueryService,
			c.MintFundingEstimateQuery,
		)

		internalMintTasksH = internalHandler.NewMintTaskHandler(c.MintUC)
	}

	if c.InvitationDeliveryUC != nil {
		invitationDeliveryHandler := internalHandler.NewInvitationDeliveryHandler(
			c.InvitationDeliveryUC,
		)

		internalInvitationDeliveryProcessH = http.HandlerFunc(
			invitationDeliveryHandler.Process,
		)

		internalInvitationDeliveryDispatchH = http.HandlerFunc(
			invitationDeliveryHandler.DispatchDue,
		)
	}

	if c.OrderDispatchNotificationUC != nil {
		orderDispatchNotificationHandler := internalHandler.NewOrderDispatchNotificationHandler(
			c.OrderDispatchNotificationUC,
		)

		internalOrderDispatchNotificationProcessH = http.HandlerFunc(
			orderDispatchNotificationHandler.Process,
		)

		internalOrderDispatchNotificationDispatchH = http.HandlerFunc(
			orderDispatchNotificationHandler.DispatchDue,
		)
	}

	if c.RefundCompletionNotificationUC != nil {
		refundCompletionNotificationHandler := internalHandler.NewRefundCompletionNotificationHandler(
			c.RefundCompletionNotificationUC,
		)

		internalRefundCompletionNotificationProcessH = http.HandlerFunc(
			refundCompletionNotificationHandler.Process,
		)

		internalRefundCompletionNotificationDispatchH = http.HandlerFunc(
			refundCompletionNotificationHandler.DispatchDue,
		)
	}

	if c.SettlementUC != nil && c.SettlementQueue != nil {
		settlementTaskHandler := internalHandler.NewSettlementTaskHandler(
			c.SettlementUC,
			c.SettlementQueue,
		)

		internalSettlementProcessH = http.HandlerFunc(
			settlementTaskHandler.Process,
		)

		internalSettlementDispatchH = http.HandlerFunc(
			settlementTaskHandler.DispatchDue,
		)
	}

	if c.OwnerResolveQ != nil {
		ownerResolveH = consoleHandler.NewOwnerResolveHandler(c.OwnerResolveQ)
	}

	if c.InvitationUC != nil && c.Infra.FirebaseAuth != nil {
		invitationH = consoleHandler.NewInvitationHandler(
			c.InvitationUC,
			c.CompanyRepo,
			c.BrandRepo,
			c.Infra.FirebaseAuth,
		)
	}

	return httpin.RouterDeps{
		AuthMw:                         authMw,
		BootstrapMw:                    bootstrapMw,
		AuthBootstrap:                  authBootstrapH,
		Accounts:                       accountsH,
		Announcements:                  announcementsH,
		ReportDecisionNotifications:    reportDecisionNotificationsH,
		Permissions:                    permissionsH,
		Brands:                         brandsH,
		Companies:                      companiesH,
		CompanyShippingAddresses:       companyShippingAddressesH,
		Transportation:                 transportationH,
		Inquiries:                      inquiriesH,
		Inventories:                    inventoriesH,
		Lists:                          listsH,
		ListSaveOperations:             listSaveOperationsH,
		ProductsPrint:                  productsPrintH,
		ProductBP:                      productBPH,
		ProductBPCategories:            productBPCategoriesH,
		TokenBP:                        tokenBPH,
		TokenBPCreateOperations:        tokenBPCreateOperationsH,
		Messages:                       messagesH,
		Orders:                         ordersH,
		Transactions:                   transactionsH,
		Wallets:                        walletsH,
		Members:                        membersH,
		Productions:                    productionsH,
		Models:                         modelsH,
		Inspector:                      inspectorH,
		Mint:                           mintH,
		InternalMintTasks:              internalMintTasksH,
		InternalListSaveOperationTasks: internalListSaveOperationTasksH,
		InternalTokenBlueprintCreateOperationTasks:   internalTokenBlueprintCreateOperationTasksH,
		InternalInvitationDeliveryProcess:            internalInvitationDeliveryProcessH,
		InternalInvitationDeliveryDispatch:           internalInvitationDeliveryDispatchH,
		InternalOrderDispatchNotificationProcess:     internalOrderDispatchNotificationProcessH,
		InternalOrderDispatchNotificationDispatch:    internalOrderDispatchNotificationDispatchH,
		InternalRefundCompletionNotificationProcess:  internalRefundCompletionNotificationProcessH,
		InternalRefundCompletionNotificationDispatch: internalRefundCompletionNotificationDispatchH,
		InternalSettlementProcess:                    internalSettlementProcessH,
		InternalSettlementDispatch:                   internalSettlementDispatchH,
		OwnerResolve:                                 ownerResolveH,
		Invitation:                                   invitationH,
		Sales:                                        salesH,
		TokenBPReview:                                tokenBPReviewH,
		ProductBPReview:                              productBPReviewH,
	}
}
