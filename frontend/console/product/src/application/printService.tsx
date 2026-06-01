// frontend/console/product/src/application/printService.tsx

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  listProductsByProductionId as listProductsByProductionIdApi,
  type PrintRow,
  type ProductSummaryForPrint,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";

import {
  buildQrPdfBlobA4,
  openQrPdfInNewTab,
  type QrPdfItem,
} from "../utils/qrPdfBuilder";

import { notifyPrintLogCompleted } from "../../../production/src/application/detail/notifyPrintLogCompleted";

export type { PrintRow, ProductSummaryForPrint, PrintLogForPrint };

export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  return listPrintLogsByProductionIdApi(id);
}

export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  return listProductsByProductionIdApi(id);
}

function buildModelNumberByModelIdMap(
  rows: PrintRow[] | undefined,
): Map<string, string> {
  const map = new Map<string, string>();

  for (const row of Array.isArray(rows) ? rows : []) {
    const modelId = String(row.modelId ?? "").trim();
    const modelNumber = String(row.modelNumber ?? "").trim();

    if (!modelId || !modelNumber) {
      continue;
    }

    map.set(modelId, modelNumber);
  }

  return map;
}

function buildProductLabelMap(
  products: ProductSummaryForPrint[] | undefined,
  rows: PrintRow[] | undefined,
): Map<string, string> {
  const map = new Map<string, string>();
  const modelNumberByModelId = buildModelNumberByModelIdMap(rows);

  for (const product of Array.isArray(products) ? products : []) {
    const productId = String(product.id ?? "").trim();
    const modelId = String(product.modelId ?? "").trim();

    if (!productId) {
      continue;
    }

    const modelNumber =
      String(product.modelNumber ?? "").trim() ||
      modelNumberByModelId.get(modelId) ||
      "";

    if (!modelNumber) {
      continue;
    }

    map.set(productId, modelNumber);
  }

  return map;
}

type SortedPrintTarget = {
  productId: string;
  displayOrder: number;
  payload: string;
  originalIndex: number;
};

// items と qrPayloads をペアで保持したまま displayOrder 順に並べる
function getSortedPrintTargets(log: PrintLogForPrint): SortedPrintTarget[] {
  const items = Array.isArray(log.items) ? log.items : [];
  const payloads = Array.isArray(log.qrPayloads) ? log.qrPayloads : [];

  const paired: SortedPrintTarget[] = items
    .map((item, index) => {
      const productId = String(item.productId ?? "").trim();

      const displayOrderNum = Number(item.displayOrder);
      const displayOrder = Number.isFinite(displayOrderNum)
        ? displayOrderNum
        : Number.MAX_SAFE_INTEGER;

      const payload = String(payloads[index] ?? "").trim();

      return {
        productId,
        displayOrder,
        payload,
        originalIndex: index,
      };
    })
    .filter((target) => target.productId !== "" && target.payload !== "");

  // displayOrder のみでソートし、同値なら Firestore 配列順を維持
  paired.sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }

    return a.originalIndex - b.originalIndex;
  });

  return paired;
}

async function buildAndOpenQrPdfFromLogs(args: {
  logs: PrintLogForPrint[];
  products?: ProductSummaryForPrint[];
  rows?: PrintRow[];
}): Promise<number> {
  const { logs, products, rows } = args;

  const qrItems: QrPdfItem[] = [];
  const productLabelMap = buildProductLabelMap(products, rows);

  for (const log of Array.isArray(logs) ? logs : []) {
    const sortedTargets = getSortedPrintTargets(log);

    for (const target of sortedTargets) {
      const label = productLabelMap.get(target.productId) ?? "";

      qrItems.push({
        payload: target.payload,
        label,
      });
    }
  }

  if (qrItems.length === 0) {
    return 0;
  }

  const pdfBlob = await buildQrPdfBlobA4(qrItems, {
    cols: 5,
    cellHeight: 100,
  });

  openQrPdfInNewTab(pdfBlob);

  return qrItems.length;
}

/**
 * 既存 print_log を取得し、存在する場合は GET 結果だけで QR PDF を開く。
 *
 * この関数では作成系 API は呼ばない。
 */
export async function printExistingLogsForProduction(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();

  if (!id) {
    throw new Error("productionId is required");
  }

  const safeRows = Array.isArray(rows) ? rows : [];

  const logs = await listPrintLogsByProductionIdApi(id);

  if (logs.length === 0) {
    return [];
  }

  const products = await listProductsByProductionId(id);

  const totalQrCount = await buildAndOpenQrPdfFromLogs({
    logs,
    products,
    rows: safeRows,
  });

  await notifyPrintLogCompleted({
    productionId: id,
    logCount: logs.length,
    totalQrCount,
    reusedExistingLogs: true,
  });

  return logs;
}

/**
 * 初回印刷用。
 *
 * products / print_log がまだ無い productionId に対してのみ使う。
 * 既存 print_log の再印刷用途では printExistingLogsForProduction または
 * printOrCreateProductsForPrint を使う。
 */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();

  if (!id) {
    throw new Error("productionId is required");
  }

  const safeRows = Array.isArray(rows) ? rows : [];

  await createProductsForPrintApi({
    productionId: id,
    rows: safeRows,
  });

  const logs = await listPrintLogsByProductionIdApi(id);

  if (logs.length === 0) {
    await notifyPrintLogCompleted({
      productionId: id,
      logCount: 0,
      totalQrCount: 0,
      reusedExistingLogs: false,
    });

    return logs;
  }

  const products = await listProductsByProductionId(id);

  const totalQrCount = await buildAndOpenQrPdfFromLogs({
    logs,
    products,
    rows: safeRows,
  });

  await notifyPrintLogCompleted({
    productionId: id,
    logCount: logs.length,
    totalQrCount,
    reusedExistingLogs: false,
  });

  return logs;
}

/**
 * 印刷ボタン用の入口。
 *
 * 1. まず GET /products/print-logs?productionId=... を実行する
 * 2. 既存 print_log があれば、その GET 結果で QR PDF を開く
 * 3. 既存 print_log が無い場合だけ、初回作成として POST 系処理に進む
 *
 * GET が失敗した場合は POST にフォールバックしない。
 * 認証エラーや backend エラーを隠さないため。
 */
export async function printOrCreateProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const existingLogs = await printExistingLogsForProduction(params);

  if (existingLogs.length > 0) {
    return existingLogs;
  }

  return createProductsForPrint(params);
}