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

/** print_log の items 要素 */
export type PrintedItemForPrint = {
  productId: string;
  displayOrder: number;
};

/** print_log 一覧の型（QR ペイロードを含めてフロントで扱う想定） */
export type PrintLogForPrint = {
  id: string;
  productionId: string;

  // ✅ 正: items（displayOrder を保持）
  items: PrintedItemForPrint[];

  // ✅ usecase が付与して返す想定（Firestoreには保存しない）
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
  if (!raw || !Array.isArray(raw)) return [];

  return (raw as any[])
    .map((log: any) => {
      const rawItems = Array.isArray(log.items) ? log.items : [];

      const items: PrintedItemForPrint[] = rawItems
        .map((it: any) => {
          const productId = String(it?.productId ?? "");
          const displayOrderNum = Number(it?.displayOrder);
          const displayOrder = Number.isFinite(displayOrderNum)
            ? displayOrderNum
            : 0;
          return { productId, displayOrder };
        })
        .filter((it: PrintedItemForPrint) => it.productId !== "");

      const qrPayloads: string[] = Array.isArray(log.qrPayloads)
        ? log.qrPayloads.map((v: unknown) => String(v)).filter((v: string) => !!v)
        : [];

      const mapped: PrintLogForPrint = {
        id: log.id ?? log.ID ?? "",
        productionId: log.productionId ?? log.ProductionID ?? "",
        items,
        qrPayloads,
      };

      return mapped;
    })
    .filter((log) => log.id && log.productionId === id);
}

/* ---------------------------------------------------------
 * 印刷用 Product 作成 + print_log 取得
 *   1. Product を rows の順で逐次作成
 *   2. print_log を作成
 *   3. listPrintLogsByProductionId で結果を取得
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  const printedAtISO = new Date().toISOString();

  // rows の並び順を壊さないため、逐次作成する
  for (const row of rows) {
    const q = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity as number)))
      : 0;

    const rawModelId = row.modelId ?? "";
    const modelId = rawModelId.trim();

    if (!modelId || q <= 0) continue;

    for (let i = 0; i < q; i += 1) {
      await createProductHTTP({
        modelId,
        productionId: id,
        printedAt: printedAtISO,
      });
    }
  }

  await createPrintLogsHTTP(id);

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
  if (!raw || !Array.isArray(raw)) return [];

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