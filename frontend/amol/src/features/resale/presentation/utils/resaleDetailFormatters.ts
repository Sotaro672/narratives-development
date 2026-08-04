// frontend/amol/src/features/resale/presentation/utils/resaleDetailFormatters.ts

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import type {
  ResaleListingWithModel,
  ResaleModelColor,
  ResaleModelVolume,
} from "../types/resaleDetailPageTypes";

/**
 * 再販ステータスを表示用の文言へ変換する。
 */
export function formatResaleStatus(
  value: unknown,
): string {
  const status =
    textOrEmpty(value);

  switch (status) {
    case "listing":
      return "出品中";

    case "suspended":
      return "公開停止";

    case "sold":
      return "売却済み";

    default:
      return status || "-";
  }
}

/**
 * モデル種別を表示用の文言へ変換する。
 */
export function formatResaleModelKind(
  value: unknown,
): string {
  const kind =
    textOrEmpty(value);

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
  color:
    | ResaleModelColor
    | null
    | undefined,
): string {
  if (!color) {
    return "-";
  }

  const name =
    textOrEmpty(
      color.name,
    );

  const rgb =
    Number(
      color.rgb,
    );

  const hasRgb =
    Number.isFinite(
      rgb,
    );

  if (
    !name &&
    !hasRgb
  ) {
    return "-";
  }

  if (!hasRgb) {
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
  volume:
    | ResaleModelVolume
    | null
    | undefined,
): string {
  if (!volume) {
    return "-";
  }

  const amount =
    Number(
      volume.amount ??
        volume.value ??
        0,
    );

  const unit =
    textOrEmpty(
      volume.unit,
    );

  if (
    !Number.isFinite(
      amount,
    ) ||
    amount <= 0
  ) {
    return unit || "-";
  }

  const amountLabel =
    amount.toLocaleString(
      "ja-JP",
    );

  return unit
    ? `${amountLabel}${unit}`
    : amountLabel;
}

/**
 * 採寸情報を表示用の文言へ変換する。
 */
export function formatResaleMeasurements(
  measurements:
    | Record<string, number>
    | null
    | undefined,
): string {
  if (!measurements) {
    return "-";
  }

  const entries =
    Object.entries(
      measurements,
    )
      .map(
        ([
          key,
          value,
        ]) => ({
          label:
            textOrEmpty(
              key,
            ),
          value:
            Number(
              value,
            ),
        }),
      )
      .filter(
        ({
          label,
          value,
        }) =>
          Boolean(label) &&
          Number.isFinite(
            value,
          ),
      )
      .sort(
        (
          first,
          second,
        ) =>
          first.label.localeCompare(
            second.label,
            "ja",
          ),
      );

  if (
    entries.length === 0
  ) {
    return "-";
  }

  return entries
    .map(
      ({
        label,
        value,
      }) =>
        `${label}: ${value.toLocaleString("ja-JP")}`,
    )
    .join(" / ");
}

/**
 * 再販詳細で表示するトークン画像URLを解決する。
 *
 * 優先順位:
 * 1. tokenIconUrl
 * 2. tokenIcon
 * 3. imageUrl
 * 4. metadata.image
 */
export function resolveResaleTokenIconUrl(
  item:
    | ResaleListingWithModel
    | null
    | undefined,
): string {
  if (!item) {
    return "";
  }

  return (
    textOrEmpty(
      item.tokenIconUrl,
    ) ||
    textOrEmpty(
      item.tokenIcon,
    ) ||
    textOrEmpty(
      item.imageUrl,
    ) ||
    textOrEmpty(
      item.metadata?.image,
    )
  );
}