// backend\internal\domain\transportation\island_master.go
package transportation

type IslandDefinition struct {
	IslandCode     IslandCode     `json:"islandCode"`
	PrefectureCode PrefectureCode `json:"prefectureCode"`
	DisplayName    string         `json:"displayName"`
}

const (
	// 北海道
	IslandCodeHokkaidoRebun     IslandCode = "hokkaido-rebun"
	IslandCodeHokkaidoRishiri   IslandCode = "hokkaido-rishiri"
	IslandCodeHokkaidoTeuri     IslandCode = "hokkaido-teuri"
	IslandCodeHokkaidoYagishiri IslandCode = "hokkaido-yagishiri"
	IslandCodeHokkaidoOkushiri  IslandCode = "hokkaido-okushiri"

	// 山形
	IslandCodeYamagataTobishima IslandCode = "yamagata-tobishima"

	// 東京 伊豆諸島
	IslandCodeTokyoIzuOshima   IslandCode = "tokyo-izu-oshima"
	IslandCodeTokyoToshima     IslandCode = "tokyo-toshima"
	IslandCodeTokyoNiijima     IslandCode = "tokyo-niijima"
	IslandCodeTokyoShikinejima IslandCode = "tokyo-shikinejima"
	IslandCodeTokyoKozushima   IslandCode = "tokyo-kozushima"
	IslandCodeTokyoMiyakejima  IslandCode = "tokyo-miyakejima"
	IslandCodeTokyoMikurajima  IslandCode = "tokyo-mikurajima"
	IslandCodeTokyoHachijojima IslandCode = "tokyo-hachijojima"
	IslandCodeTokyoAogashima   IslandCode = "tokyo-aogashima"

	// 東京 小笠原
	IslandCodeTokyoOgasawaraChichijima IslandCode = "tokyo-ogasawara-chichijima"
	IslandCodeTokyoOgasawaraHahajima   IslandCode = "tokyo-ogasawara-hahajima"

	// 新潟
	IslandCodeNiigataSado     IslandCode = "niigata-sado"
	IslandCodeNiigataAwashima IslandCode = "niigata-awashima"

	// 石川
	IslandCodeIshikawaHegurajima IslandCode = "ishikawa-hegurajima"

	// 島根 隠岐
	IslandCodeShimaneDogo         IslandCode = "shimane-dogo"
	IslandCodeShimaneNakanoshima  IslandCode = "shimane-nakanoshima"
	IslandCodeShimaneNishinoshima IslandCode = "shimane-nishinoshima"
	IslandCodeShimaneChiburijima  IslandCode = "shimane-chiburijima"

	// 長崎
	IslandCodeNagasakiTsushima  IslandCode = "nagasaki-tsushima"
	IslandCodeNagasakiIki       IslandCode = "nagasaki-iki"
	IslandCodeNagasakiUku       IslandCode = "nagasaki-uku"
	IslandCodeNagasakiOjika     IslandCode = "nagasaki-ojika"
	IslandCodeNagasakiNakadori  IslandCode = "nagasaki-nakadori"
	IslandCodeNagasakiWakamatsu IslandCode = "nagasaki-wakamatsu"
	IslandCodeNagasakiNaru      IslandCode = "nagasaki-naru"
	IslandCodeNagasakiHisaka    IslandCode = "nagasaki-hisaka"
	IslandCodeNagasakiFukue     IslandCode = "nagasaki-fukue"

	// 熊本
	IslandCodeKumamotoYushima  IslandCode = "kumamoto-yushima"
	IslandCodeKumamotoGoshoura IslandCode = "kumamoto-goshoura"

	// 大分
	IslandCodeOitaHimeshima IslandCode = "oita-himeshima"
	IslandCodeOitaHotojima  IslandCode = "oita-hotojima"

	// 宮崎
	IslandCodeMiyazakiShimanoura IslandCode = "miyazaki-shimanoura"

	// 鹿児島 甑島
	IslandCodeKagoshimaKamikoshiki  IslandCode = "kagoshima-kamikoshiki"
	IslandCodeKagoshimaNakakoshiki  IslandCode = "kagoshima-nakakoshiki"
	IslandCodeKagoshimaShimokoshiki IslandCode = "kagoshima-shimokoshiki"

	// 鹿児島 大隅諸島
	IslandCodeKagoshimaTanegashima      IslandCode = "kagoshima-tanegashima"
	IslandCodeKagoshimaYakushima        IslandCode = "kagoshima-yakushima"
	IslandCodeKagoshimaKuchinoerabujima IslandCode = "kagoshima-kuchinoerabujima"

	// 鹿児島 三島
	IslandCodeKagoshimaTakeshima IslandCode = "kagoshima-mishima-takeshima"
	IslandCodeKagoshimaIoujima   IslandCode = "kagoshima-mishima-ioujima"
	IslandCodeKagoshimaKuroshima IslandCode = "kagoshima-mishima-kuroshima"

	// 鹿児島 トカラ列島
	IslandCodeKagoshimaKuchinoshima IslandCode = "kagoshima-tokara-kuchinoshima"
	IslandCodeKagoshimaNakanoshima  IslandCode = "kagoshima-tokara-nakanoshima"
	IslandCodeKagoshimaSuwanosejima IslandCode = "kagoshima-tokara-suwanosejima"
	IslandCodeKagoshimaTairajima    IslandCode = "kagoshima-tokara-tairajima"
	IslandCodeKagoshimaAkusekijima  IslandCode = "kagoshima-tokara-akusekijima"
	IslandCodeKagoshimaKodakarajima IslandCode = "kagoshima-tokara-kodakarajima"
	IslandCodeKagoshimaTakarajima   IslandCode = "kagoshima-tokara-takarajima"

	// 鹿児島 奄美群島
	IslandCodeKagoshimaAmamiOshima    IslandCode = "kagoshima-amami-oshima"
	IslandCodeKagoshimaKikaijima      IslandCode = "kagoshima-kikaijima"
	IslandCodeKagoshimaTokunoshima    IslandCode = "kagoshima-tokunoshima"
	IslandCodeKagoshimaOkinoerabujima IslandCode = "kagoshima-okinoerabujima"
	IslandCodeKagoshimaYoronjima      IslandCode = "kagoshima-yoronjima"

	// 沖縄
	IslandCodeOkinawaIejima      IslandCode = "okinawa-iejima"
	IslandCodeOkinawaIheya       IslandCode = "okinawa-iheya"
	IslandCodeOkinawaIzena       IslandCode = "okinawa-izena"
	IslandCodeOkinawaAguni       IslandCode = "okinawa-aguni"
	IslandCodeOkinawaTonaki      IslandCode = "okinawa-tonaki"
	IslandCodeOkinawaKumejima    IslandCode = "okinawa-kumejima"
	IslandCodeOkinawaTokashiki   IslandCode = "okinawa-tokashiki"
	IslandCodeOkinawaZamami      IslandCode = "okinawa-zamami"
	IslandCodeOkinawaMinamidaito IslandCode = "okinawa-minamidaito"
	IslandCodeOkinawaKitadaito   IslandCode = "okinawa-kitadaito"
	IslandCodeOkinawaMiyakojima  IslandCode = "okinawa-miyakojima"
	IslandCodeOkinawaTarama      IslandCode = "okinawa-tarama"
	IslandCodeOkinawaIshigaki    IslandCode = "okinawa-ishigaki"
	IslandCodeOkinawaTaketomi    IslandCode = "okinawa-taketomi"
	IslandCodeOkinawaKohama      IslandCode = "okinawa-kohama"
	IslandCodeOkinawaKuroshima   IslandCode = "okinawa-kuroshima"
	IslandCodeOkinawaIriomote    IslandCode = "okinawa-iriomote"
	IslandCodeOkinawaHatoma      IslandCode = "okinawa-hatoma"
	IslandCodeOkinawaHateruma    IslandCode = "okinawa-hateruma"
	IslandCodeOkinawaYonaguni    IslandCode = "okinawa-yonaguni"
)

var islandDefinitions = []IslandDefinition{
	{IslandCode: IslandCodeHokkaidoRebun, PrefectureCode: PrefectureHokkaido, DisplayName: "礼文島"},
	{IslandCode: IslandCodeHokkaidoRishiri, PrefectureCode: PrefectureHokkaido, DisplayName: "利尻島"},
	{IslandCode: IslandCodeHokkaidoTeuri, PrefectureCode: PrefectureHokkaido, DisplayName: "天売島"},
	{IslandCode: IslandCodeHokkaidoYagishiri, PrefectureCode: PrefectureHokkaido, DisplayName: "焼尻島"},
	{IslandCode: IslandCodeHokkaidoOkushiri, PrefectureCode: PrefectureHokkaido, DisplayName: "奥尻島"},

	{IslandCode: IslandCodeYamagataTobishima, PrefectureCode: PrefectureYamagata, DisplayName: "飛島"},

	{IslandCode: IslandCodeTokyoIzuOshima, PrefectureCode: PrefectureTokyo, DisplayName: "伊豆大島"},
	{IslandCode: IslandCodeTokyoToshima, PrefectureCode: PrefectureTokyo, DisplayName: "利島"},
	{IslandCode: IslandCodeTokyoNiijima, PrefectureCode: PrefectureTokyo, DisplayName: "新島"},
	{IslandCode: IslandCodeTokyoShikinejima, PrefectureCode: PrefectureTokyo, DisplayName: "式根島"},
	{IslandCode: IslandCodeTokyoKozushima, PrefectureCode: PrefectureTokyo, DisplayName: "神津島"},
	{IslandCode: IslandCodeTokyoMiyakejima, PrefectureCode: PrefectureTokyo, DisplayName: "三宅島"},
	{IslandCode: IslandCodeTokyoMikurajima, PrefectureCode: PrefectureTokyo, DisplayName: "御蔵島"},
	{IslandCode: IslandCodeTokyoHachijojima, PrefectureCode: PrefectureTokyo, DisplayName: "八丈島"},
	{IslandCode: IslandCodeTokyoAogashima, PrefectureCode: PrefectureTokyo, DisplayName: "青ヶ島"},
	{IslandCode: IslandCodeTokyoOgasawaraChichijima, PrefectureCode: PrefectureTokyo, DisplayName: "父島"},
	{IslandCode: IslandCodeTokyoOgasawaraHahajima, PrefectureCode: PrefectureTokyo, DisplayName: "母島"},

	{IslandCode: IslandCodeNiigataSado, PrefectureCode: PrefectureNiigata, DisplayName: "佐渡島"},
	{IslandCode: IslandCodeNiigataAwashima, PrefectureCode: PrefectureNiigata, DisplayName: "粟島"},
	{IslandCode: IslandCodeIshikawaHegurajima, PrefectureCode: PrefectureIshikawa, DisplayName: "舳倉島"},

	{IslandCode: IslandCodeShimaneDogo, PrefectureCode: PrefectureShimane, DisplayName: "島後"},
	{IslandCode: IslandCodeShimaneNakanoshima, PrefectureCode: PrefectureShimane, DisplayName: "中ノ島"},
	{IslandCode: IslandCodeShimaneNishinoshima, PrefectureCode: PrefectureShimane, DisplayName: "西ノ島"},
	{IslandCode: IslandCodeShimaneChiburijima, PrefectureCode: PrefectureShimane, DisplayName: "知夫里島"},

	{IslandCode: IslandCodeNagasakiTsushima, PrefectureCode: PrefectureNagasaki, DisplayName: "対馬島"},
	{IslandCode: IslandCodeNagasakiIki, PrefectureCode: PrefectureNagasaki, DisplayName: "壱岐島"},
	{IslandCode: IslandCodeNagasakiUku, PrefectureCode: PrefectureNagasaki, DisplayName: "宇久島"},
	{IslandCode: IslandCodeNagasakiOjika, PrefectureCode: PrefectureNagasaki, DisplayName: "小値賀島"},
	{IslandCode: IslandCodeNagasakiNakadori, PrefectureCode: PrefectureNagasaki, DisplayName: "中通島"},
	{IslandCode: IslandCodeNagasakiWakamatsu, PrefectureCode: PrefectureNagasaki, DisplayName: "若松島"},
	{IslandCode: IslandCodeNagasakiNaru, PrefectureCode: PrefectureNagasaki, DisplayName: "奈留島"},
	{IslandCode: IslandCodeNagasakiHisaka, PrefectureCode: PrefectureNagasaki, DisplayName: "久賀島"},
	{IslandCode: IslandCodeNagasakiFukue, PrefectureCode: PrefectureNagasaki, DisplayName: "福江島"},

	{IslandCode: IslandCodeKumamotoYushima, PrefectureCode: PrefectureKumamoto, DisplayName: "湯島"},
	{IslandCode: IslandCodeKumamotoGoshoura, PrefectureCode: PrefectureKumamoto, DisplayName: "御所浦島"},

	{IslandCode: IslandCodeOitaHimeshima, PrefectureCode: PrefectureOita, DisplayName: "姫島"},
	{IslandCode: IslandCodeOitaHotojima, PrefectureCode: PrefectureOita, DisplayName: "保戸島"},
	{IslandCode: IslandCodeMiyazakiShimanoura, PrefectureCode: PrefectureMiyazaki, DisplayName: "島野浦島"},

	{IslandCode: IslandCodeKagoshimaKamikoshiki, PrefectureCode: PrefectureKagoshima, DisplayName: "上甑島"},
	{IslandCode: IslandCodeKagoshimaNakakoshiki, PrefectureCode: PrefectureKagoshima, DisplayName: "中甑島"},
	{IslandCode: IslandCodeKagoshimaShimokoshiki, PrefectureCode: PrefectureKagoshima, DisplayName: "下甑島"},
	{IslandCode: IslandCodeKagoshimaTanegashima, PrefectureCode: PrefectureKagoshima, DisplayName: "種子島"},
	{IslandCode: IslandCodeKagoshimaYakushima, PrefectureCode: PrefectureKagoshima, DisplayName: "屋久島"},
	{IslandCode: IslandCodeKagoshimaKuchinoerabujima, PrefectureCode: PrefectureKagoshima, DisplayName: "口永良部島"},
	{IslandCode: IslandCodeKagoshimaTakeshima, PrefectureCode: PrefectureKagoshima, DisplayName: "竹島"},
	{IslandCode: IslandCodeKagoshimaIoujima, PrefectureCode: PrefectureKagoshima, DisplayName: "硫黄島"},
	{IslandCode: IslandCodeKagoshimaKuroshima, PrefectureCode: PrefectureKagoshima, DisplayName: "黒島"},
	{IslandCode: IslandCodeKagoshimaKuchinoshima, PrefectureCode: PrefectureKagoshima, DisplayName: "口之島"},
	{IslandCode: IslandCodeKagoshimaNakanoshima, PrefectureCode: PrefectureKagoshima, DisplayName: "中之島"},
	{IslandCode: IslandCodeKagoshimaSuwanosejima, PrefectureCode: PrefectureKagoshima, DisplayName: "諏訪之瀬島"},
	{IslandCode: IslandCodeKagoshimaTairajima, PrefectureCode: PrefectureKagoshima, DisplayName: "平島"},
	{IslandCode: IslandCodeKagoshimaAkusekijima, PrefectureCode: PrefectureKagoshima, DisplayName: "悪石島"},
	{IslandCode: IslandCodeKagoshimaKodakarajima, PrefectureCode: PrefectureKagoshima, DisplayName: "小宝島"},
	{IslandCode: IslandCodeKagoshimaTakarajima, PrefectureCode: PrefectureKagoshima, DisplayName: "宝島"},
	{IslandCode: IslandCodeKagoshimaAmamiOshima, PrefectureCode: PrefectureKagoshima, DisplayName: "奄美大島"},
	{IslandCode: IslandCodeKagoshimaKikaijima, PrefectureCode: PrefectureKagoshima, DisplayName: "喜界島"},
	{IslandCode: IslandCodeKagoshimaTokunoshima, PrefectureCode: PrefectureKagoshima, DisplayName: "徳之島"},
	{IslandCode: IslandCodeKagoshimaOkinoerabujima, PrefectureCode: PrefectureKagoshima, DisplayName: "沖永良部島"},
	{IslandCode: IslandCodeKagoshimaYoronjima, PrefectureCode: PrefectureKagoshima, DisplayName: "与論島"},

	{IslandCode: IslandCodeOkinawaIejima, PrefectureCode: PrefectureOkinawa, DisplayName: "伊江島"},
	{IslandCode: IslandCodeOkinawaIheya, PrefectureCode: PrefectureOkinawa, DisplayName: "伊平屋島"},
	{IslandCode: IslandCodeOkinawaIzena, PrefectureCode: PrefectureOkinawa, DisplayName: "伊是名島"},
	{IslandCode: IslandCodeOkinawaAguni, PrefectureCode: PrefectureOkinawa, DisplayName: "粟国島"},
	{IslandCode: IslandCodeOkinawaTonaki, PrefectureCode: PrefectureOkinawa, DisplayName: "渡名喜島"},
	{IslandCode: IslandCodeOkinawaKumejima, PrefectureCode: PrefectureOkinawa, DisplayName: "久米島"},
	{IslandCode: IslandCodeOkinawaTokashiki, PrefectureCode: PrefectureOkinawa, DisplayName: "渡嘉敷島"},
	{IslandCode: IslandCodeOkinawaZamami, PrefectureCode: PrefectureOkinawa, DisplayName: "座間味島"},
	{IslandCode: IslandCodeOkinawaMinamidaito, PrefectureCode: PrefectureOkinawa, DisplayName: "南大東島"},
	{IslandCode: IslandCodeOkinawaKitadaito, PrefectureCode: PrefectureOkinawa, DisplayName: "北大東島"},
	{IslandCode: IslandCodeOkinawaMiyakojima, PrefectureCode: PrefectureOkinawa, DisplayName: "宮古島"},
	{IslandCode: IslandCodeOkinawaTarama, PrefectureCode: PrefectureOkinawa, DisplayName: "多良間島"},
	{IslandCode: IslandCodeOkinawaIshigaki, PrefectureCode: PrefectureOkinawa, DisplayName: "石垣島"},
	{IslandCode: IslandCodeOkinawaTaketomi, PrefectureCode: PrefectureOkinawa, DisplayName: "竹富島"},
	{IslandCode: IslandCodeOkinawaKohama, PrefectureCode: PrefectureOkinawa, DisplayName: "小浜島"},
	{IslandCode: IslandCodeOkinawaKuroshima, PrefectureCode: PrefectureOkinawa, DisplayName: "黒島"},
	{IslandCode: IslandCodeOkinawaIriomote, PrefectureCode: PrefectureOkinawa, DisplayName: "西表島"},
	{IslandCode: IslandCodeOkinawaHatoma, PrefectureCode: PrefectureOkinawa, DisplayName: "鳩間島"},
	{IslandCode: IslandCodeOkinawaHateruma, PrefectureCode: PrefectureOkinawa, DisplayName: "波照間島"},
	{IslandCode: IslandCodeOkinawaYonaguni, PrefectureCode: PrefectureOkinawa, DisplayName: "与那国島"},
}

var islandDefinitionByCode = func() map[IslandCode]IslandDefinition {
	result := make(map[IslandCode]IslandDefinition, len(islandDefinitions))

	for _, definition := range islandDefinitions {
		result[definition.IslandCode] = definition
	}

	return result
}()

func IslandDefinitions() []IslandDefinition {
	result := make([]IslandDefinition, len(islandDefinitions))
	copy(result, islandDefinitions)
	return result
}

func IslandCodes() []IslandCode {
	result := make([]IslandCode, 0, len(islandDefinitions))

	for _, definition := range islandDefinitions {
		result = append(result, definition.IslandCode)
	}

	return result
}

func ParseIslandCode(code string) (IslandCode, error) {
	islandCode := IslandCode(code)

	if !IsValidIslandCode(islandCode) {
		return "", ErrInvalidIslandCode
	}

	return islandCode, nil
}

func IsValidIslandCode(code IslandCode) bool {
	_, ok := islandDefinitionByCode[code]
	return ok
}

func IslandDefinitionByCode(code IslandCode) (IslandDefinition, error) {
	definition, ok := islandDefinitionByCode[code]

	if !ok {
		return IslandDefinition{}, ErrInvalidIslandCode
	}

	return definition, nil
}
