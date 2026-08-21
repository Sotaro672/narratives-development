// backend/internal/domain/transportation/service.go
package transportation

import (
	"context"
	"errors"
)

type Carrier string

const (
	CarrierYamato Carrier = "yamato"
	CarrierSagawa Carrier = "sagawa"
	CarrierPost   Carrier = "post"
	CarrierCustom Carrier = "custom"

	DomesticCountryCode = "JP"
)

var (
	ErrInvalidCarrier = errors.New(
		"transportation: invalid carrier",
	)

	ErrInvalidPackage = errors.New(
		"transportation: invalid package",
	)

	ErrInvalidAddress = errors.New(
		"transportation: invalid address",
	)

	ErrUnsupportedCountry = errors.New(
		"transportation: unsupported country",
	)

	ErrCarrierRateNotConfigured = errors.New(
		"transportation: carrier rate not configured",
	)

	ErrServiceUnavailable = errors.New(
		"transportation: service unavailable",
	)
)

type Package struct {
	WeightGrams int
	WidthMM     int
	LengthMM    int
	HeightMM    int
}

func (p Package) Validate() error {
	if p.WeightGrams <= 0 ||
		p.WidthMM <= 0 ||
		p.LengthMM <= 0 ||
		p.HeightMM <= 0 {
		return ErrInvalidPackage
	}

	return nil
}

func (p Package) TotalSizeMM() int {
	return p.WidthMM +
		p.LengthMM +
		p.HeightMM
}

type Address struct {
	Country string
	ZipCode string
	State   string
	City    string

	IslandCode string
}

func (a Address) Validate() error {
	if a.Country == "" {
		return ErrInvalidAddress
	}

	if a.Country != DomesticCountryCode {
		return ErrUnsupportedCountry
	}

	if a.State == "" {
		return ErrInvalidAddress
	}

	if a.ZipCode == "" {
		return ErrInvalidAddress
	}

	if a.City == "" {
		return ErrInvalidAddress
	}

	return nil
}

type CalculateInput struct {
	Carrier Carrier

	Package Package

	Origin      Address
	Destination Address

	CompanyID string

	TransportationID string
}

type CarrierRateInput struct {
	Package Package

	OriginPrefectureCode      PrefectureCode
	DestinationPrefectureCode PrefectureCode

	DestinationIslandCode string
}

type Quote struct {
	Carrier Carrier

	Size   int
	Amount int64
}

type carrierCalculator interface {
	Calculate(
		input CarrierRateInput,
	) (Quote, error)
}

type Service struct {
	repo RepositoryPort

	calculators map[Carrier]carrierCalculator
}

func NewService(
	repo RepositoryPort,
) *Service {
	return &Service{
		repo: repo,

		calculators: map[Carrier]carrierCalculator{
			CarrierYamato: newYamatoRateCalculator(),

			CarrierSagawa: newSagawaRateCalculator(),

			CarrierPost: newPostRateCalculator(),
		},
	}
}

func IsValidCarrier(
	carrier Carrier,
) bool {
	switch carrier {
	case CarrierYamato,
		CarrierSagawa,
		CarrierPost,
		CarrierCustom:
		return true
	default:
		return false
	}
}

func (s *Service) Calculate(
	ctx context.Context,
	input CalculateInput,
) (Quote, error) {
	if s == nil {
		return Quote{},
			ErrServiceUnavailable
	}

	if !IsValidCarrier(
		input.Carrier,
	) {
		return Quote{},
			ErrInvalidCarrier
	}

	if err :=
		input.Package.Validate(); err != nil {
		return Quote{}, err
	}

	if err :=
		input.Origin.Validate(); err != nil {
		return Quote{}, err
	}

	if err :=
		input.Destination.Validate(); err != nil {
		return Quote{}, err
	}

	originPrefectureCode, err :=
		PrefectureCodeFromState(
			input.Origin.State,
		)
	if err != nil {
		return Quote{}, err
	}

	destinationPrefectureCode, err :=
		PrefectureCodeFromState(
			input.Destination.State,
		)
	if err != nil {
		return Quote{}, err
	}

	if input.Carrier == CarrierCustom {
		return s.calculateCustom(
			ctx,
			input,
			destinationPrefectureCode,
		)
	}

	return s.calculateCarrier(
		input,
		originPrefectureCode,
		destinationPrefectureCode,
	)
}

func (s *Service) calculateCarrier(
	input CalculateInput,
	originPrefectureCode PrefectureCode,
	destinationPrefectureCode PrefectureCode,
) (Quote, error) {
	calculator, ok :=
		s.calculators[input.Carrier]

	if !ok || calculator == nil {
		return Quote{},
			ErrCarrierRateNotConfigured
	}

	quote, err :=
		calculator.Calculate(
			CarrierRateInput{
				Package: input.Package,

				OriginPrefectureCode: originPrefectureCode,

				DestinationPrefectureCode: destinationPrefectureCode,

				DestinationIslandCode: input.Destination.IslandCode,
			},
		)
	if err != nil {
		return Quote{}, err
	}

	if quote.Carrier == "" {
		quote.Carrier =
			input.Carrier
	}

	if quote.Carrier != input.Carrier {
		return Quote{},
			ErrInvalidCarrier
	}

	if quote.Size <= 0 {
		return Quote{},
			ErrInvalidPackage
	}

	if quote.Amount < MinRateAmount {
		return Quote{},
			ErrInvalidRateAmount
	}

	return quote, nil
}

func (s *Service) calculateCustom(
	ctx context.Context,
	input CalculateInput,
	destinationPrefectureCode PrefectureCode,
) (Quote, error) {
	if s.repo == nil {
		return Quote{},
			ErrServiceUnavailable
	}

	if err :=
		validateCompanyID(
			input.CompanyID,
		); err != nil {
		return Quote{}, err
	}

	if err :=
		validateTransportationID(
			input.TransportationID,
		); err != nil {
		return Quote{}, err
	}

	setting, err :=
		s.repo.GetByID(
			ctx,
			input.TransportationID,
		)
	if err != nil {
		return Quote{}, err
	}

	if setting == nil {
		return Quote{},
			ErrNotFound
	}

	if setting.ID !=
		input.TransportationID {
		return Quote{},
			ErrNotFound
	}

	if setting.CompanyID !=
		input.CompanyID {
		return Quote{},
			ErrNotFound
	}

	amount, err :=
		setting.ResolveFee(
			destinationPrefectureCode,
			input.Destination.IslandCode,
		)
	if err != nil {
		return Quote{}, err
	}

	if amount < MinRateAmount {
		return Quote{},
			ErrInvalidRateAmount
	}

	return Quote{
		Carrier: CarrierCustom,
		Size:    0,
		Amount:  amount,
	}, nil
}

func (s *Service) registerCarrierCalculator(
	carrier Carrier,
	calculator carrierCalculator,
) error {
	if s == nil {
		return ErrServiceUnavailable
	}

	switch carrier {
	case CarrierYamato,
		CarrierSagawa,
		CarrierPost:
	default:
		return ErrInvalidCarrier
	}

	if calculator == nil {
		return ErrCarrierRateNotConfigured
	}

	if s.calculators == nil {
		s.calculators =
			make(
				map[Carrier]carrierCalculator,
			)
	}

	s.calculators[carrier] =
		calculator

	return nil
}

var prefectureCodeByState = map[string]PrefectureCode{
	"北海道":  PrefectureHokkaido,
	"青森県":  PrefectureAomori,
	"岩手県":  PrefectureIwate,
	"宮城県":  PrefectureMiyagi,
	"秋田県":  PrefectureAkita,
	"山形県":  PrefectureYamagata,
	"福島県":  PrefectureFukushima,
	"茨城県":  PrefectureIbaraki,
	"栃木県":  PrefectureTochigi,
	"群馬県":  PrefectureGunma,
	"埼玉県":  PrefectureSaitama,
	"千葉県":  PrefectureChiba,
	"東京都":  PrefectureTokyo,
	"神奈川県": PrefectureKanagawa,
	"新潟県":  PrefectureNiigata,
	"富山県":  PrefectureToyama,
	"石川県":  PrefectureIshikawa,
	"福井県":  PrefectureFukui,
	"山梨県":  PrefectureYamanashi,
	"長野県":  PrefectureNagano,
	"岐阜県":  PrefectureGifu,
	"静岡県":  PrefectureShizuoka,
	"愛知県":  PrefectureAichi,
	"三重県":  PrefectureMie,
	"滋賀県":  PrefectureShiga,
	"京都府":  PrefectureKyoto,
	"大阪府":  PrefectureOsaka,
	"兵庫県":  PrefectureHyogo,
	"奈良県":  PrefectureNara,
	"和歌山県": PrefectureWakayama,
	"鳥取県":  PrefectureTottori,
	"島根県":  PrefectureShimane,
	"岡山県":  PrefectureOkayama,
	"広島県":  PrefectureHiroshima,
	"山口県":  PrefectureYamaguchi,
	"徳島県":  PrefectureTokushima,
	"香川県":  PrefectureKagawa,
	"愛媛県":  PrefectureEhime,
	"高知県":  PrefectureKochi,
	"福岡県":  PrefectureFukuoka,
	"佐賀県":  PrefectureSaga,
	"長崎県":  PrefectureNagasaki,
	"熊本県":  PrefectureKumamoto,
	"大分県":  PrefectureOita,
	"宮崎県":  PrefectureMiyazaki,
	"鹿児島県": PrefectureKagoshima,
	"沖縄県":  PrefectureOkinawa,
}

func PrefectureCodeFromState(
	state string,
) (PrefectureCode, error) {
	if code, err :=
		ParsePrefectureCode(
			state,
		); err == nil {
		return code, nil
	}

	code, ok :=
		prefectureCodeByState[state]

	if !ok {
		return "",
			ErrInvalidPrefectureCode
	}

	return code, nil
}
