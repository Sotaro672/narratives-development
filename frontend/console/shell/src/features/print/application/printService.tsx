// frontend/console/shell/src/features/print/application/printService.tsx

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  listProductsByProductionId as listProductsByProductionIdApi,
  type ProductSummaryForPrint,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";
import {
  buildQrPdfBlobA4,
  openQrPdfInNewTab,
  type QrPdfItem,
} from "../utils/qrPdfBuilder";

export type { ProductSummaryForPrint, PrintLogForPrint };

export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];
  return listPrintLogsByProductionIdApi(productionId);
}

export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  if (!productionId) return [];
  return listProductsByProductionIdApi(productionId);
}

function buildProductLabelMap(
  products: ProductSummaryForPrint[],
): Map<string, string> {
  return new Map(
    products.map((product) => [product.id, product.modelNumber]),
  );
}

async function buildAndOpenQrPdfFromLogs(
  logs: PrintLogForPrint[],
  products: ProductSummaryForPrint[],
): Promise<void> {
  const productLabelMap = buildProductLabelMap(products);
  const qrItems: QrPdfItem[] = [];

  for (const log of logs) {
    for (let index = 0; index < log.items.length; index += 1) {
      const item = log.items[index];
      const payload = log.qrPayloads[index];

      if (!payload) continue;

      qrItems.push({
        payload,
        label: productLabelMap.get(item.productId) ?? "",
      });
    }
  }

  if (qrItems.length === 0) return;

  const pdfBlob = await buildQrPdfBlobA4(qrItems, {
    cols: 5,
    cellHeight: 100,
  });

  openQrPdfInNewTab(pdfBlob);
}

/**
 * 既存 print_log を取得し、存在する場合は GET 結果だけで QR PDF を開く。
 * この関数では作成系 API は呼ばない。
 */
export async function printExistingLogsForProduction(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  const logs = await listPrintLogsByProductionIdApi(productionId);

  if (logs.length === 0) return [];

  const products = await listProductsByProductionIdApi(productionId);
  await buildAndOpenQrPdfFromLogs(logs, products);

  return logs;
}

/**
 * 初回印刷用。
 *
 * POST /products/print-logs により backend 側で products / print_log /
 * inspections の作成と productions.printed の更新をまとめて実行する。
 */
export async function createProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  const logs = await createProductsForPrintApi({
    productionId,
  });

  if (logs.length === 0) return [];

  const products = await listProductsByProductionIdApi(productionId);
  await buildAndOpenQrPdfFromLogs(logs, products);

  return logs;
}

/**
 * 印刷ボタン用の入口。
 *
 * 1. GET /products/print-logs?productionId=... で既存 print_log を確認する
 * 2. 存在する場合は既存データから QR PDF を生成する
 * 3. 存在しない場合だけ POST /products/print-logs を実行する
 *
 * GET が失敗した場合は POST にフォールバックしない。
 */
export async function printOrCreateProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const existingLogs = await printExistingLogsForProduction(params);

  if (existingLogs.length > 0) {
    return existingLogs;
  }

  return createProductsForPrint(params);
}