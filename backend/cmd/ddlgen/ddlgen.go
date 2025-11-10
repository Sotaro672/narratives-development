// backend/cmd/ddlgen/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	// ドメインごとに import（アルファベット順）
	"narratives/internal/domain/account"
	annoucement "narratives/internal/domain/announcement"
	annattach "narratives/internal/domain/announcementAttachment"
	"narratives/internal/domain/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
	avatarstate "narratives/internal/domain/avatarState"
	billingaddress "narratives/internal/domain/billingAddress"
	"narratives/internal/domain/brand"
	campaigndomain "narratives/internal/domain/campaign"
	campaignimage "narratives/internal/domain/campaignImage"
	campaignperf "narratives/internal/domain/campaignPerformance"
	"narratives/internal/domain/company"
	dsc "narratives/internal/domain/discount"
	fulfill "narratives/internal/domain/fulfillment"
	inquiry "narratives/internal/domain/inquiry"
	inquiryimg "narratives/internal/domain/inquiryImage"
	inventory "narratives/internal/domain/inventory"
	inv "narratives/internal/domain/invoice"
	listdomain "narratives/internal/domain/list"
	listimage "narratives/internal/domain/listImage"
	"narratives/internal/domain/member"
	messagedomain "narratives/internal/domain/message"
	messageimage "narratives/internal/domain/messageImage"
	mintrequest "narratives/internal/domain/mintRequest"
	modeldomain "narratives/internal/domain/model"
	orderdomain "narratives/internal/domain/order"
	orderitem "narratives/internal/domain/orderItem"
	paymentdomain "narratives/internal/domain/payment"
	permission "narratives/internal/domain/permission"
	"narratives/internal/domain/product"
	productblueprint "narratives/internal/domain/productBlueprint"
	"narratives/internal/domain/production"
	sale "narratives/internal/domain/sale"
	shippingaddress "narratives/internal/domain/shippingAddress"
	"narratives/internal/domain/token"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
	tokencontents "narratives/internal/domain/tokenContents"
	tokenicon "narratives/internal/domain/tokenIcon"
	tokenoperation "narratives/internal/domain/tokenOperation"
	trackingdom "narratives/internal/domain/tracking"
	trdom "narratives/internal/domain/transaction"
	userdomain "narratives/internal/domain/user"
	walletdom "narratives/internal/domain/wallet"
)

func mustWrite(path string, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}

func main() {
	outDir := filepath.Join("internal", "infra", "database", "migrations")

	// 出力ファイル（アルファベット順）
	outAccount := filepath.Join(outDir, "init_account.sql")
	outAnnoucement := filepath.Join(outDir, "init_annoucement.sql")
	outAnnoucementAttachment := filepath.Join(outDir, "init_annoucement_attachment.sql")
	outAvatar := filepath.Join(outDir, "init_avatar.sql")
	outAvatarIcon := filepath.Join(outDir, "init_avatar_icon.sql")
	outAvatarState := filepath.Join(outDir, "init_avatar_state.sql")
	outBillingAddress := filepath.Join(outDir, "init_billing_address.sql")
	outBrand := filepath.Join(outDir, "init_brand.sql")
	outCampaign := filepath.Join(outDir, "init_campaign.sql")
	outCampaignImage := filepath.Join(outDir, "init_campaign_image.sql")
	outCampaignPerformance := filepath.Join(outDir, "init_campaign_performance.sql")
	outCompany := filepath.Join(outDir, "init_company.sql")
	outDiscount := filepath.Join(outDir, "init_discount.sql")
	outFulfillment := filepath.Join(outDir, "init_fulfillment.sql")
	outInquiry := filepath.Join(outDir, "init_inquiry.sql")
	outInquiryImages := filepath.Join(outDir, "init_inquiry_images.sql")
	outInventory := filepath.Join(outDir, "init_inventory.sql")
	outInvoice := filepath.Join(outDir, "init_invoice.sql")
	outList := filepath.Join(outDir, "init_list.sql")
	outListImage := filepath.Join(outDir, "init_list_image.sql")
	outMember := filepath.Join(outDir, "init_member.sql")
	outMessage := filepath.Join(outDir, "init_message.sql")
	outMessageImage := filepath.Join(outDir, "init_message_image.sql")
	outMintRequest := filepath.Join(outDir, "init_mint_request.sql")
	outMintRequests := filepath.Join(outDir, "init_mint_requests.sql")
	outModel := filepath.Join(outDir, "init_model.sql")
	outOrderItems := filepath.Join(outDir, "init_order_items.sql")
	outOrders := filepath.Join(outDir, "init_orders.sql")
	outPayments := filepath.Join(outDir, "init_payments.sql")
	outPermission := filepath.Join(outDir, "init_permission.sql")
	outProduct := filepath.Join(outDir, "init_product.sql")
	outProductBlueprint := filepath.Join(outDir, "init_product_blueprint.sql")
	outProduction := filepath.Join(outDir, "init_production.sql")
	outShippingAddress := filepath.Join(outDir, "init_shipping_address.sql")
	outToken := filepath.Join(outDir, "init_token.sql")
	outTokenBlueprint := filepath.Join(outDir, "init_token_blueprint.sql")
	outTokenContents := filepath.Join(outDir, "init_token_contents.sql")
	outTokenIcon := filepath.Join(outDir, "init_token_icon.sql")
	outUser := filepath.Join(outDir, "init_user.sql")
	outWallet := filepath.Join(outDir, "init_wallet.sql")

	// mustWrite 実行（アルファベット順）※ inventory の重複を解消
	mustWrite(outAccount, account.AccountsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAccount))

	mustWrite(outAnnoucement, annoucement.AnnouncementsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAnnoucement))

	mustWrite(outAnnoucementAttachment, annattach.AnnouncementAttachmentsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAnnoucementAttachment))

	mustWrite(outAvatar, avatar.AvatarsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAvatar))

	mustWrite(outAvatarIcon, avataricon.AvatarIconsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAvatarIcon))

	mustWrite(outAvatarState, avatarstate.AvatarStatesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outAvatarState))

	mustWrite(outBillingAddress, billingaddress.BillingAddressesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outBillingAddress))

	mustWrite(outBrand, brand.BrandsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outBrand))

	mustWrite(outCampaign, campaigndomain.CampaignsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outCampaign))

	mustWrite(outCampaignImage, campaignimage.CampaignImagesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outCampaignImage))

	mustWrite(outCampaignPerformance, campaignperf.CampaignPerformanceTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outCampaignPerformance))

	mustWrite(outCompany, company.CompaniesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outCompany))

	mustWrite(outDiscount, dsc.DiscountsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDiscount))

	mustWrite(outFulfillment, fulfill.FulfillmentsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outFulfillment))

	mustWrite(outInquiry, inquiry.InquiriesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outInquiry))

	mustWrite(outInquiryImages, inquiryimg.InquiryImagesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outInquiryImages))

	mustWrite(outInventory, inventory.InventoriesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outInventory))

	mustWrite(outInvoice, inv.InvoicesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outInvoice))

	mustWrite(outList, listdomain.ListsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outList))

	mustWrite(outListImage, listimage.ListImagesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outListImage))

	mustWrite(outMember, member.MembersTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outMember))

	mustWrite(outMessage, messagedomain.MessagesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outMessage))

	mustWrite(outMessageImage, messageimage.MessageImagesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outMessageImage))

	mustWrite(outMintRequest, mintrequest.MintRequestsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outMintRequest))

	mustWrite(outMintRequests, mintrequest.MintRequestsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outMintRequests))

	mustWrite(outModel, modeldomain.ModelsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outModel))

	mustWrite(outOrderItems, orderitem.OrderItemsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outOrderItems))

	mustWrite(outOrders, orderdomain.OrdersTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outOrders))

	mustWrite(outPayments, paymentdomain.PaymentsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outPayments))

	mustWrite(outPermission, permission.PermissionsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outPermission))

	mustWrite(outProduct, product.ProductsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outProduct))

	mustWrite(outProductBlueprint, productblueprint.ProductBlueprintsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outProductBlueprint))

	mustWrite(outProduction, production.ProductionPlansTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outProduction))

	mustWrite(outShippingAddress, shippingaddress.ShippingAddressesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outShippingAddress))

	mustWrite(outToken, token.TokensTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outToken))

	mustWrite(outTokenBlueprint, tokenblueprint.TokenBlueprintsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outTokenBlueprint))

	mustWrite(outTokenContents, tokencontents.TokenContentsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outTokenContents))

	mustWrite(outTokenIcon, tokenicon.TokenIconsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outTokenIcon))

	mustWrite(outUser, userdomain.UsersTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outUser))

	mustWrite(outWallet, walletdom.WalletsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outWallet))

	// 直接書き出し（既存ロジック、順序は既にアルファベット順）
	mustWrite(filepath.Join(outDir, "init_sales.sql"), sale.SalesTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_sales.sql"))

	mustWrite(filepath.Join(outDir, "init_tokens.sql"), token.TokensTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_tokens.sql"))

	mustWrite(filepath.Join(outDir, "init_token_blueprints.sql"), tokenblueprint.TokenBlueprintsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_token_blueprints.sql"))

	mustWrite(filepath.Join(outDir, "init_token_contents.sql"), tokencontents.TokenContentsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_token_contents.sql"))

	mustWrite(filepath.Join(outDir, "init_token_icons.sql"), tokenicon.TokenIconsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_token_icons.sql"))

	mustWrite(filepath.Join(outDir, "init_token_operation.sql"), tokenoperation.TokenOperationDomainDDLs)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_token_operation.sql"))

	mustWrite(filepath.Join(outDir, "init_trackings.sql"), trackingdom.TrackingsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_trackings.sql"))

	mustWrite(filepath.Join(outDir, "init_transactions.sql"), trdom.TransactionsTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_transactions.sql"))

	mustWrite(filepath.Join(outDir, "init_users.sql"), userdomain.UsersTableDDL)
	fmt.Println("✅ Generated:", filepath.Join("backend", outDir, "init_users.sql"))
}
