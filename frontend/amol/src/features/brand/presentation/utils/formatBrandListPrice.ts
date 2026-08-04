// frontend/amol/src/features/brand/presentation/utils/formatBrandListPrice.ts

import {
  formatPrice,
} from "../../../../components/utils/price";

import type {
  ListPriceRow,
} from "../../types/brand";

export function formatBrandListPrice(
  prices: ListPriceRow[],
): string {
  if (
    !Array.isArray(prices) ||
    prices.length === 0
  ) {
    return formatPrice(undefined);
  }

  const firstPrice =
    prices[0];

  const amount =
    firstPrice?.amount ??
    firstPrice?.price;

  return formatPrice(
    amount,
    {
      currency:
        firstPrice?.currency,
    },
  );
}