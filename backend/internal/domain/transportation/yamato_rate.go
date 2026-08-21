// backend/internal/domain/transportation/yamato_rate.go
package transportation

import "errors"

const (
	YamatoPublicRateVersion = "2025-10-01"

	yamatoMaxWeightGrams = 30000
	yamatoMaxTotalSizeMM = 2000
	yamatoMaxSideMM      = 1700
)

var (
	ErrYamatoPackageTooLarge = errors.New("transportation: yamato package too large")
	ErrYamatoRateNotFound    = errors.New("transportation: yamato rate not found")
)

type yamatoZone uint8

const (
	yamatoZoneHokkaido yamatoZone = iota + 1
	yamatoZoneNorthTohoku
	yamatoZoneSouthTohoku
	yamatoZoneKanto
	yamatoZoneShinetsu
	yamatoZoneHokuriku
	yamatoZoneChubu
	yamatoZoneKansai
	yamatoZoneChugoku
	yamatoZoneShikoku
	yamatoZoneKyushu
	yamatoZoneOkinawa
)

type yamatoRateBand uint8

const (
	yamatoRateBandA yamatoRateBand = iota + 1
	yamatoRateBandB
	yamatoRateBandC
	yamatoRateBandD
	yamatoRateBandE
	yamatoRateBandF
	yamatoRateBandG
	yamatoRateBandH
	yamatoRateBandI
	yamatoRateBandJ
)

type yamatoZonePair struct {
	From yamatoZone
	To   yamatoZone
}

type yamatoRateCalculator struct{}

func newYamatoRateCalculator() carrierCalculator {
	return &yamatoRateCalculator{}
}

func (c *yamatoRateCalculator) Calculate(input CarrierRateInput) (Quote, error) {
	if err := input.Package.Validate(); err != nil {
		return Quote{}, err
	}

	size, err := resolveYamatoSize(input.Package)
	if err != nil {
		return Quote{}, err
	}

	originZone, err := yamatoZoneByPrefectureCode(input.OriginPrefectureCode)
	if err != nil {
		return Quote{}, err
	}

	destinationZone, err := yamatoZoneByPrefectureCode(input.DestinationPrefectureCode)
	if err != nil {
		return Quote{}, err
	}

	amount, err := resolveYamatoRate(
		input.OriginPrefectureCode,
		input.DestinationPrefectureCode,
		originZone,
		destinationZone,
		size,
	)
	if err != nil {
		return Quote{}, err
	}

	return Quote{
		Carrier: CarrierYamato,
		Size:    size,
		Amount:  amount,
	}, nil
}

func resolveYamatoSize(pkg Package) (int, error) {
	if err := pkg.Validate(); err != nil {
		return 0, err
	}

	if pkg.WeightGrams > yamatoMaxWeightGrams {
		return 0, ErrYamatoPackageTooLarge
	}

	if pkg.WidthMM > yamatoMaxSideMM ||
		pkg.LengthMM > yamatoMaxSideMM ||
		pkg.HeightMM > yamatoMaxSideMM {
		return 0, ErrYamatoPackageTooLarge
	}

	totalSizeMM := pkg.TotalSizeMM()
	if totalSizeMM > yamatoMaxTotalSizeMM {
		return 0, ErrYamatoPackageTooLarge
	}

	dimensionSize, err := yamatoSizeByDimension(totalSizeMM)
	if err != nil {
		return 0, err
	}

	weightSize, err := yamatoSizeByWeight(pkg.WeightGrams)
	if err != nil {
		return 0, err
	}

	if weightSize > dimensionSize {
		return weightSize, nil
	}

	return dimensionSize, nil
}

func yamatoSizeByDimension(totalSizeMM int) (int, error) {
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
	case totalSizeMM <= 1800:
		return 180, nil
	case totalSizeMM <= 2000:
		return 200, nil
	default:
		return 0, ErrYamatoPackageTooLarge
	}
}

func yamatoSizeByWeight(weightGrams int) (int, error) {
	switch {
	case weightGrams <= 0:
		return 0, ErrInvalidPackage
	case weightGrams <= 2000:
		return 60, nil
	case weightGrams <= 5000:
		return 80, nil
	case weightGrams <= 10000:
		return 100, nil
	case weightGrams <= 15000:
		return 120, nil
	case weightGrams <= 20000:
		return 140, nil
	case weightGrams <= 25000:
		return 160, nil
	case weightGrams <= 30000:
		return 180, nil
	default:
		return 0, ErrYamatoPackageTooLarge
	}
}

func yamatoRateIndexBySize(size int) (int, error) {
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
	case 180:
		return 6, nil
	case 200:
		return 7, nil
	default:
		return 0, ErrInvalidPackage
	}
}

func resolveYamatoRate(
	originPrefectureCode PrefectureCode,
	destinationPrefectureCode PrefectureCode,
	originZone yamatoZone,
	destinationZone yamatoZone,
	size int,
) (int64, error) {
	index, err := yamatoRateIndexBySize(size)
	if err != nil {
		return 0, err
	}

	if originPrefectureCode == destinationPrefectureCode &&
		originPrefectureCode != PrefectureOkinawa {
		return yamatoSamePrefectureRates[index], nil
	}

	if originZone == yamatoZoneOkinawa || destinationZone == yamatoZoneOkinawa {
		rates, err := yamatoOkinawaRouteRates(originZone, destinationZone)
		if err != nil {
			return 0, err
		}
		return rates[index], nil
	}

	pair := normalizeYamatoZonePair(originZone, destinationZone)

	band, ok := yamatoMainlandRateBandByPair[pair]
	if !ok {
		return 0, ErrYamatoRateNotFound
	}

	rates, ok := yamatoRatesByBand[band]
	if !ok {
		return 0, ErrYamatoRateNotFound
	}

	return rates[index], nil
}

func normalizeYamatoZonePair(a yamatoZone, b yamatoZone) yamatoZonePair {
	if a <= b {
		return yamatoZonePair{
			From: a,
			To:   b,
		}
	}

	return yamatoZonePair{
		From: b,
		To:   a,
	}
}

func yamatoOkinawaRouteRates(originZone yamatoZone, destinationZone yamatoZone) ([8]int64, error) {
	if originZone == yamatoZoneOkinawa && destinationZone == yamatoZoneOkinawa {
		return yamatoOkinawaSameZoneRates, nil
	}

	mainlandZone := originZone
	if mainlandZone == yamatoZoneOkinawa {
		mainlandZone = destinationZone
	}

	rates, ok := yamatoOkinawaRatesByMainlandZone[mainlandZone]
	if !ok {
		return [8]int64{}, ErrYamatoRateNotFound
	}

	return rates, nil
}

func yamatoZoneByPrefectureCode(code PrefectureCode) (yamatoZone, error) {
	switch code {
	case PrefectureHokkaido:
		return yamatoZoneHokkaido, nil

	case PrefectureAomori, PrefectureIwate, PrefectureAkita:
		return yamatoZoneNorthTohoku, nil

	case PrefectureMiyagi, PrefectureYamagata, PrefectureFukushima:
		return yamatoZoneSouthTohoku, nil

	case PrefectureIbaraki,
		PrefectureTochigi,
		PrefectureGunma,
		PrefectureSaitama,
		PrefectureChiba,
		PrefectureTokyo,
		PrefectureKanagawa,
		PrefectureYamanashi:
		return yamatoZoneKanto, nil

	case PrefectureNiigata, PrefectureNagano:
		return yamatoZoneShinetsu, nil

	case PrefectureToyama, PrefectureIshikawa, PrefectureFukui:
		return yamatoZoneHokuriku, nil

	case PrefectureGifu, PrefectureShizuoka, PrefectureAichi, PrefectureMie:
		return yamatoZoneChubu, nil

	case PrefectureShiga,
		PrefectureKyoto,
		PrefectureOsaka,
		PrefectureHyogo,
		PrefectureNara,
		PrefectureWakayama:
		return yamatoZoneKansai, nil

	case PrefectureTottori,
		PrefectureShimane,
		PrefectureOkayama,
		PrefectureHiroshima,
		PrefectureYamaguchi:
		return yamatoZoneChugoku, nil

	case PrefectureTokushima, PrefectureKagawa, PrefectureEhime, PrefectureKochi:
		return yamatoZoneShikoku, nil

	case PrefectureFukuoka,
		PrefectureSaga,
		PrefectureNagasaki,
		PrefectureKumamoto,
		PrefectureOita,
		PrefectureMiyazaki,
		PrefectureKagoshima:
		return yamatoZoneKyushu, nil

	case PrefectureOkinawa:
		return yamatoZoneOkinawa, nil

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
// 6 = 180
// 7 = 200

var yamatoSamePrefectureRates = [8]int64{
	790,
	1090,
	1410,
	1730,
	2090,
	2410,
	3030,
	3690,
}

var yamatoRatesByBand = map[yamatoRateBand][8]int64{
	yamatoRateBandA: {
		940,
		1230,
		1530,
		2040,
		2630,
		3020,
		3680,
		4470,
	},

	yamatoRateBandB: {
		1060,
		1350,
		1650,
		2170,
		2780,
		3160,
		4480,
		5410,
	},

	yamatoRateBandC: {
		1190,
		1480,
		1790,
		2310,
		2930,
		3320,
		4900,
		6220,
	},

	yamatoRateBandD: {
		1320,
		1610,
		1920,
		2460,
		3100,
		3480,
		5060,
		6380,
	},

	yamatoRateBandE: {
		1460,
		1740,
		2050,
		2610,
		3250,
		3630,
		5220,
		6540,
	},

	yamatoRateBandF: {
		1610,
		1900,
		2200,
		2780,
		3440,
		3820,
		6460,
		8050,
	},

	yamatoRateBandG: {
		1760,
		2050,
		2360,
		2940,
		3620,
		4010,
		6650,
		8230,
	},

	yamatoRateBandH: {
		1920,
		2200,
		2510,
		3120,
		3810,
		4180,
		6820,
		8410,
	},

	yamatoRateBandI: {
		2070,
		2360,
		2670,
		3280,
		3990,
		4370,
		7410,
		9320,
	},

	yamatoRateBandJ: {
		2340,
		2620,
		2930,
		3580,
		4310,
		4690,
		7860,
		9770,
	},
}

var yamatoMainlandRateBandByPair = map[yamatoZonePair]yamatoRateBand{
	// 北海道
	{From: yamatoZoneHokkaido, To: yamatoZoneHokkaido}:    yamatoRateBandA,
	{From: yamatoZoneHokkaido, To: yamatoZoneNorthTohoku}: yamatoRateBandC,
	{From: yamatoZoneHokkaido, To: yamatoZoneSouthTohoku}: yamatoRateBandD,
	{From: yamatoZoneHokkaido, To: yamatoZoneKanto}:       yamatoRateBandE,
	{From: yamatoZoneHokkaido, To: yamatoZoneShinetsu}:    yamatoRateBandE,
	{From: yamatoZoneHokkaido, To: yamatoZoneHokuriku}:    yamatoRateBandF,
	{From: yamatoZoneHokkaido, To: yamatoZoneChubu}:       yamatoRateBandF,
	{From: yamatoZoneHokkaido, To: yamatoZoneKansai}:      yamatoRateBandH,
	{From: yamatoZoneHokkaido, To: yamatoZoneChugoku}:     yamatoRateBandI,
	{From: yamatoZoneHokkaido, To: yamatoZoneShikoku}:     yamatoRateBandI,
	{From: yamatoZoneHokkaido, To: yamatoZoneKyushu}:      yamatoRateBandJ,

	// 北東北
	{From: yamatoZoneNorthTohoku, To: yamatoZoneNorthTohoku}: yamatoRateBandA,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneSouthTohoku}: yamatoRateBandA,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneKanto}:       yamatoRateBandB,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneShinetsu}:    yamatoRateBandB,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneHokuriku}:    yamatoRateBandC,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneChubu}:       yamatoRateBandC,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneKansai}:      yamatoRateBandD,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneChugoku}:     yamatoRateBandE,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneShikoku}:     yamatoRateBandE,
	{From: yamatoZoneNorthTohoku, To: yamatoZoneKyushu}:      yamatoRateBandG,

	// 南東北
	{From: yamatoZoneSouthTohoku, To: yamatoZoneSouthTohoku}: yamatoRateBandA,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneKanto}:       yamatoRateBandA,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneShinetsu}:    yamatoRateBandA,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneHokuriku}:    yamatoRateBandB,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneChubu}:       yamatoRateBandB,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneKansai}:      yamatoRateBandC,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneChugoku}:     yamatoRateBandE,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneShikoku}:     yamatoRateBandE,
	{From: yamatoZoneSouthTohoku, To: yamatoZoneKyushu}:      yamatoRateBandG,

	// 関東
	{From: yamatoZoneKanto, To: yamatoZoneKanto}:    yamatoRateBandA,
	{From: yamatoZoneKanto, To: yamatoZoneShinetsu}: yamatoRateBandA,
	{From: yamatoZoneKanto, To: yamatoZoneHokuriku}: yamatoRateBandA,
	{From: yamatoZoneKanto, To: yamatoZoneChubu}:    yamatoRateBandA,
	{From: yamatoZoneKanto, To: yamatoZoneKansai}:   yamatoRateBandB,
	{From: yamatoZoneKanto, To: yamatoZoneChugoku}:  yamatoRateBandC,
	{From: yamatoZoneKanto, To: yamatoZoneShikoku}:  yamatoRateBandC,
	{From: yamatoZoneKanto, To: yamatoZoneKyushu}:   yamatoRateBandE,

	// 信越
	{From: yamatoZoneShinetsu, To: yamatoZoneShinetsu}: yamatoRateBandA,
	{From: yamatoZoneShinetsu, To: yamatoZoneHokuriku}: yamatoRateBandA,
	{From: yamatoZoneShinetsu, To: yamatoZoneChubu}:    yamatoRateBandA,
	{From: yamatoZoneShinetsu, To: yamatoZoneKansai}:   yamatoRateBandB,
	{From: yamatoZoneShinetsu, To: yamatoZoneChugoku}:  yamatoRateBandC,
	{From: yamatoZoneShinetsu, To: yamatoZoneShikoku}:  yamatoRateBandC,
	{From: yamatoZoneShinetsu, To: yamatoZoneKyushu}:   yamatoRateBandE,

	// 北陸
	{From: yamatoZoneHokuriku, To: yamatoZoneHokuriku}: yamatoRateBandA,
	{From: yamatoZoneHokuriku, To: yamatoZoneChubu}:    yamatoRateBandA,
	{From: yamatoZoneHokuriku, To: yamatoZoneKansai}:   yamatoRateBandA,
	{From: yamatoZoneHokuriku, To: yamatoZoneChugoku}:  yamatoRateBandB,
	{From: yamatoZoneHokuriku, To: yamatoZoneShikoku}:  yamatoRateBandB,
	{From: yamatoZoneHokuriku, To: yamatoZoneKyushu}:   yamatoRateBandC,

	// 中部
	{From: yamatoZoneChubu, To: yamatoZoneChubu}:   yamatoRateBandA,
	{From: yamatoZoneChubu, To: yamatoZoneKansai}:  yamatoRateBandA,
	{From: yamatoZoneChubu, To: yamatoZoneChugoku}: yamatoRateBandB,
	{From: yamatoZoneChubu, To: yamatoZoneShikoku}: yamatoRateBandB,
	{From: yamatoZoneChubu, To: yamatoZoneKyushu}:  yamatoRateBandC,

	// 関西
	{From: yamatoZoneKansai, To: yamatoZoneKansai}:  yamatoRateBandA,
	{From: yamatoZoneKansai, To: yamatoZoneChugoku}: yamatoRateBandA,
	{From: yamatoZoneKansai, To: yamatoZoneShikoku}: yamatoRateBandA,
	{From: yamatoZoneKansai, To: yamatoZoneKyushu}:  yamatoRateBandB,

	// 中国
	{From: yamatoZoneChugoku, To: yamatoZoneChugoku}: yamatoRateBandA,
	{From: yamatoZoneChugoku, To: yamatoZoneShikoku}: yamatoRateBandA,
	{From: yamatoZoneChugoku, To: yamatoZoneKyushu}:  yamatoRateBandA,

	// 四国
	{From: yamatoZoneShikoku, To: yamatoZoneShikoku}: yamatoRateBandA,
	{From: yamatoZoneShikoku, To: yamatoZoneKyushu}:  yamatoRateBandB,

	// 九州
	{From: yamatoZoneKyushu, To: yamatoZoneKyushu}: yamatoRateBandA,
}

var yamatoOkinawaRatesByMainlandZone = map[yamatoZone][8]int64{
	yamatoZoneHokkaido: {
		2340,
		2950,
		3590,
		4240,
		4910,
		5560,
		9080,
		10730,
	},

	yamatoZoneNorthTohoku: {
		1920,
		2530,
		3170,
		3820,
		4490,
		5140,
		8550,
		10200,
	},

	yamatoZoneSouthTohoku: {
		1760,
		2380,
		3020,
		3670,
		4340,
		4990,
		8290,
		9940,
	},

	yamatoZoneKanto: {
		1460,
		2070,
		2710,
		3360,
		4030,
		4680,
		7210,
		8860,
	},

	yamatoZoneShinetsu: {
		1610,
		2230,
		2860,
		3510,
		4180,
		4830,
		8020,
		9670,
	},

	yamatoZoneHokuriku: {
		1610,
		2230,
		2860,
		3510,
		4180,
		4830,
		8020,
		9670,
	},

	yamatoZoneChubu: {
		1460,
		2070,
		2710,
		3360,
		4030,
		4680,
		7210,
		8860,
	},

	yamatoZoneKansai: {
		1460,
		2070,
		2710,
		3360,
		4030,
		4680,
		7210,
		8860,
	},

	yamatoZoneChugoku: {
		1460,
		2070,
		2710,
		3360,
		4030,
		4680,
		7210,
		8860,
	},

	yamatoZoneShikoku: {
		1460,
		2070,
		2710,
		3360,
		4030,
		4680,
		7210,
		8860,
	},

	yamatoZoneKyushu: {
		1320,
		1940,
		2580,
		3230,
		3900,
		4550,
		6970,
		8620,
	},
}

var yamatoOkinawaSameZoneRates = [8]int64{
	940,
	1230,
	1530,
	1850,
	2190,
	2510,
	3060,
	3720,
}
