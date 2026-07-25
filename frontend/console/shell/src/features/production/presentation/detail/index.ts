//frontend\console\production\src\presentation\detail\index.ts
export type {
  ProductionDetail,
  ProductionQuantityRow,
  ModelVariationSummary,
} from "../../application/detail/types";

export {
  loadProductionDetail,
} from "../../application/detail/loadProductionDetail";
export {
  loadModelVariationIndexByProductBlueprintId,
  buildModelIndexFromVariations,
} from "../../application/detail/buildModelVariationIndex";
export {
  buildQuantityRowsFromModels,
} from "../../application/detail/buildQuantityRows";

export {
  updateProductionDetail,
} from "../../application/detail/updateProductionDetail";
export {
  notifyPrintLogCompleted,
} from "../../application/detail/notifyPrintLogCompleted";