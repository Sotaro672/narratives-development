// frontend/admin/shell/src/shared/util/modelMetaFormat.ts

import type { ContractListPriceRow } from "../type/contractListDetail";

export function formatModelMeta(price: ContractListPriceRow): string {
  const values: string[] = [];

  if (price.modelNumber) {
    values.push(price.modelNumber);
  }

  if (price.kind === "apparel") {
    if (price.size) {
      values.push(price.size);
    }
    if (price.color) {
      values.push(price.color);
    }
  }

  if (price.kind === "alcohol" && price.volumeValue != null) {
    values.push(`${price.volumeValue}${price.volumeUnit || ""}`);
  }

  return values.length > 0 ? values.join(" / ") : "モデル情報なし";
}