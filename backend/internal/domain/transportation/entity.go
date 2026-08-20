// backend/internal/domain/transportation/entity.go
package transportation

import (
	"errors"
	"sort"
	"time"
)

type Region string
type PrefectureCode string

const (
	PrefectureCount           = 47
	MaxCompanyIDLength        = 128
	MaxIslandCodeLength       = 128
	MinRateAmount       int64 = 0

	RegionHokkaido      Region = "hokkaido"
	RegionTohoku        Region = "tohoku"
	RegionKanto         Region = "kanto"
	RegionChubu         Region = "chubu"
	RegionKinki         Region = "kinki"
	RegionChugoku       Region = "chugoku"
	RegionShikoku       Region = "shikoku"
	RegionKyushuOkinawa Region = "kyushu_okinawa"

	PrefectureHokkaido  PrefectureCode = "01"
	PrefectureAomori    PrefectureCode = "02"
	PrefectureIwate     PrefectureCode = "03"
	PrefectureMiyagi    PrefectureCode = "04"
	PrefectureAkita     PrefectureCode = "05"
	PrefectureYamagata  PrefectureCode = "06"
	PrefectureFukushima PrefectureCode = "07"
	PrefectureIbaraki   PrefectureCode = "08"
	PrefectureTochigi   PrefectureCode = "09"
	PrefectureGunma     PrefectureCode = "10"
	PrefectureSaitama   PrefectureCode = "11"
	PrefectureChiba     PrefectureCode = "12"
	PrefectureTokyo     PrefectureCode = "13"
	PrefectureKanagawa  PrefectureCode = "14"
	PrefectureNiigata   PrefectureCode = "15"
	PrefectureToyama    PrefectureCode = "16"
	PrefectureIshikawa  PrefectureCode = "17"
	PrefectureFukui     PrefectureCode = "18"
	PrefectureYamanashi PrefectureCode = "19"
	PrefectureNagano    PrefectureCode = "20"
	PrefectureGifu      PrefectureCode = "21"
	PrefectureShizuoka  PrefectureCode = "22"
	PrefectureAichi     PrefectureCode = "23"
	PrefectureMie       PrefectureCode = "24"
	PrefectureShiga     PrefectureCode = "25"
	PrefectureKyoto     PrefectureCode = "26"
	PrefectureOsaka     PrefectureCode = "27"
	PrefectureHyogo     PrefectureCode = "28"
	PrefectureNara      PrefectureCode = "29"
	PrefectureWakayama  PrefectureCode = "30"
	PrefectureTottori   PrefectureCode = "31"
	PrefectureShimane   PrefectureCode = "32"
	PrefectureOkayama   PrefectureCode = "33"
	PrefectureHiroshima PrefectureCode = "34"
	PrefectureYamaguchi PrefectureCode = "35"
	PrefectureTokushima PrefectureCode = "36"
	PrefectureKagawa    PrefectureCode = "37"
	PrefectureEhime     PrefectureCode = "38"
	PrefectureKochi     PrefectureCode = "39"
	PrefectureFukuoka   PrefectureCode = "40"
	PrefectureSaga      PrefectureCode = "41"
	PrefectureNagasaki  PrefectureCode = "42"
	PrefectureKumamoto  PrefectureCode = "43"
	PrefectureOita      PrefectureCode = "44"
	PrefectureMiyazaki  PrefectureCode = "45"
	PrefectureKagoshima PrefectureCode = "46"
	PrefectureOkinawa   PrefectureCode = "47"
)

var (
	ErrInvalidCompanyID          = errors.New("transportation: invalid companyId")
	ErrInvalidRegion             = errors.New("transportation: invalid region")
	ErrInvalidPrefectureCode     = errors.New("transportation: invalid prefectureCode")
	ErrDuplicatePrefectureRate   = errors.New("transportation: duplicate prefectureRate")
	ErrIncompletePrefectureRates = errors.New("transportation: incomplete prefectureRates")
	ErrInvalidIslandCode         = errors.New("transportation: invalid islandCode")
	ErrDuplicateIslandRate       = errors.New("transportation: duplicate islandRate")
	ErrInvalidRateAmount         = errors.New("transportation: invalid rate amount")
	ErrInvalidCreatedAt          = errors.New("transportation: invalid createdAt")
	ErrInvalidUpdatedAt          = errors.New("transportation: invalid updatedAt")
	ErrPrefectureRateNotFound    = errors.New("transportation: prefectureRate not found")
)

type PrefectureGroup struct {
	Region          Region           `json:"region"`
	PrefectureCodes []PrefectureCode `json:"prefectureCodes"`
}

var prefectureGroups = []PrefectureGroup{
	{Region: RegionHokkaido, PrefectureCodes: []PrefectureCode{
		PrefectureHokkaido,
	}},
	{Region: RegionTohoku, PrefectureCodes: []PrefectureCode{
		PrefectureAomori, PrefectureIwate, PrefectureMiyagi, PrefectureAkita, PrefectureYamagata, PrefectureFukushima,
	}},
	{Region: RegionKanto, PrefectureCodes: []PrefectureCode{
		PrefectureIbaraki, PrefectureTochigi, PrefectureGunma, PrefectureSaitama, PrefectureChiba, PrefectureTokyo, PrefectureKanagawa,
	}},
	{Region: RegionChubu, PrefectureCodes: []PrefectureCode{
		PrefectureNiigata, PrefectureToyama, PrefectureIshikawa, PrefectureFukui, PrefectureYamanashi, PrefectureNagano, PrefectureGifu, PrefectureShizuoka, PrefectureAichi,
	}},
	{Region: RegionKinki, PrefectureCodes: []PrefectureCode{
		PrefectureMie, PrefectureShiga, PrefectureKyoto, PrefectureOsaka, PrefectureHyogo, PrefectureNara, PrefectureWakayama,
	}},
	{Region: RegionChugoku, PrefectureCodes: []PrefectureCode{
		PrefectureTottori, PrefectureShimane, PrefectureOkayama, PrefectureHiroshima, PrefectureYamaguchi,
	}},
	{Region: RegionShikoku, PrefectureCodes: []PrefectureCode{
		PrefectureTokushima, PrefectureKagawa, PrefectureEhime, PrefectureKochi,
	}},
	{Region: RegionKyushuOkinawa, PrefectureCodes: []PrefectureCode{
		PrefectureFukuoka, PrefectureSaga, PrefectureNagasaki, PrefectureKumamoto, PrefectureOita, PrefectureMiyazaki, PrefectureKagoshima, PrefectureOkinawa,
	}},
}

var allowedRegions = func() map[Region]struct{} {
	regions := make(map[Region]struct{}, len(prefectureGroups))
	for _, group := range prefectureGroups {
		regions[group.Region] = struct{}{}
	}
	return regions
}()

var prefectureRegionMap = func() map[PrefectureCode]Region {
	result := make(map[PrefectureCode]Region, PrefectureCount)
	for _, group := range prefectureGroups {
		for _, code := range group.PrefectureCodes {
			result[code] = group.Region
		}
	}
	return result
}()

var allowedPrefectureCodes = func() map[PrefectureCode]struct{} {
	codes := make(map[PrefectureCode]struct{}, PrefectureCount)
	for code := range prefectureRegionMap {
		codes[code] = struct{}{}
	}
	return codes
}()

type PrefectureRate struct {
	PrefectureCode PrefectureCode `json:"prefectureCode"`
	Amount         int64          `json:"amount"`
}

type IslandRate struct {
	IslandCode     string         `json:"islandCode"`
	PrefectureCode PrefectureCode `json:"prefectureCode"`
	Amount         int64          `json:"amount"`
}

type TransportationFeeSetting struct {
	CompanyID       string           `json:"companyId"`
	PrefectureRates []PrefectureRate `json:"prefectureRates"`
	IslandRates     []IslandRate     `json:"islandRates"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

func Regions() []Region {
	regions := make([]Region, 0, len(prefectureGroups))
	for _, group := range prefectureGroups {
		regions = append(regions, group.Region)
	}
	return regions
}

func PrefectureGroups() []PrefectureGroup {
	groups := make([]PrefectureGroup, len(prefectureGroups))
	for i, group := range prefectureGroups {
		groups[i] = PrefectureGroup{Region: group.Region, PrefectureCodes: clonePrefectureCodes(group.PrefectureCodes)}
	}
	return groups
}

func PrefectureCodes() []PrefectureCode {
	codes := make([]PrefectureCode, 0, PrefectureCount)
	for _, group := range prefectureGroups {
		codes = append(codes, group.PrefectureCodes...)
	}
	return codes
}

func PrefectureCodesByRegion(region Region) ([]PrefectureCode, error) {
	if !IsValidRegion(region) {
		return nil, ErrInvalidRegion
	}
	for _, group := range prefectureGroups {
		if group.Region == region {
			return clonePrefectureCodes(group.PrefectureCodes), nil
		}
	}
	return nil, ErrInvalidRegion
}

func RegionByPrefectureCode(code PrefectureCode) (Region, error) {
	region, ok := prefectureRegionMap[code]
	if !ok {
		return "", ErrInvalidPrefectureCode
	}
	return region, nil
}

func ParsePrefectureCode(code string) (PrefectureCode, error) {
	prefectureCode := PrefectureCode(code)
	if !IsValidPrefectureCode(prefectureCode) {
		return "", ErrInvalidPrefectureCode
	}
	return prefectureCode, nil
}

func IsValidRegion(region Region) bool {
	_, ok := allowedRegions[region]
	return ok
}

func IsValidPrefectureCode(code PrefectureCode) bool {
	_, ok := allowedPrefectureCodes[code]
	return ok
}

func New(companyID string, prefectureRates []PrefectureRate, islandRates []IslandRate, createdAt time.Time, updatedAt time.Time) (TransportationFeeSetting, error) {
	setting := TransportationFeeSetting{CompanyID: companyID, PrefectureRates: clonePrefectureRates(prefectureRates), IslandRates: cloneIslandRates(islandRates), CreatedAt: createdAt.UTC(), UpdatedAt: updatedAt.UTC()}
	setting.normalizeRateOrder()
	if err := setting.Validate(); err != nil {
		return TransportationFeeSetting{}, err
	}
	return setting, nil
}

func NewWithNow(companyID string, prefectureRates []PrefectureRate, islandRates []IslandRate, now time.Time) (TransportationFeeSetting, error) {
	now = now.UTC()
	return New(companyID, prefectureRates, islandRates, now, now)
}

func (s *TransportationFeeSetting) UpdateRates(prefectureRates []PrefectureRate, islandRates []IslandRate, now time.Time) error {
	if s == nil {
		return ErrInvalidCompanyID
	}
	next := TransportationFeeSetting{CompanyID: s.CompanyID, PrefectureRates: clonePrefectureRates(prefectureRates), IslandRates: cloneIslandRates(islandRates), CreatedAt: s.CreatedAt, UpdatedAt: now.UTC()}
	next.normalizeRateOrder()
	if err := next.Validate(); err != nil {
		return err
	}
	*s = next
	return nil
}

func (s TransportationFeeSetting) ResolveFee(prefectureCode PrefectureCode, islandCode string) (int64, error) {
	if !IsValidPrefectureCode(prefectureCode) {
		return 0, ErrInvalidPrefectureCode
	}
	if islandCode != "" {
		if err := validateIslandCode(islandCode); err != nil {
			return 0, err
		}
		for _, rate := range s.IslandRates {
			if rate.PrefectureCode == prefectureCode && rate.IslandCode == islandCode {
				return rate.Amount, nil
			}
		}
	}
	for _, rate := range s.PrefectureRates {
		if rate.PrefectureCode == prefectureCode {
			return rate.Amount, nil
		}
	}
	return 0, ErrPrefectureRateNotFound
}

func (s TransportationFeeSetting) Validate() error {
	if err := validateCompanyID(s.CompanyID); err != nil {
		return err
	}
	if err := validatePrefectureRates(s.PrefectureRates); err != nil {
		return err
	}
	if err := validateIslandRates(s.IslandRates); err != nil {
		return err
	}
	return validateTimestamps(s.CreatedAt, s.UpdatedAt)
}

func validateCompanyID(companyID string) error {
	if companyID == "" || len([]rune(companyID)) > MaxCompanyIDLength {
		return ErrInvalidCompanyID
	}
	return nil
}

func validatePrefectureRates(rates []PrefectureRate) error {
	if len(rates) != PrefectureCount {
		return ErrIncompletePrefectureRates
	}
	seen := make(map[PrefectureCode]struct{}, PrefectureCount)
	for _, rate := range rates {
		if !IsValidPrefectureCode(rate.PrefectureCode) {
			return ErrInvalidPrefectureCode
		}
		if rate.Amount < MinRateAmount {
			return ErrInvalidRateAmount
		}
		if _, exists := seen[rate.PrefectureCode]; exists {
			return ErrDuplicatePrefectureRate
		}
		seen[rate.PrefectureCode] = struct{}{}
	}
	if len(seen) != PrefectureCount {
		return ErrIncompletePrefectureRates
	}
	return nil
}

func validateIslandRates(rates []IslandRate) error {
	seen := make(map[string]struct{}, len(rates))
	for _, rate := range rates {
		if !IsValidPrefectureCode(rate.PrefectureCode) {
			return ErrInvalidPrefectureCode
		}
		if err := validateIslandCode(rate.IslandCode); err != nil {
			return err
		}
		if rate.Amount < MinRateAmount {
			return ErrInvalidRateAmount
		}
		key := string(rate.PrefectureCode) + ":" + rate.IslandCode
		if _, exists := seen[key]; exists {
			return ErrDuplicateIslandRate
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateIslandCode(islandCode string) error {
	if islandCode == "" || len([]rune(islandCode)) > MaxIslandCodeLength {
		return ErrInvalidIslandCode
	}
	for i, r := range islandCode {
		isLowerAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		isHyphen := r == '-'
		if !isLowerAlpha && !isDigit && !isHyphen {
			return ErrInvalidIslandCode
		}
		if i == 0 && isHyphen {
			return ErrInvalidIslandCode
		}
	}
	if islandCode[len(islandCode)-1] == '-' {
		return ErrInvalidIslandCode
	}
	return nil
}

func validateTimestamps(createdAt time.Time, updatedAt time.Time) error {
	if createdAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if updatedAt.IsZero() || updatedAt.Before(createdAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

func clonePrefectureCodes(codes []PrefectureCode) []PrefectureCode {
	if len(codes) == 0 {
		return nil
	}
	cloned := make([]PrefectureCode, len(codes))
	copy(cloned, codes)
	return cloned
}

func clonePrefectureRates(rates []PrefectureRate) []PrefectureRate {
	if len(rates) == 0 {
		return nil
	}
	cloned := make([]PrefectureRate, len(rates))
	copy(cloned, rates)
	return cloned
}

func cloneIslandRates(rates []IslandRate) []IslandRate {
	if len(rates) == 0 {
		return nil
	}
	cloned := make([]IslandRate, len(rates))
	copy(cloned, rates)
	return cloned
}

func (s *TransportationFeeSetting) normalizeRateOrder() {
	sort.Slice(s.PrefectureRates, func(i, j int) bool {
		return s.PrefectureRates[i].PrefectureCode < s.PrefectureRates[j].PrefectureCode
	})
	sort.Slice(s.IslandRates, func(i, j int) bool {
		if s.IslandRates[i].PrefectureCode == s.IslandRates[j].PrefectureCode {
			return s.IslandRates[i].IslandCode < s.IslandRates[j].IslandCode
		}
		return s.IslandRates[i].PrefectureCode < s.IslandRates[j].PrefectureCode
	})
}
