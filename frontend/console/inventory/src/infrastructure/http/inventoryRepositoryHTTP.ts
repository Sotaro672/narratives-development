// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.ts
// ✅ API_BASE 互換が必要な箇所があるかもしれないので re-export（任意）
export { API_BASE } from "../api/inventoryApi";

export type {
  InventoryProductSummary,
  InventoryListRowDTO,
  InventoryIDsByProductAndTokenDTO,
  TokenBlueprintSummaryDTO,
  ProductBlueprintSummaryDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
  InventoryDetailRowDTO,
  InventoryDetailDTO,
  ListCreatePriceRowDTO,
  ListCreateDTO,
} from "./inventoryRepositoryHTTP.types";

export {
  fetchInventoryListDTO,
  fetchInventoryProductSummary,
  fetchPrintedInventorySummaries,
  fetchInventoryIDsByProductAndTokenDTO,
  fetchTokenBlueprintPatchDTO,
  fetchListCreateDTO,
  fetchInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.fetchers";
