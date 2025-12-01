// frontend/console/product/src/application/printService.tsx

// このファイルは、アプリケーション層からインフラ層 API を再エクスポートする薄いラッパです。
// 既存の import パスを壊さずに、api 要素を infrastructure/api/printApi.ts に分離しています。

export type {
  PrintRow,
  ProductSummaryForPrint,
  PrintLogForPrint,
} from "../infrastructure/api/printApi";

export {
  listPrintLogsByProductionId,
  createProductsForPrint,
  listProductsByProductionId,
} from "../infrastructure/api/printApi";
