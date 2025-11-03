package product

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"context"
)

const (
	apiBaseURL = "https://api.example.com/products"
)

func fetchAPI(endpoint string, method string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func getAllProducts() ([]Product, error) {
	resp, err := fetchAPI(apiBaseURL, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}
	return products, nil
}

func getProductById(id string) (*Product, error) {
	resp, err := fetchAPI(fmt.Sprintf("%s/%s", apiBaseURL, id), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, err
	}
	return &product, nil
}

func getProductsByProductionId(productionId string) ([]Product, error) {
	resp, err := fetchAPI(fmt.Sprintf("%s/production/%s", apiBaseURL, productionId), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}
	return products, nil
}

func getProductsByModelId(modelId string) ([]Product, error) {
	resp, err := fetchAPI(fmt.Sprintf("%s/model/%s", apiBaseURL, modelId), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}
	return products, nil
}

func getProductByTokenId(tokenId string) (*Product, error) {
	resp, err := fetchAPI(fmt.Sprintf("%s/token/%s", apiBaseURL, tokenId), http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, err
	}
	return &product, nil
}

func createProduct(product Product) (*Product, error) {
	resp, err := fetchAPI(apiBaseURL, http.MethodPost, product)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createdProduct Product
	if err := json.NewDecoder(resp.Body).Decode(&createdProduct); err != nil {
		return nil, err
	}
	return &createdProduct, nil
}

func updateProduct(id string, product Product) (*Product, error) {
	resp, err := fetchAPI(fmt.Sprintf("%s/%s", apiBaseURL, id), http.MethodPut, product)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updatedProduct Product
	if err := json.NewDecoder(resp.Body).Decode(&updatedProduct); err != nil {
		return nil, err
	}
	return &updatedProduct, nil
}

func updateProductInspection(id string, inspection Inspection) error {
	_, err := fetchAPI(fmt.Sprintf("%s/%s/inspection", apiBaseURL, id), http.MethodPut, inspection)
	return err
}

func connectTokenToProduct(productId string, tokenId string) error {
	_, err := fetchAPI(fmt.Sprintf("%s/%s/token/%s", apiBaseURL, productId, tokenId), http.MethodPost, nil)
	return err
}

func deleteProduct(id string) error {
	_, err := fetchAPI(fmt.Sprintf("%s/%s", apiBaseURL, id), http.MethodDelete, nil)
	return err
}

// ===============================
// Create/Update inputs (contracts)
// ===============================

// CreateProductInput - 作成入力（id/updatedAtはリポジトリ側で採番/付与してよい）
type CreateProductInput struct {
	ModelID          string            `json:"modelId"`
	ProductionID     string            `json:"productionId"`
	InspectionResult *InspectionResult `json:"inspectionResult,omitempty"` // nilなら notYet
	ConnectedToken   *string           `json:"connectedToken,omitempty"`

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	PrintedBy   *string    `json:"printedBy,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`
	InspectedBy *string    `json:"inspectedBy,omitempty"`

	UpdatedBy string `json:"updatedBy"`
}

// UpdateProductInput - 部分更新（nilは未更新）
type UpdateProductInput struct {
	ModelID          *string           `json:"modelId,omitempty"`
	ProductionID     *string           `json:"productionId,omitempty"`
	InspectionResult *InspectionResult `json:"inspectionResult,omitempty"`
	ConnectedToken   *string           `json:"connectedToken,omitempty"` // 空文字→nilにする等の扱いは実装側判断

	PrintedAt   *time.Time `json:"printedAt,omitempty"`
	PrintedBy   *string    `json:"printedBy,omitempty"`
	InspectedAt *time.Time `json:"inspectedAt,omitempty"`
	InspectedBy *string    `json:"inspectedBy,omitempty"`

	UpdatedBy *string `json:"updatedBy,omitempty"`
}

// UpdateInspectionInput - 検査更新（専用オペレーションが必要な場合）
type UpdateInspectionInput struct {
	InspectionResult InspectionResult `json:"inspectionResult"`
	InspectedBy      string           `json:"inspectedBy"`
	InspectedAt      *time.Time       `json:"inspectedAt,omitempty"`
}

// ConnectTokenInput - トークン接続/切断 (TokenID=nil で切断)
type ConnectTokenInput struct {
	TokenID *string `json:"tokenId"`
}

// ===============================
// Query contracts
// ===============================

// Filter - 検索条件
type Filter struct {
	ID           string
	ModelID      string
	ProductionID string

	InspectionResults []InspectionResult
	HasToken          *bool // nil=全件, true=トークンあり, false=なし
	TokenID           string

	PrintedFrom   *time.Time
	PrintedTo     *time.Time
	InspectedFrom *time.Time
	InspectedTo   *time.Time
	UpdatedFrom   *time.Time
	UpdatedTo     *time.Time
}

// Sort - 並び順
type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByUpdatedAt    SortColumn = "updatedAt"
	SortByPrintedAt    SortColumn = "printedAt"
	SortByInspectedAt  SortColumn = "inspectedAt"
	SortByModelID      SortColumn = "modelId"
	SortByProductionID SortColumn = "productionId"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Page - ページ指定
type Page struct {
	Number  int
	PerPage int
}

// PageResult - ページ結果
type PageResult struct {
	Items      []Product
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ===============================
// Repository Port
// ===============================

type RepositoryPort interface {
	// 取得
	GetByID(ctx context.Context, id string) (*Product, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 作成/更新/削除
	Create(ctx context.Context, in CreateProductInput) (*Product, error)
	Update(ctx context.Context, id string, in UpdateProductInput) (*Product, error)
	Delete(ctx context.Context, id string) error

	// ドメイン特化オペレーション（必要なら）
	UpdateInspection(ctx context.Context, id string, in UpdateInspectionInput) (*Product, error)
	ConnectToken(ctx context.Context, id string, in ConnectTokenInput) (*Product, error)
}

// 共通エラー
var (
	ErrNotFound = errors.New("product: not found")
	ErrConflict = errors.New("product: conflict")
)

// Additional filtering and statistics functions can be added here.

// keep: prevent gopls unusedfunc diagnostics for planned functions
// These functions are intentionally kept for future use by adapters.
var (
	_ = getAllProducts
	_ = getProductById
	_ = getProductsByProductionId
	_ = getProductsByModelId
	_ = getProductByTokenId
	_ = createProduct
	_ = updateProduct
	_ = updateProductInspection
	_ = connectTokenToProduct
	_ = deleteProduct
)
