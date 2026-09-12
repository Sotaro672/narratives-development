// backend/internal/application/query/admin/contract_detail_result.go
package query

type ContractDetailResult struct {
	Company           ContractCompanyRow            `json:"company"`
	Brands            []ContractBrandRow            `json:"brands"`
	Announcements     []ContractAnnouncementRow     `json:"announcements"`
	Lists             []ContractListRow             `json:"lists"`
	TokenBlueprints   []ContractTokenBlueprintRow   `json:"tokenBlueprints"`
	ProductBlueprints []ContractProductBlueprintRow `json:"productBlueprints"`
}

type ContractCompanyRow struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RepresentativeName string `json:"representativeName"`
	IsActive           bool   `json:"isActive"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type ContractBrandRow struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ManagerName          string `json:"managerName"`
	WebsiteURL           string `json:"websiteUrl"`
	BrandIcon            string `json:"brandIcon"`
	BrandBackgroundImage string `json:"brandBackgroundImage"`
	IsActive             bool   `json:"isActive"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type ContractAnnouncementRow struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	TokenBlueprintID  string `json:"tokenBlueprintId"`
	TokenName         string `json:"tokenName"`
	Published         bool   `json:"published"`
	TargetAvatarCount int    `json:"targetAvatarCount"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type ContractListRow struct {
	ID           string `json:"id"`
	ReadableID   string `json:"readableId"`
	InventoryID  string `json:"inventoryId"`
	Title        string `json:"title"`
	ProductName  string `json:"productName"`
	TokenName    string `json:"tokenName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Status       string `json:"status"`
	ReportCount  int    `json:"reportCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ContractTokenBlueprintRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Minted       bool   `json:"minted"`
	ReportCount  int    `json:"reportCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ContractProductBlueprintRow struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Printed      bool   `json:"printed"`
	ReportCount  int    `json:"reportCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}
