// frontend/amol/src/features/wallet/utils/format.ts
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