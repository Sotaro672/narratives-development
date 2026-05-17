// frontend/amol/src/features/wallet/utils/format.ts
import type { WalletOrderItemSnapshot } from "../types/orderTypes";

export function formatAmount(amount: number, currency?: string): string {
  const normalizedAmount = Number.isFinite(amount) ? amount : 0;
  const normalizedCurrency = (currency || "JPY").toUpperCase();

  try {
    return new Intl.NumberFormat("ja-JP", {
      style: "currency",
      currency: normalizedCurrency,
      maximumFractionDigits: 0,
    }).format(normalizedAmount);
  } catch {
    return `${normalizedAmount.toLocaleString("ja-JP")} ${normalizedCurrency}`;
  }
}

export function formatWalletOrderItemVolume(
  item: WalletOrderItemSnapshot,
): string {
  if (
    typeof item.volumeValue === "number" &&
    Number.isFinite(item.volumeValue) &&
    item.volumeUnit
  ) {
    return `${item.volumeValue}${item.volumeUnit}`;
  }

  return "";
}

export function formatWalletOrderItemModelLabel(
  item: WalletOrderItemSnapshot,
): string {
  if (item.kind === "alcohol") {
    const volume = formatWalletOrderItemVolume(item);

    return [item.modelNumber, volume].filter(Boolean).join(" / ");
  }

  const colorName = item.color?.name ?? "";
  const size = item.size ?? "";

  return [
    item.modelNumber ? `品番: ${item.modelNumber}` : "",
    size ? `サイズ: ${size}` : "",
    colorName ? `色: ${colorName}` : "",
  ]
    .filter(Boolean)
    .join("　");
}

export function getWalletOrderItemMetaEntries(
  item: WalletOrderItemSnapshot,
): string[] {
  if (item.kind === "alcohol") {
    return [
      item.modelNumber ? `品番: ${item.modelNumber}` : "",
      formatWalletOrderItemVolume(item)
        ? `容量: ${formatWalletOrderItemVolume(item)}`
        : "",
    ].filter(Boolean);
  }

  return [
    item.modelNumber ? `品番: ${item.modelNumber}` : "",
    item.size ? `サイズ: ${item.size}` : "",
    item.color?.name ? `色: ${item.color.name}` : "",
  ].filter(Boolean);
}