// frontend/amol/src/features/catalog/application/catalogCartUsecase.ts
import { addCatalogItemToCart } from "../infrastructure/catalogCartRepository";
import type {
  CatalogModelVariation,
  CatalogResponse,
} from "../types";

export async function addSelectedCatalogItemToCart(args: {
  apiBaseUrl: string;
  catalog: CatalogResponse | null;
  selectedModel: CatalogModelVariation | null;
  hasSelectedModelStock: boolean;
  isAlcoholCatalog: boolean;
}): Promise<void> {
  if (!args.catalog || !args.selectedModel) {
    throw new Error(
      args.isAlcoholCatalog
        ? "容量を選択してください。"
        : "カラーとサイズを選択してください。",
    );
  }

  if (!args.hasSelectedModelStock) {
    throw new Error("選択した商品の在庫がありません。");
  }

  await addCatalogItemToCart({
    apiBaseUrl: args.apiBaseUrl,
    catalog: args.catalog,
    selectedModel: args.selectedModel,
  });
}