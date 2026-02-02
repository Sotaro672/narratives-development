// backend/internal/platform/di/console/container_router.go
package console

import (
	httpin "narratives/internal/adapters/in/http/console"
)

func (c *Container) RouterDeps() httpin.RouterDeps {
	return httpin.RouterDeps{
		AccountUC:          c.AccountUC,
		AnnouncementUC:     c.AnnouncementUC,
		AvatarUC:           c.AvatarUC,
		BillingAddressUC:   c.BillingAddressUC,
		BrandUC:            c.BrandUC,
		CampaignUC:         c.CampaignUC,
		CompanyUC:          c.CompanyUC,
		InquiryUC:          c.InquiryUC,
		InventoryUC:        c.InventoryUC,
		InvoiceUC:          c.InvoiceUC,
		ListUC:             c.ListUC,
		MemberUC:           c.MemberUC,
		MessageUC:          c.MessageUC,
		ModelUC:            c.ModelUC,
		OrderUC:            c.OrderUC,
		PaymentUC:          c.PaymentUC,
		PermissionUC:       c.PermissionUC,
		PrintUC:            c.PrintUC,
		TokenUC:            c.TokenUC,
		ProductionUC:       c.ProductionUC,
		ProductBlueprintUC: c.ProductBlueprintUC,
		ShippingAddressUC:  c.ShippingAddressUC,

		TokenBlueprintUC:      c.TokenBlueprintUC,
		TokenBlueprintQueryUC: c.TokenBlueprintQueryUC,

		TokenOperationUC: c.TokenOperationUC,
		TrackingUC:       c.TrackingUC,
		UserUC:           c.UserUC,
		WalletUC:         c.WalletUC,

		CompanyProductionQueryService: c.CompanyProductionQueryService,
		MintRequestQueryService:       c.MintRequestQueryService,

		InventoryQuery:  c.InventoryQuery,
		ListCreateQuery: c.ListCreateQuery,

		ListManagementQuery: c.ListManagementQuery,
		ListDetailQuery:     c.ListDetailQuery,

		// ✅ NEW: ListImage endpoints wiring (upload/delete)
		ListImageUploader: c.ListImageUploader,
		ListImageDeleter:  c.ListImageDeleter,

		ProductUC:    c.ProductUC,
		InspectionUC: c.InspectionUC,
		MintUC:       c.MintUC,

		InvitationQuery:   c.InvitationQuery,
		InvitationCommand: c.InvitationCommand,

		AuthBootstrap: c.AuthBootstrap,

		FirebaseAuth: c.Infra.FirebaseAuth,
		MemberRepo:   c.MemberRepo,

		MemberService: c.MemberService,
		BrandService:  c.BrandService,
		NameResolver:  c.NameResolver,

		MessageRepo: c.MessageRepo,

		OwnerResolveQ: c.OwnerResolveQ,
	}
}
