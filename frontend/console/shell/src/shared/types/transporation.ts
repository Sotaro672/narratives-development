// frontend/console/shell/src/shared/types/transporation.ts

export type TransportationRegion =
  | "hokkaido"
  | "tohoku"
  | "kanto"
  | "chubu"
  | "kinki"
  | "chugoku"
  | "shikoku"
  | "kyushu"
  | "okinawa"
  | "islands";

export type PrefectureCode =
  | "01"
  | "02"
  | "03"
  | "04"
  | "05"
  | "06"
  | "07"
  | "08"
  | "09"
  | "10"
  | "11"
  | "12"
  | "13"
  | "14"
  | "15"
  | "16"
  | "17"
  | "18"
  | "19"
  | "20"
  | "21"
  | "22"
  | "23"
  | "24"
  | "25"
  | "26"
  | "27"
  | "28"
  | "29"
  | "30"
  | "31"
  | "32"
  | "33"
  | "34"
  | "35"
  | "36"
  | "37"
  | "38"
  | "39"
  | "40"
  | "41"
  | "42"
  | "43"
  | "44"
  | "45"
  | "46"
  | "47";

export type IslandCode = string;

export const REGION_NAME_BY_CODE: Record<TransportationRegion, string> = {
  hokkaido: "北海道",
  tohoku: "東北",
  kanto: "関東",
  chubu: "中部",
  kinki: "近畿",
  chugoku: "中国",
  shikoku: "四国",
  kyushu: "九州",
  okinawa: "沖縄",
  islands: "島嶼部",
};

export const PREFECTURE_NAME_BY_CODE: Record<PrefectureCode, string> = {
  "01": "北海道",
  "02": "青森県",
  "03": "岩手県",
  "04": "宮城県",
  "05": "秋田県",
  "06": "山形県",
  "07": "福島県",
  "08": "茨城県",
  "09": "栃木県",
  "10": "群馬県",
  "11": "埼玉県",
  "12": "千葉県",
  "13": "東京都",
  "14": "神奈川県",
  "15": "新潟県",
  "16": "富山県",
  "17": "石川県",
  "18": "福井県",
  "19": "山梨県",
  "20": "長野県",
  "21": "岐阜県",
  "22": "静岡県",
  "23": "愛知県",
  "24": "三重県",
  "25": "滋賀県",
  "26": "京都府",
  "27": "大阪府",
  "28": "兵庫県",
  "29": "奈良県",
  "30": "和歌山県",
  "31": "鳥取県",
  "32": "島根県",
  "33": "岡山県",
  "34": "広島県",
  "35": "山口県",
  "36": "徳島県",
  "37": "香川県",
  "38": "愛媛県",
  "39": "高知県",
  "40": "福岡県",
  "41": "佐賀県",
  "42": "長崎県",
  "43": "熊本県",
  "44": "大分県",
  "45": "宮崎県",
  "46": "鹿児島県",
  "47": "沖縄県",
};

export type TransportationPrefectureRate = {
  prefectureCode: PrefectureCode;
  amount: number;
};

export type TransportationIslandRate = {
  islandCode: IslandCode;
  prefectureCode: PrefectureCode;
  amount: number;
};

export type TransportationFeeSetting = {
  id: string;
  companyId: string;
  name: string;
  prefectureRates: TransportationPrefectureRate[];
  islandRates: TransportationIslandRate[];
  createdAt: string;
  createdBy: string;
  createdByName?: string;
  updatedAt: string;
  updatedBy: string;
  updatedByName?: string;
};

export type TransportationIslandDefinition = {
  islandCode: IslandCode;
  prefectureCode: PrefectureCode;
  displayName: string;
};

export type TransportationRegionGroup = {
  region: TransportationRegion;
  prefectureCodes: PrefectureCode[];
  islandCodes: IslandCode[];
};

export type TransportationMaster = {
  regions: TransportationRegionGroup[];
  islands: TransportationIslandDefinition[];
};

export type TransportationFeeSettingInput = {
  name: string;
  prefectureRates: TransportationPrefectureRate[];
  islandRates: TransportationIslandRate[];
};

export function getPrefectureName(code: PrefectureCode): string {
  return PREFECTURE_NAME_BY_CODE[code];
}

export function getTransportationRegionName(region: TransportationRegion): string {
  return REGION_NAME_BY_CODE[region];
}

export function isPrefectureCode(value: string): value is PrefectureCode {
  return Object.prototype.hasOwnProperty.call(PREFECTURE_NAME_BY_CODE, value);
}

export function isTransportationRegion(value: string): value is TransportationRegion {
  return Object.prototype.hasOwnProperty.call(REGION_NAME_BY_CODE, value);
}