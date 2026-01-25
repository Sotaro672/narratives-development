// frontend/console/inventory/src/application/inventoryDetailService.tsx
// ✅ 互換のための barrel（既存 import を壊さない）

export type {
  ProductBlueprintPatchDTOEx,
  TokenBlueprintPatchDTOEx,
  InventoryDetailViewModel,
} from "./inventoryDetail/inventoryDetail.types";

export { queryInventoryDetailByProductAndToken } from "./inventoryDetail/inventoryDetail.query";
