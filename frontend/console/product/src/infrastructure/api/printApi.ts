// frontend/console/product/src/infrastructure/api/printApi.ts

import {
  createProductHTTP,
  createPrintLogsHTTP,
  fetchPrintLogsByProductionId,
  fetchProductsByProductionId,
} from "../repository/productRepositoryHTTP";

// 印刷用の行型（ProductionDetail 画面側から渡す）
export type PrintRow = {
  modelId?: string;
  quantity: number | null | undefined;
};

/** products 一覧の簡易型（ダイアログ表示用 + QR ラベル用） */
export type ProductSummaryForPrint = {
  id: string;
  modelId: string;
  productionId: string;
  modelNumber?: string;
};

/** print_log 一覧の型（QR ペイロードを含めてフロントで扱う想定） */
export type PrintLogForPrint = {
  id: string;
  productionId: string;
  productIds: string[];
  printedBy: string;
  printedAt: string; 
  qrPayloads: string[];
};

/* ---------------------------------------------------------
 * print_log 取得（API ラッパ）
 * --------------------------------------------------------- */
export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const raw = await fetchPrintLogsByProductionId(id);
  if (!raw) return [];
  if (!Array.isArray(raw)) return [];

  return (raw as any[])
    .map((log: any) => {
      const productIds: string[] = Array.isArray(log.productIds)
        ? log.productIds
            .map((v: unknown) => String(v))
            .filter((v: string) => !!v)
        : [];

      const qrPayloads: string[] = Array.isArray(log.qrPayloads)
        ? log.qrPayloads
            .map((v: unknown) => String(v))
            .filter((v: string) => !!v)
        : [];

      const mapped: PrintLogForPrint = {
        id: log.id ?? log.ID ?? "",
        productionId: log.productionId ?? log.ProductionID ?? "",
        productIds,
        printedBy: log.printedBy ?? log.PrintedBy ?? "",
        printedAt:
          log.printedAt ??
          log.PrintedAt ??
          "",
        qrPayloads,
      };

      return mapped;
    })
    .filter((log) => log.id && log.productionId === id);
}

/* ---------------------------------------------------------
 * 印刷用 Product 作成 + print_log 取得
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  const printedAtISO = new Date().toISOString();

  const tasks: Promise<void>[] = [];

  rows.forEach((row) => {
    const q = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity as number)))
      : 0;

    const rawModelId = row.modelId ?? "";
    const modelId = rawModelId.trim();

    if (!modelId || q <= 0) {
      return;
    }

    for (let i = 0; i < q; i += 1) {
      tasks.push(
        createProductHTTP({
          modelId,
          productionId: id,
          printedAt: printedAtISO,
        }),
      );
    }
  });

  // 1. Product を全件作成
  await Promise.all(tasks);

  // 2. print_log 作成
  await createPrintLogsHTTP(id);

  // 3. print_log を再取得
  const logs = await listPrintLogsByProductionId(id);

  return logs;
}

/* ---------------------------------------------------------
 * Products 一覧取得
 * --------------------------------------------------------- */
export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const raw = await fetchProductsByProductionId(id);
  if (!raw) return [];
  if (!Array.isArray(raw)) return [];

  const mapped = (raw as any[])
    .map((p) => ({
      id: p.id ?? p.ID ?? "",
      modelId: p.modelId ?? p.ModelID ?? "",
      productionId: p.productionId ?? p.ProductionID ?? "",
      modelNumber: p.modelNumber ?? p.ModelNumber ?? "",
    }))
    .filter((p) => p.id && p.productionId === id);

  return mapped;
}

/* ---------------------------------------------------------
 * inspection バッチ作成（現状は何もしない）
 * --------------------------------------------------------- */
export async function createInspectionBatchForProduction(
  productionId: string,
): Promise<void> {
  const id = productionId.trim();
  if (!id) {
    throw new Error("productionId is required");
  }
  return;
}
