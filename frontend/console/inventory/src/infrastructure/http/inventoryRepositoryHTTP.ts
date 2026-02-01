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
} from "./inventoryRepositoryHTTP.types";

// ✅ ListCreate は別ファイルに分離したため、こちらから re-export する
export type { 
  ListCreatePriceRowDTO, 
  ListCreateDTO 
} from "./listCreateRepositoryHTTP.types";

export {
  fetchInventoryListDTO,
  fetchTokenBlueprintPatchDTO,
  fetchInventoryDetailDTO,
} from "./inventoryRepositoryHTTP.fetchers";
