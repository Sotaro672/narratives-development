// frontend\amol\src\components\utils\price.ts

import { isFiniteNumber } from "./typeGuards";

export type FormatPriceOptions = {
  currency?: string;
  locale?: string;
  fallback?: string;
};

const DEFAULT_CURRENCY = "JPY";
const DEFAULT_LOCALE = "ja-JP";
const DEFAULT_FALLBACK = "価格未設定";

function normalizeCurrency(
  currency: string | undefined,
): string {
  const normalized = currency?.trim().toUpperCase();

  return normalized || DEFAULT_CURRENCY;
}

export function formatPrice(
  amount: unknown,
  options: FormatPriceOptions = {},
): string {
  const {
    locale = DEFAULT_LOCALE,
    fallback = DEFAULT_FALLBACK,
  } = options;

  if (!isFiniteNumber(amount)) {
    return fallback;
  }

  const currency = normalizeCurrency(
    options.currency,
  );

  if (currency === "JPY") {
    return `${amount.toLocaleString(locale)}円`;
  }

  return `${amount.toLocaleString(locale)} ${currency}`;
}

export function formatYen(
  amount: unknown,
  fallback = DEFAULT_FALLBACK,
): string {
  return formatPrice(amount, {
    currency: DEFAULT_CURRENCY,
    fallback,
  });
}