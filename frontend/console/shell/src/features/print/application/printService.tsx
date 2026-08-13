// frontend/console/shell/src/features/print/application/printService.tsx

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";
import {
  buildQrPdfBlobA4,
  openQrPdfInNewTab,
  type QrPdfItem,
} from "../utils/qrPdfBuilder";

export type { PrintLogForPrint };

export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];
  return listPrintLogsByProductionIdApi(productionId);
}

async function buildAndOpenQrPdfFromLogs(
  logs: PrintLogForPrint[],
): Promise<void> {
  const qrItems: QrPdfItem[] = [];

  for (const log of logs) {
    for (const item of log.items) {
      qrItems.push({
        payload: item.qrPayload,
        label: item.modelNumber,
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

  await buildAndOpenQrPdfFromLogs(logs);

  return logs;
}

/**
 * 初回印刷用。
 *
 * POST /products/print-logs により backend 側で products / print_log /
 * inspections の作成、productions.printed の更新、
 * QR PDF に必要な qrPayload / modelNumber の解決まで行う。
 */
export async function createProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  const logs = await createProductsForPrintApi({ productionId });

  if (logs.length === 0) return [];

  await buildAndOpenQrPdfFromLogs(logs);

  return logs;
}

/**
 * 印刷ボタン用の入口。
 *
 * 1. GET /products/print-logs?productionId=... で既存 print_log を確認する
 * 2. 存在する場合は既存 BFF response から QR PDF を生成する
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