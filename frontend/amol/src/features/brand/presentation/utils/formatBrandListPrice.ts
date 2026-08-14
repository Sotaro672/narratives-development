// frontend/amol/src/features/brand/presentation/utils/formatBrandListPrice.ts

import { formatPrice } from "../../../../components/utils/price";
import type { ListPriceRow } from "../../types/brand";

export function formatBrandListPrice(prices: ListPriceRow[]): string {
  if (prices.length === 0) {
    return formatPrice(undefined);
  }

  return formatPrice(prices[0]?.price);
}