//frontend\console\list\src\infrastructure\payload\listPricePayload.ts
import type { CreateListInput } from "../dto/createListInput";
import type { UpdateListInput } from "../dto/updateListInput";

export function normalizePricesForBackend(
  rows: CreateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r) => {
    const modelId = String((r as any)?.modelId ?? "");
    const priceMaybe = (r as any)?.price;

    if (!modelId) {
      throw new Error("missing_modelId_in_priceRows");
    }

    if (priceMaybe === null || priceMaybe === undefined || priceMaybe === "") {
      throw new Error("missing_price_in_priceRows");
    }

    return { modelId, price: Number(priceMaybe) };
  });
}

export function normalizePricesForBackendUpdate(
  rows: UpdateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r, idx) => {
    const modelId = String((r as any)?.modelId ?? "");
    const priceMaybe = (r as any)?.price;

    if (!modelId) {
      throw new Error(`missing_modelId_in_priceRows_at_${idx}`);
    }

    if (priceMaybe === null || priceMaybe === undefined || priceMaybe === "") {
      throw new Error(`missing_price_in_priceRows_at_${idx}`);
    }

    return { modelId, price: Number(priceMaybe) };
  });
}