package permission

// Static catalog of all permissions.
// This mirrors the frontend ALL_PERMISSIONS but backend is the true source of truth.
var AllPermissions = []Permission{
	MustNew("perm_wallet_view", "wallet.view", "ウォレット情報の閲覧", CategoryWallet),
	MustNew("perm_wallet_edit", "wallet.edit", "ウォレット設定の編集", CategoryWallet),

	MustNew("perm_inquiry_view", "inquiry.view", "問い合わせの閲覧", CategoryInquiry),
	MustNew("perm_inquiry_manage", "inquiry.manage", "問い合わせ対応・管理", CategoryInquiry),

	MustNew("perm_org_admin", "organization.admin", "組織の完全な管理権限", CategoryOrganization),

	MustNew("perm_brand_create", "brand.create", "ブランドの作成", CategoryBrand),
	MustNew("perm_brand_edit", "brand.edit", "ブランド情報の編集", CategoryBrand),
	MustNew("perm_brand_delete", "brand.delete", "ブランドの削除", CategoryBrand),

	MustNew("perm_token_create", "token.create", "トークンの作成", CategoryToken),
	MustNew("perm_token_manage", "token.manage", "トークンの管理・配布", CategoryToken),

	MustNew("perm_order_manage", "order.manage", "注文の管理", CategoryOrder),

	MustNew("perm_member_view", "member.view", "メンバー情報の閲覧", CategoryMember),
	MustNew("perm_member_edit", "member.edit", "メンバー情報の編集", CategoryMember),

	MustNew("perm_inventory_view", "inventory.view", "在庫情報の閲覧", CategoryInventory),

	MustNew("perm_production_manage", "production.manage", "生産工程の管理", CategoryProduction),

	MustNew("perm_system_admin", "system.admin", "システム管理全般", CategorySystem),
}

// Helper to extract permission names (for member assignment).
func AllPermissionNames() []string {
	out := make([]string, 0, len(AllPermissions))
	for _, p := range AllPermissions {
		out = append(out, p.Name)
	}
	return out
}
