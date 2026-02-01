// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.ts
// ✅ API_BASE 互換が必要な箇所があるかもしれないので re-export（任意）
export { API_BASE } from "../../../../shell/src/shared/http/apiBase";

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
  fetchTokenBlueprintPatchDTO,
  fetchListCreateDTO,
  fetchInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.fetchers";
