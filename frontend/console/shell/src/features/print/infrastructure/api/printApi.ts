// frontend/console/product/src/infrastructure/api/printApi.ts

import {
  createPrintLogsHTTP,
  fetchPrintLogsByProductionId,
  type PrintLogForPrint,
} from "../repository/productRepositoryHTTP";

export type { PrintLogForPrint };

/**
 * GET /products/print-logs?productionId={productionId}
 *
 * backend の PrintQueryService が返す BFF response をそのまま正として扱う。
 */
export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];

  return fetchPrintLogsByProductionId(productionId);
}

/**
 * POST /products/print-logs
 *
 * backend 側で以下をまとめて実行する:
 * - production.models から products 作成
 * - print_log 作成
 * - inspections 作成
 * - productions.printed = true
 * - qrPayload / modelNumber を含む印刷用 BFF response を生成
 *
 * POST response 自体が完成形なので、作成後の再 GET は行わない。
 */
export async function createProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  const log = await createPrintLogsHTTP(productionId);

  return [log];
}