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

  const logs = await listPrintLogsByProductionIdApi(id);
  return logs;
}

export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const products = await listProductsByProductionIdApi(id);
  return products;
}

function buildProductLabelMap(
  products: ProductSummaryForPrint[] | undefined,
): Map<string, string> {
  const map = new Map<string, string>();
  if (!products || products.length === 0) return map;

  for (const p of products) {
    const productId = p.id;
    if (!productId) continue;

    const modelNumber = (p.modelNumber ?? "").trim();
    if (!modelNumber) continue;

    map.set(productId, modelNumber);
  }

  return map;
}

type PrintLogItem = {
  productId: string;
  displayOrder: number;
};

type SortedPrintTarget = {
  productId: string;
  displayOrder: number;
  payload: string;
  originalIndex: number;
};

// items と qrPayloads をペアで保持したまま displayOrder 順に並べる
function getSortedPrintTargets(log: any): SortedPrintTarget[] {
  const items: any[] = Array.isArray(log?.items) ? log.items : [];
  const payloads: any[] = Array.isArray(log?.qrPayloads) ? log.qrPayloads : [];

  const paired: SortedPrintTarget[] = items
    .map((raw: any, index: number) => {
      const productId = String(raw?.productId ?? "");
      const displayOrderNum = Number(raw?.displayOrder);
      const displayOrder = Number.isFinite(displayOrderNum)
        ? displayOrderNum
        : Number.MAX_SAFE_INTEGER;
      const payload = String(payloads[index] ?? "");

      return {
        productId,
        displayOrder,
        payload,
        originalIndex: index,
      };
    })
    .filter((x) => x.productId !== "" && x.payload !== "");

  // displayOrder のみでソートし、同値なら Firestore 配列順を維持
  paired.sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }
    return a.originalIndex - b.originalIndex;
  });

  return paired;
}

async function buildAndOpenQrPdfFromLogs(
  logs: PrintLogForPrint[],
  productionId: string,
  products?: ProductSummaryForPrint[],
): Promise<number> {
  const qrItems: QrPdfItem[] = [];
  const productLabelMap = buildProductLabelMap(products);

  (Array.isArray(logs) ? logs : []).forEach((log: any) => {
    const sortedTargets = getSortedPrintTargets(log);

    sortedTargets.forEach((target) => {
      const label = productLabelMap.get(target.productId);

      qrItems.push({
        payload: target.payload,
        label,
      });
    });
  });

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

export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  const existingLogs = await listPrintLogsByProductionIdApi(id);

  if (existingLogs.length > 0) {
    const products = await listProductsByProductionId(id);

    const totalQrCount = await buildAndOpenQrPdfFromLogs(
      existingLogs,
      id,
      products,
    );

    await notifyPrintLogCompleted({
      productionId: id,
      logCount: existingLogs.length,
      totalQrCount,
      reusedExistingLogs: true,
    });

    return existingLogs;
  }

  const logs = await createProductsForPrintApi({
    productionId: id,
    rows,
  });

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

  const totalQrCount = await buildAndOpenQrPdfFromLogs(logs, id, products);

  await notifyPrintLogCompleted({
    productionId: id,
    logCount: logs.length,
    totalQrCount,
    reusedExistingLogs: false,
  });

  return logs;
}