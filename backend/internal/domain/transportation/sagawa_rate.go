// backend/internal/domain/transportation/sagawa_rate.go
package transportation

import "errors"

const (
	SagawaPublicRateVersion = "public-verified-2026-08-21"

	sagawaMaxWeightGrams = 30000
	sagawaMaxTotalSizeMM = 1600
)

var (
	ErrSagawaPackageTooLarge = errors.New(
		"transportation: sagawa package too large",
	)

	ErrSagawaRateNotFound = errors.New(
		"transportation: sagawa rate not found",
	)

	ErrSagawaIslandSurchargeRequired = errors.New(
		"transportation: sagawa island surcharge required",
	)
)

type sagawaZone uint8

const (
	sagawaZoneHokkaido sagawaZone = iota + 1
	sagawaZoneNorthTohoku
	sagawaZoneSouthTohoku
	sagawaZoneKanto
	sagawaZoneShinetsu
	sagawaZoneTokai
	sagawaZoneHokuriku
	sagawaZoneKansai
	sagawaZoneChugoku
	sagawaZoneShikoku
	sagawaZoneNorthKyushu
	sagawaZoneSouthKyushu
	sagawaZoneOkinawa
)

type sagawaRateBand uint8

const (
	sagawaRateBandA sagawaRateBand = iota + 1
	sagawaRateBandB
	sagawaRateBandC
	sagawaRateBandD
	sagawaRateBandE
	sagawaRateBandF
	sagawaRateBandG
	sagawaRateBandH
	sagawaRateBandI
	sagawaRateBandJ
	sagawaRateBandK
)

type sagawaZonePair struct {
	From sagawaZone
	To   sagawaZone
}

type sagawaRateCalculator struct{}

func newSagawaRateCalculator() carrierCalculator {
	return &sagawaRateCalculator{}
}

func (c *sagawaRateCalculator) Calculate(
	input CarrierRateInput,
) (Quote, error) {
	if err := input.Package.Validate(); err != nil {
		return Quote{}, err
	}

	if input.DestinationIslandCode != "" {
		return Quote{}, ErrSagawaIslandSurchargeRequired
	}

	size, err := resolveSagawaSize(input.Package)
	if err != nil {
		return Quote{}, err
	}

	originZone, err := sagawaZoneByPrefectureCode(
		input.OriginPrefectureCode,
	)
	if err != nil {
		return Quote{}, err
	}

	destinationZone, err := sagawaZoneByPrefectureCode(
		input.DestinationPrefectureCode,
	)
	if err != nil {
		return Quote{}, err
	}

	amount, err := resolveSagawaRate(
		originZone,
		destinationZone,
		size,
	)
	if err != nil {
		return Quote{}, err
	}

	return Quote{
		Carrier: CarrierSagawa,
		Size:    size,
		Amount:  amount,
	}, nil
}

func resolveSagawaSize(pkg Package) (int, error) {
	if err := pkg.Validate(); err != nil {
		return 0, err
	}

	if pkg.WeightGrams > sagawaMaxWeightGrams {
		return 0, ErrSagawaPackageTooLarge
	}

	totalSizeMM := pkg.TotalSizeMM()
	if totalSizeMM > sagawaMaxTotalSizeMM {
		return 0, ErrSagawaPackageTooLarge
	}

	dimensionSize, err := sagawaSizeByDimension(totalSizeMM)
	if err != nil {
		return 0, err
	}

	weightSize, err := sagawaSizeByWeight(pkg.WeightGrams)
	if err != nil {
		return 0, err
	}

	if weightSize > dimensionSize {
		return weightSize, nil
	}

	return dimensionSize, nil
}

func sagawaSizeByDimension(totalSizeMM int) (int, error) {
	switch {
	case totalSizeMM <= 0:
		return 0, ErrInvalidPackage
	case totalSizeMM <= 600:
		return 60, nil
	case totalSizeMM <= 800:
		return 80, nil
	case totalSizeMM <= 1000:
		return 100, nil
	case totalSizeMM <= 1400:
		return 140, nil
	case totalSizeMM <= 1600:
		return 160, nil
	default:
		return 0, ErrSagawaPackageTooLarge
	}
}

func sagawaSizeByWeight(weightGrams int) (int, error) {
	switch {
	case weightGrams <= 0:
		return 0, ErrInvalidPackage
	case weightGrams <= 2000:
		return 60, nil
	case weightGrams <= 5000:
		return 80, nil
	case weightGrams <= 10000:
		return 100, nil
	case weightGrams <= 20000:
		return 140, nil
	case weightGrams <= 30000:
		return 160, nil
	default:
		return 0, ErrSagawaPackageTooLarge
	}
}

func sagawaRateIndexBySize(size int) (int, error) {
	switch size {
	case 60:
		return 0, nil
	case 80:
		return 1, nil
	case 100:
		return 2, nil
	case 140:
		return 3, nil
	case 160:
		return 4, nil
	default:
		return 0, ErrInvalidPackage
	}
}

func resolveSagawaRate(
	originZone sagawaZone,
	destinationZone sagawaZone,
	size int,
) (int64, error) {
	index, err := sagawaRateIndexBySize(size)
	if err != nil {
		return 0, err
	}

	if originZone == sagawaZoneOkinawa ||
		destinationZone == sagawaZoneOkinawa {
		rates, err := sagawaOkinawaRouteRates(
			originZone,
			destinationZone,
		)
		if err != nil {
			return 0, err
		}

		return rates[index], nil
	}

	pair := normalizeSagawaZonePair(
		originZone,
		destinationZone,
	)

	band, ok := sagawaMainlandRateBandByPair[pair]
	if !ok {
		return 0, ErrSagawaRateNotFound
	}

	rates, ok := sagawaRatesByBand[band]
	if !ok {
		return 0, ErrSagawaRateNotFound
	}

	return rates[index], nil
}

func normalizeSagawaZonePair(
	a sagawaZone,
	b sagawaZone,
) sagawaZonePair {
	if a <= b {
		return sagawaZonePair{
			From: a,
			To:   b,
		}
	}

	return sagawaZonePair{
		From: b,
		To:   a,
	}
}

func sagawaOkinawaRouteRates(
	originZone sagawaZone,
	destinationZone sagawaZone,
) ([5]int64, error) {
	if originZone == sagawaZoneOkinawa &&
		destinationZone == sagawaZoneOkinawa {
		return sagawaOkinawaSameZoneRates, nil
	}

	mainlandZone := originZone
	if mainlandZone == sagawaZoneOkinawa {
		mainlandZone = destinationZone
	}

	rates, ok := sagawaOkinawaRatesByMainlandZone[mainlandZone]
	if !ok {
		return [5]int64{}, ErrSagawaRateNotFound
	}

	return rates, nil
}

func sagawaZoneByPrefectureCode(
	code PrefectureCode,
) (sagawaZone, error) {
	switch code {
	case PrefectureHokkaido:
		return sagawaZoneHokkaido, nil

	case PrefectureAomori,
		PrefectureIwate,
		PrefectureAkita:
		return sagawaZoneNorthTohoku, nil

	case PrefectureMiyagi,
		PrefectureYamagata,
		PrefectureFukushima:
		return sagawaZoneSouthTohoku, nil

	case PrefectureIbaraki,
		PrefectureTochigi,
		PrefectureGunma,
		PrefectureSaitama,
		PrefectureChiba,
		PrefectureTokyo,
		PrefectureKanagawa,
		PrefectureYamanashi:
		return sagawaZoneKanto, nil

	case PrefectureNiigata,
		PrefectureNagano:
		return sagawaZoneShinetsu, nil

	case PrefectureGifu,
		PrefectureShizuoka,
		PrefectureAichi,
		PrefectureMie:
		return sagawaZoneTokai, nil

	case PrefectureToyama,
		PrefectureIshikawa,
		PrefectureFukui:
		return sagawaZoneHokuriku, nil

	case PrefectureShiga,
		PrefectureKyoto,
		PrefectureOsaka,
		PrefectureHyogo,
		PrefectureNara,
		PrefectureWakayama:
		return sagawaZoneKansai, nil

	case PrefectureTottori,
		PrefectureShimane,
		PrefectureOkayama,
		PrefectureHiroshima,
		PrefectureYamaguchi:
		return sagawaZoneChugoku, nil

	case PrefectureTokushima,
		PrefectureKagawa,
		PrefectureEhime,
		PrefectureKochi:
		return sagawaZoneShikoku, nil

	case PrefectureFukuoka,
		PrefectureSaga,
		PrefectureNagasaki,
		PrefectureOita:
		return sagawaZoneNorthKyushu, nil

	case PrefectureKumamoto,
		PrefectureMiyazaki,
		PrefectureKagoshima:
		return sagawaZoneSouthKyushu, nil

	case PrefectureOkinawa:
		return sagawaZoneOkinawa, nil

	default:
		return 0, ErrInvalidPrefectureCode
	}
}

// index:
// 0 = 60
// 1 = 80
// 2 = 100
// 3 = 140
// 4 = 160

var sagawaRatesByBand = map[sagawaRateBand][5]int64{
	sagawaRateBandA: {
		910,
		1220,
		1520,
		2180,
		2440,
	},

	sagawaRateBandB: {
		1040,
		1340,
		1630,
		2310,
		2570,
	},

	sagawaRateBandC: {
		1180,
		1470,
		1740,
		2440,
		2700,
	},

	sagawaRateBandD: {
		1300,
		1590,
		1880,
		2570,
		2830,
	},

	sagawaRateBandE: {
		1440,
		1730,
		2000,
		2710,
		2950,
	},

	sagawaRateBandF: {
		1570,
		1840,
		2130,
		2830,
		3090,
	},

	sagawaRateBandG: {
		1820,
		2100,
		2350,
		3090,
		3340,
	},

	sagawaRateBandH: {
		1950,
		2220,
		2480,
		3220,
		3480,
	},

	sagawaRateBandI: {
		2070,
		2350,
		2600,
		3340,
		3610,
	},

	sagawaRateBandJ: {
		2210,
		2480,
		2720,
		3480,
		3750,
	},

	sagawaRateBandK: {
		1700,
		1960,
		2240,
		2950,
		3210,
	},
}

var sagawaMainlandRateBandByPair = map[sagawaZonePair]sagawaRateBand{
	// 北海道
	{From: sagawaZoneHokkaido, To: sagawaZoneHokkaido}:    sagawaRateBandA,
	{From: sagawaZoneHokkaido, To: sagawaZoneNorthTohoku}: sagawaRateBandC,
	{From: sagawaZoneHokkaido, To: sagawaZoneSouthTohoku}: sagawaRateBandD,
	{From: sagawaZoneHokkaido, To: sagawaZoneKanto}:       sagawaRateBandE,
	{From: sagawaZoneHokkaido, To: sagawaZoneShinetsu}:    sagawaRateBandE,
	{From: sagawaZoneHokkaido, To: sagawaZoneTokai}:       sagawaRateBandF,
	{From: sagawaZoneHokkaido, To: sagawaZoneHokuriku}:    sagawaRateBandF,
	{From: sagawaZoneHokkaido, To: sagawaZoneKansai}:      sagawaRateBandG,
	{From: sagawaZoneHokkaido, To: sagawaZoneChugoku}:     sagawaRateBandH,
	{From: sagawaZoneHokkaido, To: sagawaZoneShikoku}:     sagawaRateBandI,
	{From: sagawaZoneHokkaido, To: sagawaZoneNorthKyushu}: sagawaRateBandJ,
	{From: sagawaZoneHokkaido, To: sagawaZoneSouthKyushu}: sagawaRateBandJ,

	// 北東北
	{From: sagawaZoneNorthTohoku, To: sagawaZoneNorthTohoku}: sagawaRateBandA,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneSouthTohoku}: sagawaRateBandA,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneKanto}:       sagawaRateBandB,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneShinetsu}:    sagawaRateBandB,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneTokai}:       sagawaRateBandC,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneHokuriku}:    sagawaRateBandC,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneKansai}:      sagawaRateBandD,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneChugoku}:     sagawaRateBandE,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneShikoku}:     sagawaRateBandF,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneNorthKyushu}: sagawaRateBandK,
	{From: sagawaZoneNorthTohoku, To: sagawaZoneSouthKyushu}: sagawaRateBandK,

	// 南東北
	{From: sagawaZoneSouthTohoku, To: sagawaZoneSouthTohoku}: sagawaRateBandA,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneKanto}:       sagawaRateBandA,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneShinetsu}:    sagawaRateBandA,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneTokai}:       sagawaRateBandB,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneHokuriku}:    sagawaRateBandB,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneKansai}:      sagawaRateBandC,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneChugoku}:     sagawaRateBandE,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneShikoku}:     sagawaRateBandF,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneNorthKyushu}: sagawaRateBandK,
	{From: sagawaZoneSouthTohoku, To: sagawaZoneSouthKyushu}: sagawaRateBandK,

	// 関東
	{From: sagawaZoneKanto, To: sagawaZoneKanto}:       sagawaRateBandA,
	{From: sagawaZoneKanto, To: sagawaZoneShinetsu}:    sagawaRateBandA,
	{From: sagawaZoneKanto, To: sagawaZoneTokai}:       sagawaRateBandA,
	{From: sagawaZoneKanto, To: sagawaZoneHokuriku}:    sagawaRateBandA,
	{From: sagawaZoneKanto, To: sagawaZoneKansai}:      sagawaRateBandB,
	{From: sagawaZoneKanto, To: sagawaZoneChugoku}:     sagawaRateBandC,
	{From: sagawaZoneKanto, To: sagawaZoneShikoku}:     sagawaRateBandD,
	{From: sagawaZoneKanto, To: sagawaZoneNorthKyushu}: sagawaRateBandE,
	{From: sagawaZoneKanto, To: sagawaZoneSouthKyushu}: sagawaRateBandE,

	// 信越
	{From: sagawaZoneShinetsu, To: sagawaZoneShinetsu}:    sagawaRateBandA,
	{From: sagawaZoneShinetsu, To: sagawaZoneTokai}:       sagawaRateBandA,
	{From: sagawaZoneShinetsu, To: sagawaZoneHokuriku}:    sagawaRateBandA,
	{From: sagawaZoneShinetsu, To: sagawaZoneKansai}:      sagawaRateBandB,
	{From: sagawaZoneShinetsu, To: sagawaZoneChugoku}:     sagawaRateBandC,
	{From: sagawaZoneShinetsu, To: sagawaZoneShikoku}:     sagawaRateBandD,
	{From: sagawaZoneShinetsu, To: sagawaZoneNorthKyushu}: sagawaRateBandE,
	{From: sagawaZoneShinetsu, To: sagawaZoneSouthKyushu}: sagawaRateBandE,

	// 東海
	{From: sagawaZoneTokai, To: sagawaZoneTokai}:       sagawaRateBandA,
	{From: sagawaZoneTokai, To: sagawaZoneHokuriku}:    sagawaRateBandA,
	{From: sagawaZoneTokai, To: sagawaZoneKansai}:      sagawaRateBandA,
	{From: sagawaZoneTokai, To: sagawaZoneChugoku}:     sagawaRateBandB,
	{From: sagawaZoneTokai, To: sagawaZoneShikoku}:     sagawaRateBandC,
	{From: sagawaZoneTokai, To: sagawaZoneNorthKyushu}: sagawaRateBandC,
	{From: sagawaZoneTokai, To: sagawaZoneSouthKyushu}: sagawaRateBandC,

	// 北陸
	{From: sagawaZoneHokuriku, To: sagawaZoneHokuriku}:    sagawaRateBandA,
	{From: sagawaZoneHokuriku, To: sagawaZoneKansai}:      sagawaRateBandA,
	{From: sagawaZoneHokuriku, To: sagawaZoneChugoku}:     sagawaRateBandB,
	{From: sagawaZoneHokuriku, To: sagawaZoneShikoku}:     sagawaRateBandC,
	{From: sagawaZoneHokuriku, To: sagawaZoneNorthKyushu}: sagawaRateBandC,
	{From: sagawaZoneHokuriku, To: sagawaZoneSouthKyushu}: sagawaRateBandC,

	// 関西
	{From: sagawaZoneKansai, To: sagawaZoneKansai}:      sagawaRateBandA,
	{From: sagawaZoneKansai, To: sagawaZoneChugoku}:     sagawaRateBandA,
	{From: sagawaZoneKansai, To: sagawaZoneShikoku}:     sagawaRateBandB,
	{From: sagawaZoneKansai, To: sagawaZoneNorthKyushu}: sagawaRateBandB,
	{From: sagawaZoneKansai, To: sagawaZoneSouthKyushu}: sagawaRateBandB,

	// 中国
	{From: sagawaZoneChugoku, To: sagawaZoneChugoku}:     sagawaRateBandA,
	{From: sagawaZoneChugoku, To: sagawaZoneShikoku}:     sagawaRateBandB,
	{From: sagawaZoneChugoku, To: sagawaZoneNorthKyushu}: sagawaRateBandA,
	{From: sagawaZoneChugoku, To: sagawaZoneSouthKyushu}: sagawaRateBandA,

	// 四国
	{From: sagawaZoneShikoku, To: sagawaZoneShikoku}:     sagawaRateBandA,
	{From: sagawaZoneShikoku, To: sagawaZoneNorthKyushu}: sagawaRateBandB,
	{From: sagawaZoneShikoku, To: sagawaZoneSouthKyushu}: sagawaRateBandB,

	// 北九州
	{From: sagawaZoneNorthKyushu, To: sagawaZoneNorthKyushu}: sagawaRateBandA,
	{From: sagawaZoneNorthKyushu, To: sagawaZoneSouthKyushu}: sagawaRateBandA,

	// 南九州
	{From: sagawaZoneSouthKyushu, To: sagawaZoneSouthKyushu}: sagawaRateBandA,
}

var sagawaOkinawaRatesByMainlandZone = map[sagawaZone][5]int64{
	sagawaZoneHokkaido: {
		2552,
		4807,
		7579,
		11220,
		14740,
	},

	sagawaZoneNorthTohoku: {
		2442,
		4158,
		6292,
		9185,
		12595,
	},

	sagawaZoneSouthTohoku: {
		2442,
		3839,
		5753,
		8965,
		12067,
	},

	sagawaZoneKanto: {
		1914,
		3520,
		4686,
		7579,
		10560,
	},

	sagawaZoneShinetsu: {
		1914,
		3520,
		4686,
		7579,
		10560,
	},

	sagawaZoneTokai: {
		1914,
		3080,
		5016,
		7260,
		9493,
	},

	sagawaZoneHokuriku: {
		1914,
		3080,
		5016,
		7260,
		9493,
	},

	sagawaZoneKansai: {
		1914,
		2662,
		3949,
		6083,
		8426,
	},

	sagawaZoneChugoku: {
		1914,
		3201,
		4587,
		6083,
		7898,
	},

	sagawaZoneShikoku: {
		1914,
		2981,
		4158,
		5654,
		7359,
	},

	sagawaZoneNorthKyushu: {
		1914,
		2233,
		3201,
		4807,
		6512,
	},

	sagawaZoneSouthKyushu: {
		1914,
		2233,
		3201,
		4587,
		6083,
	},
}

var sagawaOkinawaSameZoneRates = [5]int64{
	910,
	1220,
	1520,
	2180,
	2440,
}
