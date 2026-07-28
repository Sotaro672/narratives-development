// frontend/console/shell/src/features/list/infrastructure/payload/listPricePayload.ts

import type { CreateListInput } from "../dto/createListInput";
import type { UpdateListInput } from "../dto/updateListInput";

type PriceRows =
  | CreateListInput["priceRows"]
  | UpdateListInput["priceRows"];

type ListPricePayload = {
  modelId: string;
  price: number;
};

export function normalizePricesForBackend(
  rows: PriceRows,
): ListPricePayload[] {
  return (rows ?? []).map(({ modelId, price }, index) => {
    if (!modelId) {
      throw new Error(`missing_modelId_in_priceRows_at_${index}`);
    }

    if (price === null) {
      throw new Error(`missing_price_in_priceRows_at_${index}`);
    }

    return {
      modelId,
      price,
    };
  });
}