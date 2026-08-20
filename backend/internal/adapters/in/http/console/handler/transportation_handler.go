// backend/internal/adapters/in/http/console/handler/transportation_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	transportationdom "narratives/internal/domain/transportation"
)

const (
	transportationBasePath   = "/transportation"
	transportationMasterPath = "/transportation/master"
)

type TransportationHandler struct {
	uc *usecase.TransportationUsecase
}

func NewTransportationHandler(uc *usecase.TransportationUsecase) http.Handler {
	return &TransportationHandler{uc: uc}
}

type transportationPrefectureRateRequest struct {
	PrefectureCode string `json:"prefectureCode"`
	Amount         int64  `json:"amount"`
}

type transportationIslandRateRequest struct {
	IslandCode     string `json:"islandCode"`
	PrefectureCode string `json:"prefectureCode"`
	Amount         int64  `json:"amount"`
}

type transportationWriteRequest struct {
	Name            string                                `json:"name"`
	PrefectureRates []transportationPrefectureRateRequest `json:"prefectureRates"`
	IslandRates     []transportationIslandRateRequest     `json:"islandRates"`
}

type transportationMasterResponse struct {
	Regions []transportationRegionResponse `json:"regions"`
	Islands []transportationIslandResponse `json:"islands"`
}

type transportationRegionResponse struct {
	Region          transportationdom.Region           `json:"region"`
	PrefectureCodes []transportationdom.PrefectureCode `json:"prefectureCodes"`
	IslandCodes     []transportationdom.IslandCode     `json:"islandCodes"`
}

type transportationIslandResponse struct {
	IslandCode     transportationdom.IslandCode     `json:"islandCode"`
	PrefectureCode transportationdom.PrefectureCode `json:"prefectureCode"`
	DisplayName    string                           `json:"displayName"`
}

func (h *TransportationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	transportationID, hasTransportationID := transportationIDFromPath(path)

	switch {
	case r.Method == http.MethodGet && path == transportationMasterPath:
		h.master(w, r)
	case r.Method == http.MethodGet && path == transportationBasePath:
		h.list(w, r)
	case r.Method == http.MethodPost && path == transportationBasePath:
		h.create(w, r)
	case r.Method == http.MethodGet && hasTransportationID:
		h.get(w, r, transportationID)
	case r.Method == http.MethodPut && hasTransportationID:
		h.update(w, r, transportationID)
	default:
		writeNotFound(w)
	}
}

func transportationIDFromPath(path string) (string, bool) {
	prefix := transportationBasePath + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}

	id := strings.TrimPrefix(path, prefix)
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}

func (h *TransportationHandler) requireUsecase(w http.ResponseWriter) bool {
	if h != nil && h.uc != nil {
		return true
	}

	writeError(w, http.StatusServiceUnavailable, "transportation_usecase_not_initialized")
	return false
}

func (h *TransportationHandler) requireCompanyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	companyID, ok := middleware.CompanyID(r)
	if ok && companyID != "" {
		return companyID, true
	}

	writeError(w, http.StatusForbidden, "company_id_not_resolved")
	return "", false
}

// masterは地方ごとの都道府県コードと島嶼部のIslandCode一覧を返します。
// 都道府県・島嶼部情報は固定Domain masterであり、Firestoreから取得しません。
func (h *TransportationHandler) master(w http.ResponseWriter, _ *http.Request) {
	groups := transportationdom.RegionGroups()
	regions := make([]transportationRegionResponse, len(groups))

	for i, group := range groups {
		regions[i] = transportationRegionResponse{
			Region:          group.Region,
			PrefectureCodes: group.PrefectureCodes,
			IslandCodes:     group.IslandCodes,
		}
	}

	definitions := transportationdom.IslandDefinitions()
	islands := make([]transportationIslandResponse, len(definitions))

	for i, definition := range definitions {
		islands[i] = transportationIslandResponse{
			IslandCode:     definition.IslandCode,
			PrefectureCode: definition.PrefectureCode,
			DisplayName:    definition.DisplayName,
		}
	}

	writeJSON(w, http.StatusOK, transportationMasterResponse{
		Regions: regions,
		Islands: islands,
	})
}

// listは認証済みcompanyが所有する配送料金設定一覧を返します。
// companyに設定が存在しない場合は空配列を返します。
func (h *TransportationHandler) list(w http.ResponseWriter, r *http.Request) {
	if !h.requireUsecase(w) {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	settings, err := h.uc.ListByCompanyID(r.Context(), companyID)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	if settings == nil {
		settings = []transportationdom.TransportationFeeSetting{}
	}

	writeJSON(w, http.StatusOK, settings)
}

// getは認証済みcompanyが所有する指定Transportation IDの配送料金設定を返します。
// 対象が存在しない、または別companyが所有する場合は404を返します。
func (h *TransportationHandler) get(w http.ResponseWriter, r *http.Request, transportationID string) {
	if !h.requireUsecase(w) {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	setting, err := h.uc.GetByID(r.Context(), companyID, transportationID)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, setting)
}

// createは認証済みcompanyに新しい配送料金設定を作成します。
// 同一companyに複数のTransportationFeeSettingを作成できます。
// Transportation IDはUsecase側で採番します。
func (h *TransportationHandler) create(w http.ResponseWriter, r *http.Request) {
	if !h.requireUsecase(w) {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	request, ok := decodeTransportationWriteRequest(w, r)
	if !ok {
		return
	}

	prefectureRates, islandRates, err := transportationRequestToDomain(request)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	created, err := h.uc.Create(
		r.Context(),
		companyID,
		usecase.CreateTransportationFeeSettingInput{
			Name:            request.Name,
			PrefectureRates: prefectureRates,
			IslandRates:     islandRates,
		},
	)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// updateは認証済みcompanyが所有する指定Transportation IDの配送料金設定を完全置換します。
// Name、PrefectureRates、IslandRatesを更新します。
// ID、CompanyID、CreatedAtは変更しません。
// Updateはupsertではなく、設定が存在しない場合は404を返します。
func (h *TransportationHandler) update(w http.ResponseWriter, r *http.Request, transportationID string) {
	if !h.requireUsecase(w) {
		return
	}

	companyID, ok := h.requireCompanyID(w, r)
	if !ok {
		return
	}

	request, ok := decodeTransportationWriteRequest(w, r)
	if !ok {
		return
	}

	prefectureRates, islandRates, err := transportationRequestToDomain(request)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	updated, err := h.uc.Update(
		r.Context(),
		companyID,
		transportationID,
		usecase.UpdateTransportationFeeSettingInput{
			Name:            request.Name,
			PrefectureRates: prefectureRates,
			IslandRates:     islandRates,
		},
	)
	if err != nil {
		writeTransportationErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func decodeTransportationWriteRequest(w http.ResponseWriter, r *http.Request) (transportationWriteRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request transportationWriteRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return transportationWriteRequest{}, false
	}

	return request, true
}

func transportationRequestToDomain(request transportationWriteRequest) ([]transportationdom.PrefectureRate, []transportationdom.IslandRate, error) {
	prefectureRates := make([]transportationdom.PrefectureRate, len(request.PrefectureRates))

	for i, rate := range request.PrefectureRates {
		prefectureCode, err := transportationdom.ParsePrefectureCode(rate.PrefectureCode)
		if err != nil {
			return nil, nil, err
		}

		prefectureRates[i] = transportationdom.PrefectureRate{
			PrefectureCode: prefectureCode,
			Amount:         rate.Amount,
		}
	}

	islandRates := make([]transportationdom.IslandRate, len(request.IslandRates))

	for i, rate := range request.IslandRates {
		islandCode, err := transportationdom.ParseIslandCode(rate.IslandCode)
		if err != nil {
			return nil, nil, err
		}

		prefectureCode, err := transportationdom.ParsePrefectureCode(rate.PrefectureCode)
		if err != nil {
			return nil, nil, err
		}

		definition, err := transportationdom.IslandDefinitionByCode(islandCode)
		if err != nil {
			return nil, nil, err
		}

		if definition.PrefectureCode != prefectureCode {
			return nil, nil, transportationdom.ErrIslandPrefectureMismatch
		}

		islandRates[i] = transportationdom.IslandRate{
			IslandCode:     islandCode,
			PrefectureCode: prefectureCode,
			Amount:         rate.Amount,
		}
	}

	return prefectureRates, islandRates, nil
}

func writeTransportationErr(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, transportationdom.ErrInvalidID),
		errors.Is(err, transportationdom.ErrInvalidCompanyID),
		errors.Is(err, transportationdom.ErrInvalidName),
		errors.Is(err, transportationdom.ErrInvalidRegion),
		errors.Is(err, transportationdom.ErrInvalidPrefectureCode),
		errors.Is(err, transportationdom.ErrDuplicatePrefectureRate),
		errors.Is(err, transportationdom.ErrIncompletePrefectureRates),
		errors.Is(err, transportationdom.ErrInvalidIslandCode),
		errors.Is(err, transportationdom.ErrIslandPrefectureMismatch),
		errors.Is(err, transportationdom.ErrDuplicateIslandRate),
		errors.Is(err, transportationdom.ErrInvalidRateAmount),
		errors.Is(err, transportationdom.ErrInvalidCreatedAt),
		errors.Is(err, transportationdom.ErrInvalidUpdatedAt),
		errors.Is(err, transportationdom.ErrPrefectureRateNotFound):
		statusCode = http.StatusBadRequest

	case errors.Is(err, transportationdom.ErrNotFound):
		statusCode = http.StatusNotFound

	case errors.Is(err, transportationdom.ErrConflict):
		statusCode = http.StatusConflict
	}

	writeError(w, statusCode, err.Error())
}
