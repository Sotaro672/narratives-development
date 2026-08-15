// frontend/amol/src/features/resale/presentation/utils/resaleDetailFormatters.ts

import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import type {
  ResaleStatus,
} from "../../../shared/types/resale";

import type {
  ResaleListingWithModel,
} from "../types/resaleDetailPageTypes";

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

/**
 * カラー情報を表示用の文言へ変換する。
 */
export function formatResaleModelColor(
  color: ResaleListingWithModel["color"],
): string {
  if (!color) {
    return "-";
  }

  const name = textOrEmpty(color.name);
  const rgb = color.rgb;

  if (!name && rgb === undefined) {
    return "-";
  }

  if (rgb === undefined) {
    return name || "-";
  }

  return name
    ? `${name} / RGB: ${rgb}`
    : `RGB: ${rgb}`;
}

/**
 * 容量情報を表示用の文言へ変換する。
 */
export function formatResaleModelVolume(
  volume: ResaleListingWithModel["volume"],
): string {
  if (!volume) {
    return "-";
  }

  const amount = volume.amount;
  const unit = textOrEmpty(volume.unit);

  if (
    amount === undefined ||
    amount <= 0
  ) {
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
  measurements: ResaleListingWithModel["measurements"],
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
  item: ResaleListingWithModel | null | undefined,
): string {
  return item?.tokenIcon ?? "";
}