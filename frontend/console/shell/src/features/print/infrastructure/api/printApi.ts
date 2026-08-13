// frontend/console/product/src/infrastructure/api/printApi.ts

import {
  createPrintLogsHTTP,
  fetchPrintLogsByProductionId,
  fetchProductsByProductionId,
} from "../repository/productRepositoryHTTP";

/** products 一覧の印刷用 BFF response */
export type ProductSummaryForPrint = {
  id: string;
  modelId: string;
  productionId: string;
  modelNumber: string;
};

/** print_log の items 要素 */
export type PrintedItemForPrint = {
  productId: string;
  displayOrder: number;
};

/** print_log の印刷用 BFF response */
export type PrintLogForPrint = {
  id: string;
  productionId: string;
  items: PrintedItemForPrint[];
  qrPayloads: string[];
};

/**
 * GET /products/print-logs?productionId={productionId}
 *
 * backend の PrintQueryService が返す BFF response をそのまま正として扱う。
 */
export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];

  return fetchPrintLogsByProductionId(
    productionId,
  ) as Promise<PrintLogForPrint[]>;
}

/**
 * POST /products/print-logs
 *
 * backend 側で以下をまとめて実行する:
 * - production.models から products 作成
 * - print_log 作成
 * - inspections 作成
 * - productions.printed = true
 */
export async function createProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  await createPrintLogsHTTP(productionId);

  return listPrintLogsByProductionId(productionId);
}

/**
 * GET /products?productionId={productionId}
 *
 * backend の PrintQueryService が modelNumber まで解決した BFF response を
 * そのまま正として扱う。
 */
export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  if (!productionId) return [];

  return fetchProductsByProductionId(
    productionId,
  ) as Promise<ProductSummaryForPrint[]>;
}