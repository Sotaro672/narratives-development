// frontend\amol\src\features\shared\presentation\utils\productModelDisplay.ts

import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

export type ProductModelColor = {
  name?: string;
  rgb?: number;
};

export type ProductModelVolume = {
  amount?: number;
  unit?: string;
};

export type ProductModelMeasurements = Record<string, number>;

export type ProductModelDisplay = {
  hasModelInfo: boolean;
  kindLabel: string;
  modelNumber: string;
  size: string;
  colorLabel: string;
  colorCssValue: string;
  measurementsLabel: string;
  volumeLabel: string;
};

export type ProductModelDisplaySource = {
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: ProductModelColor | null;
  measurements?: ProductModelMeasurements | null;
  volume?: ProductModelVolume | null;
};

function formatModelKind(kind: string | undefined): string {
  switch (textOrEmpty(kind)) {
    case "apparel":
      return "アパレル";
    case "alcohol":
      return "酒類";
    default:
      return textOrEmpty(kind) || "-";
  }
}

function formatModelColor(
  color: ProductModelColor | null | undefined,
): {
  label: string;
  cssColor: string;
} | null {
  if (!color) {
    return null;
  }

  const label = textOrEmpty(color.name);
  const rgb = color.rgb;

  if (
    rgb === undefined ||
    !Number.isInteger(rgb) ||
    rgb < 0 ||
    rgb > 0xffffff
  ) {
    return label ? { label, cssColor: "" } : null;
  }

  return {
    label,
    cssColor: `#${rgb.toString(16).padStart(6, "0")}`,
  };
}

function formatModelVolume(
  volume: ProductModelVolume | null | undefined,
): string {
  if (!volume) {
    return "-";
  }

  const amount = Number(volume.amount);
  const unit = textOrEmpty(volume.unit);

  if (!Number.isFinite(amount) || amount <= 0) {
    return unit || "-";
  }

  const amountLabel = amount.toLocaleString("ja-JP");
  return unit ? `${amountLabel}${unit}` : amountLabel;
}

function formatMeasurements(
  measurements: ProductModelMeasurements | null | undefined,
): string {
  if (!measurements) {
    return "-";
  }

  const entries = Object.entries(measurements)
    .filter(([key, value]) => {
      return textOrEmpty(key) !== "" && Number.isFinite(Number(value));
    })
    .sort(([first], [second]) => first.localeCompare(second, "ja"));

  if (entries.length === 0) {
    return "-";
  }

  return entries
    .map(([label, value]) => `${label}: ${Number(value).toLocaleString("ja-JP")}cm`)
    .join(" / ");
}

export function createProductModelDisplay(
  item: ProductModelDisplaySource | null | undefined,
): ProductModelDisplay {
  const modelId = textOrEmpty(item?.modelId);
  const kind = textOrEmpty(item?.kind);
  const kindLabel = kind ? formatModelKind(kind) : "";
  const modelNumber = textOrEmpty(item?.modelNumber);
  const size = textOrEmpty(item?.size);

  const color = formatModelColor(item?.color);
  const colorLabel = color?.label ?? "";
  const colorCssValue = color?.cssColor ?? "";

  const measurementsLabel = formatMeasurements(item?.measurements);
  const volumeLabel = formatModelVolume(item?.volume);

  const hasModelInfo =
    Boolean(modelId) ||
    Boolean(kindLabel) ||
    Boolean(modelNumber) ||
    Boolean(size) ||
    Boolean(colorLabel) ||
    Boolean(colorCssValue) ||
    measurementsLabel !== "-" ||
    volumeLabel !== "-";

  return {
    hasModelInfo,
    kindLabel,
    modelNumber,
    size,
    colorLabel,
    colorCssValue,
    measurementsLabel,
    volumeLabel,
  };
}