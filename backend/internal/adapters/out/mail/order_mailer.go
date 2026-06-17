// backend/internal/adapters/out/mail/order_mailer.go
package mail

import (
	"context"
	"fmt"
	"sort"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	modeldom "narratives/internal/domain/model"
	orderdom "narratives/internal/domain/order"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type OrderModelGetter interface {
	GetByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

type OrderInventoryGetter interface {
	GetByID(ctx context.Context, id string) (inventorydom.Mint, error)
}

type OrderProductBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type OrderTokenBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type OrderBrandGetter interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type OrderCompanyGetter interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type OrderMailer struct {
	client               *ResendClient
	modelRepo            OrderModelGetter
	inventoryRepo        OrderInventoryGetter
	productBlueprintRepo OrderProductBlueprintGetter
	tokenBlueprintRepo   OrderTokenBlueprintGetter
	brandRepo            OrderBrandGetter
	companyRepo          OrderCompanyGetter
}

func NewOrderMailer(
	client *ResendClient,
	modelRepo OrderModelGetter,
	inventoryRepo OrderInventoryGetter,
	productBlueprintRepo OrderProductBlueprintGetter,
	tokenBlueprintRepo OrderTokenBlueprintGetter,
	brandRepo OrderBrandGetter,
	companyRepo OrderCompanyGetter,
) *OrderMailer {
	return &OrderMailer{
		client:               client,
		modelRepo:            modelRepo,
		inventoryRepo:        inventoryRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
		companyRepo:          companyRepo,
	}
}

func (m *OrderMailer) SendOrderConfirmation(ctx context.Context, from, to string, ord orderdom.Order) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("order mailer is nil")
	}

	if from == "" {
		return fmt.Errorf("from address is empty")
	}
	if to == "" {
		return fmt.Errorf("to address is empty")
	}
	if ord.ID == "" {
		return fmt.Errorf("order id is empty")
	}

	modelsByID, modelErrorsByID := m.loadOrderModels(ctx, ord)
	inventoriesByID, inventoryErrorsByID := m.loadOrderInventories(ctx, ord)
	productBlueprintsByID, productBlueprintErrorsByID := m.loadOrderProductBlueprints(ctx, inventoriesByID)
	tokenBlueprintsByID, tokenBlueprintErrorsByID := m.loadOrderTokenBlueprints(ctx, inventoriesByID)
	brandsByID, brandErrorsByID := m.loadOrderBrands(ctx, productBlueprintsByID, tokenBlueprintsByID)
	companiesByID, companyErrorsByID := m.loadOrderCompanies(ctx, productBlueprintsByID)

	subject := buildOrderConfirmationMailSubject(ord)
	body := buildOrderConfirmationMailBody(
		ord,
		modelsByID,
		modelErrorsByID,
		inventoriesByID,
		inventoryErrorsByID,
		productBlueprintsByID,
		productBlueprintErrorsByID,
		tokenBlueprintsByID,
		tokenBlueprintErrorsByID,
		brandsByID,
		brandErrorsByID,
		companiesByID,
		companyErrorsByID,
	)

	return m.client.Send(ctx, from, to, subject, body)
}

func (m *OrderMailer) loadOrderModels(
	ctx context.Context,
	ord orderdom.Order,
) (map[string]modeldom.ModelVariation, map[string]string) {
	modelsByID := map[string]modeldom.ModelVariation{}
	errorsByID := map[string]string{}

	if m == nil || m.modelRepo == nil {
		return modelsByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, it := range ord.Items {
		modelID := it.ModelID
		if modelID == "" {
			continue
		}

		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}

		model, err := m.modelRepo.GetByID(ctx, modelID)
		if err != nil {
			errorsByID[modelID] = err.Error()
			continue
		}

		modelsByID[modelID] = model
	}

	return modelsByID, errorsByID
}

func (m *OrderMailer) loadOrderInventories(
	ctx context.Context,
	ord orderdom.Order,
) (map[string]inventorydom.Mint, map[string]string) {
	inventoriesByID := map[string]inventorydom.Mint{}
	errorsByID := map[string]string{}

	if m == nil || m.inventoryRepo == nil {
		return inventoriesByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, it := range ord.Items {
		inventoryID := it.InventoryID
		if inventoryID == "" {
			continue
		}

		if _, ok := seen[inventoryID]; ok {
			continue
		}
		seen[inventoryID] = struct{}{}

		inv, err := m.inventoryRepo.GetByID(ctx, inventoryID)
		if err != nil {
			errorsByID[inventoryID] = err.Error()
			continue
		}

		inventoriesByID[inventoryID] = inv
	}

	return inventoriesByID, errorsByID
}

func (m *OrderMailer) loadOrderProductBlueprints(
	ctx context.Context,
	inventoriesByID map[string]inventorydom.Mint,
) (map[string]productblueprintdom.ProductBlueprint, map[string]string) {
	productBlueprintsByID := map[string]productblueprintdom.ProductBlueprint{}
	errorsByID := map[string]string{}

	if m == nil || m.productBlueprintRepo == nil {
		return productBlueprintsByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, inv := range inventoriesByID {
		productBlueprintID := inv.ProductBlueprintID
		if productBlueprintID == "" {
			continue
		}

		if _, ok := seen[productBlueprintID]; ok {
			continue
		}
		seen[productBlueprintID] = struct{}{}

		pb, err := m.productBlueprintRepo.GetByID(ctx, productBlueprintID)
		if err != nil {
			errorsByID[productBlueprintID] = err.Error()
			continue
		}

		productBlueprintsByID[productBlueprintID] = pb
	}

	return productBlueprintsByID, errorsByID
}

func (m *OrderMailer) loadOrderTokenBlueprints(
	ctx context.Context,
	inventoriesByID map[string]inventorydom.Mint,
) (map[string]*tokenblueprintdom.TokenBlueprint, map[string]string) {
	tokenBlueprintsByID := map[string]*tokenblueprintdom.TokenBlueprint{}
	errorsByID := map[string]string{}

	if m == nil || m.tokenBlueprintRepo == nil {
		return tokenBlueprintsByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, inv := range inventoriesByID {
		tokenBlueprintID := inv.TokenBlueprintID
		if tokenBlueprintID == "" {
			continue
		}

		if _, ok := seen[tokenBlueprintID]; ok {
			continue
		}
		seen[tokenBlueprintID] = struct{}{}

		tb, err := m.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
		if err != nil {
			errorsByID[tokenBlueprintID] = err.Error()
			continue
		}

		if tb == nil {
			errorsByID[tokenBlueprintID] = "トークン情報を取得できませんでした"
			continue
		}

		tokenBlueprintsByID[tokenBlueprintID] = tb
	}

	return tokenBlueprintsByID, errorsByID
}

func (m *OrderMailer) loadOrderBrands(
	ctx context.Context,
	productBlueprintsByID map[string]productblueprintdom.ProductBlueprint,
	tokenBlueprintsByID map[string]*tokenblueprintdom.TokenBlueprint,
) (map[string]branddom.Brand, map[string]string) {
	brandsByID := map[string]branddom.Brand{}
	errorsByID := map[string]string{}

	if m == nil || m.brandRepo == nil {
		return brandsByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, pb := range productBlueprintsByID {
		brandID := pb.BrandID
		if brandID != "" {
			seen[brandID] = struct{}{}
		}
	}

	for _, tb := range tokenBlueprintsByID {
		if tb == nil {
			continue
		}

		brandID := tb.BrandID
		if brandID != "" {
			seen[brandID] = struct{}{}
		}
	}

	for brandID := range seen {
		b, err := m.brandRepo.GetByID(ctx, brandID)
		if err != nil {
			errorsByID[brandID] = err.Error()
			continue
		}

		brandsByID[brandID] = b
	}

	return brandsByID, errorsByID
}

func (m *OrderMailer) loadOrderCompanies(
	ctx context.Context,
	productBlueprintsByID map[string]productblueprintdom.ProductBlueprint,
) (map[string]companydom.Company, map[string]string) {
	companiesByID := map[string]companydom.Company{}
	errorsByID := map[string]string{}

	if m == nil || m.companyRepo == nil {
		return companiesByID, errorsByID
	}

	seen := map[string]struct{}{}

	for _, pb := range productBlueprintsByID {
		companyID := pb.CompanyID
		if companyID != "" {
			seen[companyID] = struct{}{}
		}
	}

	for companyID := range seen {
		company, err := m.companyRepo.GetByID(ctx, companyID)
		if err != nil {
			errorsByID[companyID] = err.Error()
			continue
		}

		companiesByID[companyID] = company
	}

	return companiesByID, errorsByID
}

func buildOrderConfirmationMailSubject(_ orderdom.Order) string {
	return "【AMOL】ご注文が確定しました"
}

func buildOrderConfirmationMailBody(
	ord orderdom.Order,
	modelsByID map[string]modeldom.ModelVariation,
	modelErrorsByID map[string]string,
	inventoriesByID map[string]inventorydom.Mint,
	inventoryErrorsByID map[string]string,
	productBlueprintsByID map[string]productblueprintdom.ProductBlueprint,
	productBlueprintErrorsByID map[string]string,
	tokenBlueprintsByID map[string]*tokenblueprintdom.TokenBlueprint,
	tokenBlueprintErrorsByID map[string]string,
	brandsByID map[string]branddom.Brand,
	brandErrorsByID map[string]string,
	companiesByID map[string]companydom.Company,
	companyErrorsByID map[string]string,
) string {
	var b strings.Builder

	totalQty := 0
	totalPrice := 0
	for _, it := range ord.Items {
		totalQty += it.Qty
		totalPrice += it.Price * it.Qty
	}

	b.WriteString("ご注文ありがとうございます。\n")
	b.WriteString("ご注文が確定しました。\n\n")

	b.WriteString("ご注文内容\n")
	b.WriteString(fmt.Sprintf("商品点数: %d点\n", totalQty))
	b.WriteString(fmt.Sprintf("合計金額: %d円\n", totalPrice))
	b.WriteString("\n")

	writeShippingSnapshot(&b, ord)

	b.WriteString("商品明細\n")
	for i, it := range ord.Items {
		writeOrderItemForCustomerMail(
			&b,
			i+1,
			it,
			modelsByID,
			modelErrorsByID,
			inventoriesByID,
			inventoryErrorsByID,
			productBlueprintsByID,
			productBlueprintErrorsByID,
			tokenBlueprintsByID,
			tokenBlueprintErrorsByID,
			brandsByID,
			brandErrorsByID,
			companiesByID,
			companyErrorsByID,
		)
	}

	b.WriteString("本メールは自動送信です。\n")

	return b.String()
}

func writeShippingSnapshot(b *strings.Builder, ord orderdom.Order) {
	if b == nil {
		return
	}

	hasShipping := ord.ShippingSnapshot.ZipCode != "" ||
		ord.ShippingSnapshot.State != "" ||
		ord.ShippingSnapshot.City != "" ||
		ord.ShippingSnapshot.Street != "" ||
		ord.ShippingSnapshot.Street2 != "" ||
		ord.ShippingSnapshot.Country != ""

	if !hasShipping {
		return
	}

	b.WriteString("配送先\n")
	if ord.ShippingSnapshot.ZipCode != "" {
		b.WriteString(fmt.Sprintf("郵便番号: %s\n", ord.ShippingSnapshot.ZipCode))
	}
	if ord.ShippingSnapshot.State != "" {
		b.WriteString(fmt.Sprintf("都道府県/州: %s\n", ord.ShippingSnapshot.State))
	}
	if ord.ShippingSnapshot.City != "" {
		b.WriteString(fmt.Sprintf("市区町村: %s\n", ord.ShippingSnapshot.City))
	}
	if ord.ShippingSnapshot.Street != "" {
		b.WriteString(fmt.Sprintf("住所1: %s\n", ord.ShippingSnapshot.Street))
	}
	if ord.ShippingSnapshot.Street2 != "" {
		b.WriteString(fmt.Sprintf("住所2: %s\n", ord.ShippingSnapshot.Street2))
	}
	if ord.ShippingSnapshot.Country != "" {
		b.WriteString(fmt.Sprintf("国: %s\n", ord.ShippingSnapshot.Country))
	}
	b.WriteString("\n")
}

func writeOrderItemForCustomerMail(
	b *strings.Builder,
	index int,
	it orderdom.OrderItemSnapshot,
	modelsByID map[string]modeldom.ModelVariation,
	modelErrorsByID map[string]string,
	inventoriesByID map[string]inventorydom.Mint,
	inventoryErrorsByID map[string]string,
	productBlueprintsByID map[string]productblueprintdom.ProductBlueprint,
	productBlueprintErrorsByID map[string]string,
	tokenBlueprintsByID map[string]*tokenblueprintdom.TokenBlueprint,
	tokenBlueprintErrorsByID map[string]string,
	brandsByID map[string]branddom.Brand,
	brandErrorsByID map[string]string,
	companiesByID map[string]companydom.Company,
	companyErrorsByID map[string]string,
) {
	if b == nil {
		return
	}

	inventoryID := it.InventoryID
	modelID := it.ModelID

	b.WriteString(fmt.Sprintf("%d. 商品\n", index))

	if inventoryID != "" {
		if inv, ok := inventoriesByID[inventoryID]; ok {
			writeProductBlueprintForCustomerMail(
				b,
				inv,
				productBlueprintsByID,
				productBlueprintErrorsByID,
				brandsByID,
				brandErrorsByID,
				companiesByID,
				companyErrorsByID,
			)

			writeTokenBlueprintForCustomerMail(
				b,
				inv,
				tokenBlueprintsByID,
				tokenBlueprintErrorsByID,
				brandsByID,
				brandErrorsByID,
			)
		} else if _, ok := inventoryErrorsByID[inventoryID]; ok {
			b.WriteString("商品情報: 取得できませんでした\n")
		}
	}

	if modelID != "" {
		if model, ok := modelsByID[modelID]; ok {
			writeModelVariationForCustomerMail(b, model)
		} else if _, ok := modelErrorsByID[modelID]; ok {
			b.WriteString("型番情報: 取得できませんでした\n")
		}
	}

	b.WriteString(fmt.Sprintf("数量: %d点\n", it.Qty))
	b.WriteString(fmt.Sprintf("単価: %d円\n", it.Price))
	b.WriteString(fmt.Sprintf("小計: %d円\n", it.Price*it.Qty))
	b.WriteString("\n")
}

func writeProductBlueprintForCustomerMail(
	b *strings.Builder,
	inv inventorydom.Mint,
	productBlueprintsByID map[string]productblueprintdom.ProductBlueprint,
	productBlueprintErrorsByID map[string]string,
	brandsByID map[string]branddom.Brand,
	brandErrorsByID map[string]string,
	companiesByID map[string]companydom.Company,
	companyErrorsByID map[string]string,
) {
	if b == nil {
		return
	}

	productBlueprintID := inv.ProductBlueprintID
	if productBlueprintID == "" {
		return
	}

	pb, ok := productBlueprintsByID[productBlueprintID]
	if !ok {
		if _, exists := productBlueprintErrorsByID[productBlueprintID]; exists {
			b.WriteString("商品情報: 取得できませんでした\n")
		}
		return
	}

	if pb.ProductName != "" {
		b.WriteString(fmt.Sprintf("商品名: %s\n", pb.ProductName))
	}

	productBrandID := pb.BrandID
	if productBrandID != "" {
		if brand, ok := brandsByID[productBrandID]; ok && brand.Name != "" {
			b.WriteString(fmt.Sprintf("ブランド名: %s\n", brand.Name))
		} else if _, exists := brandErrorsByID[productBrandID]; exists {
			b.WriteString("ブランド名: 取得できませんでした\n")
		}
	}

	productCompanyID := pb.CompanyID
	if productCompanyID != "" {
		if company, ok := companiesByID[productCompanyID]; ok && company.Name != "" {
			b.WriteString(fmt.Sprintf("会社名: %s\n", company.Name))
		} else if _, exists := companyErrorsByID[productCompanyID]; exists {
			b.WriteString("会社名: 取得できませんでした\n")
		}
	}
}

func writeTokenBlueprintForCustomerMail(
	b *strings.Builder,
	inv inventorydom.Mint,
	tokenBlueprintsByID map[string]*tokenblueprintdom.TokenBlueprint,
	tokenBlueprintErrorsByID map[string]string,
	brandsByID map[string]branddom.Brand,
	brandErrorsByID map[string]string,
) {
	if b == nil {
		return
	}

	tokenBlueprintID := inv.TokenBlueprintID
	if tokenBlueprintID == "" {
		return
	}

	tb, ok := tokenBlueprintsByID[tokenBlueprintID]
	if !ok || tb == nil {
		if _, exists := tokenBlueprintErrorsByID[tokenBlueprintID]; exists {
			b.WriteString("トークン情報: 取得できませんでした\n")
		}
		return
	}

	if tb.Name != "" {
		b.WriteString(fmt.Sprintf("トークン名: %s\n", tb.Name))
	}

	tokenBrandID := tb.BrandID
	if tokenBrandID != "" {
		if brand, ok := brandsByID[tokenBrandID]; ok && brand.Name != "" {
			b.WriteString(fmt.Sprintf("トークンブランド名: %s\n", brand.Name))
		} else if _, exists := brandErrorsByID[tokenBrandID]; exists {
			b.WriteString("トークンブランド名: 取得できませんでした\n")
		}
	}
}

func writeModelVariationForCustomerMail(
	b *strings.Builder,
	model modeldom.ModelVariation,
) {
	if b == nil || model == nil {
		return
	}

	switch v := model.(type) {
	case modeldom.ApparelModelVariation:
		writeApparelModelVariationForCustomerMail(b, v)
	case *modeldom.ApparelModelVariation:
		if v != nil {
			writeApparelModelVariationForCustomerMail(b, *v)
		}
	case modeldom.AlcoholModelVariation:
		writeAlcoholModelVariationForCustomerMail(b, v)
	case *modeldom.AlcoholModelVariation:
		if v != nil {
			writeAlcoholModelVariationForCustomerMail(b, *v)
		}
	default:
		if model.GetModelNumber() != "" {
			b.WriteString(fmt.Sprintf("型番: %s\n", model.GetModelNumber()))
		}
	}
}

func writeApparelModelVariationForCustomerMail(
	b *strings.Builder,
	model modeldom.ApparelModelVariation,
) {
	if b == nil {
		return
	}

	if model.ModelNumber != "" {
		b.WriteString(fmt.Sprintf("型番: %s\n", model.ModelNumber))
	}
	if model.Size != "" {
		b.WriteString(fmt.Sprintf("サイズ: %s\n", model.Size))
	}
	if model.Color.Name != "" {
		b.WriteString(fmt.Sprintf("カラー: %s\n", model.Color.Name))
	}
	if model.Color.RGB >= 0 {
		b.WriteString(fmt.Sprintf("カラーRGB: %d\n", model.Color.RGB))
	}
	writeMeasurementsForCustomerMail(b, model.Measurements)
}

func writeAlcoholModelVariationForCustomerMail(
	b *strings.Builder,
	model modeldom.AlcoholModelVariation,
) {
	if b == nil {
		return
	}

	if model.ModelNumber != "" {
		b.WriteString(fmt.Sprintf("型番: %s\n", model.ModelNumber))
	}

	if model.Volume.Value > 0 || model.Volume.Unit != "" {
		b.WriteString("容量: ")
		if model.Volume.Value > 0 {
			b.WriteString(fmt.Sprintf("%d", model.Volume.Value))
		}
		if model.Volume.Unit != "" {
			b.WriteString(model.Volume.Unit)
		}
		b.WriteString("\n")
	}
}

func writeMeasurementsForCustomerMail(
	b *strings.Builder,
	measurements modeldom.Measurements,
) {
	if b == nil || len(measurements) == 0 {
		return
	}

	keys := make([]string, 0, len(measurements))
	for key := range measurements {
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return
	}

	sort.Strings(keys)

	b.WriteString("採寸:\n")
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("  %s: %v\n", measurementLabel(key), measurements[key]))
	}
}

func measurementLabel(key string) string {
	switch key {
	case "height":
		return "高さ"
	case "width":
		return "幅"
	case "depth":
		return "奥行き"
	case "length":
		return "長さ"
	case "shoulderWidth":
		return "肩幅"
	case "bodyWidth":
		return "身幅"
	case "sleeveLength":
		return "袖丈"
	case "bodyLength":
		return "着丈"
	case "waist":
		return "ウエスト"
	case "hip":
		return "ヒップ"
	case "rise":
		return "股上"
	case "inseam":
		return "股下"
	case "hemWidth":
		return "裾幅"
	default:
		return key
	}
}
