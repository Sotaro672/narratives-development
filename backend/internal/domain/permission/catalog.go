// backend/internal/domain/permission/catalog.go
package permission

// static な権限カタログ（バックエンドが唯一の真実）
// Permission エンティティ側のポリシーにより、
// action は read-only 系 ("read" / "list" / "view" / "export") のみ許可。
// そのため name は必ず "<category>[.<subscope>].(read|list|view|export)" の形に統一する。
var allPermissions = []Permission{
	// Wallet
	MustNew("perm_wallet_view", "wallet.view", "ウォレット情報の閲覧", CategoryWallet),
	MustNew("perm_wallet_edit", "wallet.settings.view", "ウォレット設定情報の閲覧", CategoryWallet),

	// Inquiry
	MustNew("perm_inquiry_view", "inquiry.view", "問い合わせ一覧の閲覧", CategoryInquiry),
	MustNew("perm_inquiry_manage", "inquiry.detail.view", "問い合わせ詳細・履歴の閲覧", CategoryInquiry),

	// Organization
	MustNew("perm_org_admin", "organization.settings.view", "組織設定および構成情報の閲覧", CategoryOrganization),

	// Brand
	MustNew("perm_brand_create", "brand.view", "ブランド情報の閲覧", CategoryBrand),
	MustNew("perm_brand_edit", "brand.detail.view", "ブランド詳細情報の閲覧", CategoryBrand),
	MustNew("perm_brand_delete", "brand.archive.view", "アーカイブ済みブランド情報の閲覧", CategoryBrand),

	// Token
	MustNew("perm_token_create", "token.view", "トークン情報の閲覧", CategoryToken),
	MustNew("perm_token_manage", "token.distribution.view", "トークン配布履歴・割当状況の閲覧", CategoryToken),

	// Order
	MustNew("perm_order_manage", "order.view", "注文情報の閲覧", CategoryOrder),

	// Member
	MustNew("perm_member_view", "member.view", "メンバー情報の閲覧", CategoryMember),
	MustNew("perm_member_edit", "member.roles.view", "メンバー権限・ロール設定の閲覧", CategoryMember),

	// Inventory
	MustNew("perm_inventory_view", "inventory.view", "在庫情報の閲覧", CategoryInventory),

	// Production
	MustNew("perm_production_manage", "production.status.view", "生産工程ステータスの閲覧", CategoryProduction),

	// System
	MustNew("perm_system_admin", "system.admin.view", "システム設定および管理情報の閲覧", CategorySystem),
}

// AllPermissions は定義済みの権限一覧をコピーして返す
func AllPermissions() []Permission {
	out := make([]Permission, len(allPermissions))
	copy(out, allPermissions)
	return out
}

// AllPermissionNames は name だけのスライスを返す（例: "wallet.view", ...）
func AllPermissionNames() []string {
	names := make([]string, 0, len(allPermissions))
	for _, p := range allPermissions {
		names = append(names, p.Name)
	}
	return names
}
