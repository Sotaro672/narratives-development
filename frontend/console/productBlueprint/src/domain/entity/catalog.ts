// frontend/console/productBlueprint/src/domain/entity/catalog.ts

// ============================
// フィット種別
// ============================
export type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

// UI 用フィット選択肢
export const FIT_OPTIONS: { value: Fit; label: string }[] = [
  { value: "レギュラーフィット", label: "レギュラーフィット" },
  { value: "スリムフィット", label: "スリムフィット" },
  { value: "リラックスフィット", label: "リラックスフィット" },
  { value: "オーバーサイズ", label: "オーバーサイズ" },
];

// ============================
// 商品IDタグ選択肢
// ============================

export const PRODUCT_ID_TAG_OPTIONS: { value: string; label: string }[] = [
  { value: "QRコード", label: "QRコード" },
  { value: "バーコード", label: "バーコード" },
];

// ============================
// 品質保証（洗濯タグ）
// 6つのカテゴリー階層付き
// ============================

export type WashTagCategory =
  | "洗濯"
  | "漂白"
  | "乾燥"
  | "アイロン"
  | "ドライクリーニング"
  | "ウェットクリーニング";

export type WashTagOption = {
  /** 保存用の値（qualityAssurance[] に入る想定） */
  value: string;
  /** UI 表示ラベル */
  label: string;
  /** 6分類のカテゴリー */
  category: WashTagCategory;
};

export const WASH_TAG_OPTIONS: WashTagOption[] = [
  // ──────────────
  // 洗濯
  // ──────────────
  { category: "洗濯", value: "手洗い", label: "手洗い" },
  { category: "洗濯", value: "洗濯機可", label: "洗濯機可" },
  { category: "洗濯", value: "弱い洗濯", label: "弱い洗濯" },
  { category: "洗濯", value: "液温30℃限度", label: "液温30℃限度" },
  { category: "洗濯", value: "液温40℃限度", label: "液温40℃限度" },
  { category: "洗濯", value: "水洗い不可", label: "水洗い不可" },

  // ──────────────
  // 漂白
  // ──────────────
  { category: "漂白", value: "酸素系漂白可", label: "酸素系漂白可" },
  { category: "漂白", value: "塩素系漂白可", label: "塩素系漂白可" },
  { category: "漂白", value: "漂白不可", label: "漂白不可" },

  // ──────────────
  // 乾燥
  // ──────────────
  { category: "乾燥", value: "タンブル乾燥可 低温", label: "タンブル乾燥可（低温）" },
  { category: "乾燥", value: "タンブル乾燥可 中温", label: "タンブル乾燥可（中温）" },
  { category: "乾燥", value: "タンブル乾燥不可", label: "タンブル乾燥不可" },
  { category: "乾燥", value: "つり干し", label: "つり干し" },
  { category: "乾燥", value: "日陰つり干し", label: "日陰つり干し" },
  { category: "乾燥", value: "平干し", label: "平干し" },
  { category: "乾燥", value: "日陰平干し", label: "日陰平干し" },

  // ──────────────
  // アイロン
  // ──────────────
  { category: "アイロン", value: "アイロン低温", label: "アイロン低温（110℃まで）" },
  { category: "アイロン", value: "アイロン中温", label: "アイロン中温（150℃まで）" },
  { category: "アイロン", value: "アイロン高温", label: "アイロン高温（200℃まで）" },
  { category: "アイロン", value: "アイロン不可", label: "アイロン不可" },

  // ──────────────
  // ドライクリーニング
  // ──────────────
  {
    category: "ドライクリーニング",
    value: "ドライクリーニング可",
    label: "ドライクリーニング可",
  },
  {
    category: "ドライクリーニング",
    value: "石油系ドライ可",
    label: "石油系ドライクリーニング可",
  },
  {
    category: "ドライクリーニング",
    value: "ドライクリーニング不可",
    label: "ドライクリーニング不可",
  },

  // ──────────────
  // ウェットクリーニング
  // ──────────────
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング可",
    label: "ウェットクリーニング可",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング弱",
    label: "ウェットクリーニング（弱）",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング非常に弱",
    label: "ウェットクリーニング（非常に弱）",
  },
  {
    category: "ウェットクリーニング",
    value: "ウェットクリーニング不可",
    label: "ウェットクリーニング不可",
  },
];
