// backend/internal/domain/transportation/post_rate.go
package transportation

import "errors"

const (
	PostPublicRateVersion = "public-verified-2026-08-21"

	postStandardMaxWeightGrams = 25000
	postMaxWeightGrams         = 30000
	postMaxTotalSizeMM         = 1700

	postHeavySurcharge int64 = 560
)

var (
	ErrPostPackageTooLarge = errors.New(
		"transportation: post package too large",
	)

	ErrPostPackageTooHeavy = errors.New(
		"transportation: post package too heavy",
	)

	ErrPostRateNotFound = errors.New(
		"transportation: post rate not found",
	)
)

type postZone uint8

const (
	postZoneHokkaido postZone = iota + 1
	postZoneTohoku
	postZoneKanto
	postZoneShinetsu
	postZoneHokuriku
	postZoneTokai
	postZoneKinki
	postZoneChugoku
	postZoneShikoku
	postZoneKyushu
	postZoneOkinawa
)

type postRateBand uint16

const (
	postRateBand880  postRateBand = 880
	postRateBand990  postRateBand = 990
	postRateBand1100 postRateBand = 1100
	postRateBand1150 postRateBand = 1150
	postRateBand1340 postRateBand = 1340
	postRateBand1410 postRateBand = 1410
	postRateBand1450 postRateBand = 1450
	postRateBand1590 postRateBand = 1590
	postRateBand1600 postRateBand = 1600
	postRateBand1740 postRateBand = 1740
	postRateBand1750 postRateBand = 1750
)

type postZonePair struct {
	From postZone
	To   postZone
}

type postRateCalculator struct{}

func newPostRateCalculator() carrierCalculator {
	return &postRateCalculator{}
}

func (c *postRateCalculator) Calculate(
	input CarrierRateInput,
) (Quote, error) {
	if err := input.Package.Validate(); err != nil {
		return Quote{}, err
	}

	size, err := resolvePostSize(input.Package)
	if err != nil {
		return Quote{}, err
	}

	originZone, err := postZoneByPrefectureCode(
		input.OriginPrefectureCode,
	)
	if err != nil {
		return Quote{}, err
	}

	destinationZone, err := postZoneByPrefectureCode(
		input.DestinationPrefectureCode,
	)
	if err != nil {
		return Quote{}, err
	}

	amount, err := resolvePostRate(
		input.OriginPrefectureCode,
		input.DestinationPrefectureCode,
		originZone,
		destinationZone,
		size,
	)
	if err != nil {
		return Quote{}, err
	}

	if input.Package.WeightGrams > postStandardMaxWeightGrams {
		amount += postHeavySurcharge
	}

	return Quote{
		Carrier: CarrierPost,
		Size:    size,
		Amount:  amount,
	}, nil
}

func resolvePostSize(pkg Package) (int, error) {
	if err := pkg.Validate(); err != nil {
		return 0, err
	}

	if pkg.WeightGrams > postMaxWeightGrams {
		return 0, ErrPostPackageTooHeavy
	}

	totalSizeMM := pkg.TotalSizeMM()
	if totalSizeMM > postMaxTotalSizeMM {
		return 0, ErrPostPackageTooLarge
	}

	switch {
	case totalSizeMM <= 0:
		return 0, ErrInvalidPackage
	case totalSizeMM <= 600:
		return 60, nil
	case totalSizeMM <= 800:
		return 80, nil
	case totalSizeMM <= 1000:
		return 100, nil
	case totalSizeMM <= 1200:
		return 120, nil
	case totalSizeMM <= 1400:
		return 140, nil
	case totalSizeMM <= 1600:
		return 160, nil
	case totalSizeMM <= 1700:
		return 170, nil
	default:
		return 0, ErrPostPackageTooLarge
	}
}

func postRateIndexBySize(size int) (int, error) {
	switch size {
	case 60:
		return 0, nil
	case 80:
		return 1, nil
	case 100:
		return 2, nil
	case 120:
		return 3, nil
	case 140:
		return 4, nil
	case 160:
		return 5, nil
	case 170:
		return 6, nil
	default:
		return 0, ErrInvalidPackage
	}
}

func resolvePostRate(
	originPrefectureCode PrefectureCode,
	destinationPrefectureCode PrefectureCode,
	originZone postZone,
	destinationZone postZone,
	size int,
) (int64, error) {
	index, err := postRateIndexBySize(size)
	if err != nil {
		return 0, err
	}

	if originPrefectureCode == destinationPrefectureCode {
		return postSamePrefectureRates[index], nil
	}

	pair := normalizePostZonePair(
		originZone,
		destinationZone,
	)

	band, ok := postRateBandByPair[pair]
	if !ok {
		return 0, ErrPostRateNotFound
	}

	rates, ok := postRatesByBand[band]
	if !ok {
		return 0, ErrPostRateNotFound
	}

	return rates[index], nil
}

func normalizePostZonePair(
	a postZone,
	b postZone,
) postZonePair {
	if a <= b {
		return postZonePair{
			From: a,
			To:   b,
		}
	}

	return postZonePair{
		From: b,
		To:   a,
	}
}

func postZoneByPrefectureCode(
	code PrefectureCode,
) (postZone, error) {
	switch code {
	case PrefectureHokkaido:
		return postZoneHokkaido, nil

	case PrefectureAomori,
		PrefectureIwate,
		PrefectureMiyagi,
		PrefectureAkita,
		PrefectureYamagata,
		PrefectureFukushima:
		return postZoneTohoku, nil

	case PrefectureIbaraki,
		PrefectureTochigi,
		PrefectureGunma,
		PrefectureSaitama,
		PrefectureChiba,
		PrefectureTokyo,
		PrefectureKanagawa,
		PrefectureYamanashi:
		return postZoneKanto, nil

	case PrefectureNiigata,
		PrefectureNagano:
		return postZoneShinetsu, nil

	case PrefectureToyama,
		PrefectureIshikawa,
		PrefectureFukui:
		return postZoneHokuriku, nil

	case PrefectureGifu,
		PrefectureShizuoka,
		PrefectureAichi,
		PrefectureMie:
		return postZoneTokai, nil

	case PrefectureShiga,
		PrefectureKyoto,
		PrefectureOsaka,
		PrefectureHyogo,
		PrefectureNara,
		PrefectureWakayama:
		return postZoneKinki, nil

	case PrefectureTottori,
		PrefectureShimane,
		PrefectureOkayama,
		PrefectureHiroshima,
		PrefectureYamaguchi:
		return postZoneChugoku, nil

	case PrefectureTokushima,
		PrefectureKagawa,
		PrefectureEhime,
		PrefectureKochi:
		return postZoneShikoku, nil

	case PrefectureFukuoka,
		PrefectureSaga,
		PrefectureNagasaki,
		PrefectureKumamoto,
		PrefectureOita,
		PrefectureMiyazaki,
		PrefectureKagoshima:
		return postZoneKyushu, nil

	case PrefectureOkinawa:
		return postZoneOkinawa, nil

	default:
		return 0, ErrInvalidPrefectureCode
	}
}

// index:
// 0 = 60
// 1 = 80
// 2 = 100
// 3 = 120
// 4 = 140
// 5 = 160
// 6 = 170

var postSamePrefectureRates = [7]int64{
	820,
	1130,
	1450,
	1770,
	2120,
	2450,
	3000,
}

var postRatesByBand = map[postRateBand][7]int64{
	postRateBand880: {
		880,
		1200,
		1500,
		1830,
		2170,
		2500,
		3070,
	},

	postRateBand990: {
		990,
		1310,
		1620,
		1940,
		2300,
		2610,
		3750,
	},

	postRateBand1100: {
		1100,
		1450,
		1810,
		2130,
		2510,
		2820,
		3970,
	},

	postRateBand1150: {
		1150,
		1440,
		1780,
		2080,
		2440,
		2750,
		3890,
	},

	postRateBand1340: {
		1340,
		1690,
		2030,
		2370,
		2730,
		3060,
		4200,
	},

	postRateBand1410: {
		1410,
		1710,
		2020,
		2340,
		2680,
		3010,
		4140,
	},

	postRateBand1450: {
		1450,
		1810,
		2160,
		2490,
		2860,
		3180,
		4350,
	},

	postRateBand1590: {
		1590,
		1890,
		2190,
		2500,
		2850,
		3170,
		4860,
	},

	postRateBand1600: {
		1600,
		1970,
		2320,
		2630,
		3010,
		3330,
		5040,
	},

	postRateBand1740: {
		1740,
		2040,
		2350,
		2650,
		3010,
		3330,
		5030,
	},

	postRateBand1750: {
		1750,
		2050,
		2380,
		2680,
		3060,
		3380,
		5090,
	},
}

var postRateBandByPair = map[postZonePair]postRateBand{
	// 北海道
	{From: postZoneHokkaido, To: postZoneTohoku}:   postRateBand1150,
	{From: postZoneHokkaido, To: postZoneKanto}:    postRateBand1410,
	{From: postZoneHokkaido, To: postZoneShinetsu}: postRateBand1410,
	{From: postZoneHokkaido, To: postZoneHokuriku}: postRateBand1590,
	{From: postZoneHokkaido, To: postZoneTokai}:    postRateBand1590,
	{From: postZoneHokkaido, To: postZoneKinki}:    postRateBand1740,
	{From: postZoneHokkaido, To: postZoneChugoku}:  postRateBand1740,
	{From: postZoneHokkaido, To: postZoneShikoku}:  postRateBand1740,
	{From: postZoneHokkaido, To: postZoneKyushu}:   postRateBand1740,
	{From: postZoneHokkaido, To: postZoneOkinawa}:  postRateBand1750,

	// 東北
	{From: postZoneTohoku, To: postZoneTohoku}:   postRateBand880,
	{From: postZoneTohoku, To: postZoneKanto}:    postRateBand880,
	{From: postZoneTohoku, To: postZoneShinetsu}: postRateBand880,
	{From: postZoneTohoku, To: postZoneHokuriku}: postRateBand990,
	{From: postZoneTohoku, To: postZoneTokai}:    postRateBand990,
	{From: postZoneTohoku, To: postZoneKinki}:    postRateBand1150,
	{From: postZoneTohoku, To: postZoneChugoku}:  postRateBand1410,
	{From: postZoneTohoku, To: postZoneShikoku}:  postRateBand1410,
	{From: postZoneTohoku, To: postZoneKyushu}:   postRateBand1740,
	{From: postZoneTohoku, To: postZoneOkinawa}:  postRateBand1750,

	// 関東
	{From: postZoneKanto, To: postZoneKanto}:    postRateBand880,
	{From: postZoneKanto, To: postZoneShinetsu}: postRateBand880,
	{From: postZoneKanto, To: postZoneHokuriku}: postRateBand880,
	{From: postZoneKanto, To: postZoneTokai}:    postRateBand880,
	{From: postZoneKanto, To: postZoneKinki}:    postRateBand990,
	{From: postZoneKanto, To: postZoneChugoku}:  postRateBand1150,
	{From: postZoneKanto, To: postZoneShikoku}:  postRateBand1150,
	{From: postZoneKanto, To: postZoneKyushu}:   postRateBand1410,
	{From: postZoneKanto, To: postZoneOkinawa}:  postRateBand1450,

	// 信越
	{From: postZoneShinetsu, To: postZoneShinetsu}: postRateBand880,
	{From: postZoneShinetsu, To: postZoneHokuriku}: postRateBand880,
	{From: postZoneShinetsu, To: postZoneTokai}:    postRateBand880,
	{From: postZoneShinetsu, To: postZoneKinki}:    postRateBand990,
	{From: postZoneShinetsu, To: postZoneChugoku}:  postRateBand1150,
	{From: postZoneShinetsu, To: postZoneShikoku}:  postRateBand1150,
	{From: postZoneShinetsu, To: postZoneKyushu}:   postRateBand1410,
	{From: postZoneShinetsu, To: postZoneOkinawa}:  postRateBand1600,

	// 北陸
	{From: postZoneHokuriku, To: postZoneHokuriku}: postRateBand880,
	{From: postZoneHokuriku, To: postZoneTokai}:    postRateBand880,
	{From: postZoneHokuriku, To: postZoneKinki}:    postRateBand880,
	{From: postZoneHokuriku, To: postZoneChugoku}:  postRateBand990,
	{From: postZoneHokuriku, To: postZoneShikoku}:  postRateBand990,
	{From: postZoneHokuriku, To: postZoneKyushu}:   postRateBand1150,
	{From: postZoneHokuriku, To: postZoneOkinawa}:  postRateBand1600,

	// 東海
	{From: postZoneTokai, To: postZoneTokai}:   postRateBand880,
	{From: postZoneTokai, To: postZoneKinki}:   postRateBand880,
	{From: postZoneTokai, To: postZoneChugoku}: postRateBand990,
	{From: postZoneTokai, To: postZoneShikoku}: postRateBand990,
	{From: postZoneTokai, To: postZoneKyushu}:  postRateBand1150,
	{From: postZoneTokai, To: postZoneOkinawa}: postRateBand1450,

	// 近畿
	{From: postZoneKinki, To: postZoneKinki}:   postRateBand880,
	{From: postZoneKinki, To: postZoneChugoku}: postRateBand880,
	{From: postZoneKinki, To: postZoneShikoku}: postRateBand880,
	{From: postZoneKinki, To: postZoneKyushu}:  postRateBand990,
	{From: postZoneKinki, To: postZoneOkinawa}: postRateBand1450,

	// 中国
	{From: postZoneChugoku, To: postZoneChugoku}: postRateBand880,
	{From: postZoneChugoku, To: postZoneShikoku}: postRateBand880,
	{From: postZoneChugoku, To: postZoneKyushu}:  postRateBand880,
	{From: postZoneChugoku, To: postZoneOkinawa}: postRateBand1340,

	// 四国
	{From: postZoneShikoku, To: postZoneShikoku}: postRateBand880,
	{From: postZoneShikoku, To: postZoneKyushu}:  postRateBand990,
	{From: postZoneShikoku, To: postZoneOkinawa}: postRateBand1450,

	// 九州
	{From: postZoneKyushu, To: postZoneKyushu}:  postRateBand880,
	{From: postZoneKyushu, To: postZoneOkinawa}: postRateBand1100,
}
