// frontend/amol/src/features/resale/presentation/utils/resaleDetailFormatters.ts

import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import type {
  ResaleStatus,
  ResaleListing,
} from "../../../shared/types/resale";

/**
 * 再販ステータスを表示用の文言へ変換する。
 */
export function formatResaleStatus(
  status: ResaleStatus,
): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "公開停止";
    case "sold":
      return "売却済み";
  }
}

/**
 * モデル種別を表示用の文言へ変換する。
 */
export function formatResaleModelKind(
  kind: string | undefined,
): string {
  switch (kind) {
    case "apparel":
      return "アパレル";
    case "alcohol":
      return "酒類";
    default:
      return kind || "-";
  }
}

export type ResaleModelColorDisplay = {
  label: string;
  cssColor: string;
};

/**
 * カラー情報を表示用の色名とCSSカラーへ変換する。
 *
 * rgb は 0xRRGGBB 相当の整数値として扱う。
 * 例:
 * - 16711680 -> #ff0000
 * - 65280    -> #00ff00
 * - 255      -> #0000ff
 */
export function formatResaleModelColor(
  color: ResaleListing["color"],
): ResaleModelColorDisplay | null {
  if (!color) {
    return null;
  }

  const name = textOrEmpty(color.name);
  const rgb = color.rgb;

  if (
    rgb === undefined ||
    !Number.isInteger(rgb) ||
    rgb < 0 ||
    rgb > 0xffffff
  ) {
    return name
      ? {
          label: name,
          cssColor: "",
        }
      : null;
  }

  return {
    label: name,
    cssColor: `#${rgb.toString(16).padStart(6, "0")}`,
  };
}

/**
 * 容量情報を表示用の文言へ変換する。
 */
export function formatResaleModelVolume(
  volume: ResaleListing["volume"],
): string {
  if (!volume) {
    return "-";
  }

  const amount = volume.amount;
  const unit = textOrEmpty(volume.unit);

  if (amount === undefined || amount <= 0) {
    return unit || "-";
  }

  const amountLabel = amount.toLocaleString("ja-JP");

  return unit
    ? `${amountLabel}${unit}`
    : amountLabel;
}

/**
 * 採寸情報を表示用の文言へ変換する。
 */
export function formatResaleMeasurements(
  measurements: ResaleListing["measurements"],
): string {
  if (!measurements) {
    return "-";
  }

  const entries = Object.entries(measurements)
    .filter(([key]) => Boolean(key))
    .sort(([first], [second]) =>
      first.localeCompare(second, "ja"),
    );

  if (entries.length === 0) {
    return "-";
  }

  return entries
    .map(
      ([label, value]) =>
        `${label}: ${value.toLocaleString("ja-JP")}`,
    )
    .join(" / ");
}

/**
 * 再販詳細で表示するトークンアイコンURLを返す。
 */
export function resolveResaleTokenIconUrl(
  item: ResaleListing | null | undefined,
): string {
  return item?.tokenIcon ?? "";
}