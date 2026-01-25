// frontend\console\inventory\src\application\inventoryDetail\inventoryDetailService.tsx
// ✅ 互換のための barrel（既存 import を壊さない）

export type {
  ProductBlueprintPatchDTOEx,
  TokenBlueprintPatchDTOEx,
  InventoryDetailViewModel,
} from "./inventoryDetail.types";

export { queryInventoryDetailByProductAndToken } from "./inventoryDetail.query";
