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

  /**
   * QR ラベル表示用。
   *
   * products response の modelNumber が空の場合でも、
   * frontend が保持している modelNumber を PDF ラベル生成に使う。
   */
  modelNumber?: string;
};

/** products 一覧の簡易型（ダイアログ表示用 + QR ラベル用） */
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

/** print_log 一覧の型（QR ペイロードを含めてフロントで扱う） */
export type PrintLogForPrint = {
  id: string;
  productionId: string;
  items: PrintedItemForPrint[];
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
  if (!Array.isArray(raw)) return [];

  return raw
    .map((log: any): PrintLogForPrint => {
      const rawItems = Array.isArray(log.items) ? log.items : [];

      const items: PrintedItemForPrint[] = rawItems
        .map((item: any): PrintedItemForPrint => {
          const productId = String(item?.productId ?? "").trim();
          const displayOrderNum = Number(item?.displayOrder);
          const displayOrder = Number.isFinite(displayOrderNum)
            ? displayOrderNum
            : 0;

          return {
            productId,
            displayOrder,
          };
        })
        .filter((item: PrintedItemForPrint) => item.productId !== "");

      const qrPayloads: string[] = Array.isArray(log.qrPayloads)
        ? log.qrPayloads
            .map((value: unknown) => String(value).trim())
            .filter((value: string) => value !== "")
        : [];

      return {
        id: String(log.id ?? "").trim(),
        productionId: String(log.productionId ?? "").trim(),
        items,
        qrPayloads,
      };
    })
    .filter((log) => log.id !== "" && log.productionId === id);
}

/* ---------------------------------------------------------
 * 印刷用 Product 作成 + print_log 取得
 *   1. Product を rows の順で逐次作成
 *   2. print_log を作成
 *   3. listPrintLogsByProductionId で結果を取得
 *
 * NOTE:
 * modelNumber は createProductHTTP には送らない。
 * Product 作成 API は modelId / productionId / printedAt を正として受ける。
 * modelNumber は printService 層で PDF ラベル生成用に使う。
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();

  if (!id) {
    throw new Error("productionId is required");
  }

  const printedAtISO = new Date().toISOString();

  for (const row of Array.isArray(rows) ? rows : []) {
    const quantity = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity)))
      : 0;

    const modelId = String(row.modelId ?? "").trim();

    if (!modelId || quantity <= 0) {
      continue;
    }

    for (let i = 0; i < quantity; i += 1) {
      await createProductHTTP({
        modelId,
        productionId: id,
        printedAt: printedAtISO,
      });
    }
  }

  await createPrintLogsHTTP(id);

  return listPrintLogsByProductionId(id);
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
  if (!Array.isArray(raw)) return [];

  return raw
    .map((product: any): ProductSummaryForPrint => ({
      id: String(product.id ?? "").trim(),
      modelId: String(product.modelId ?? "").trim(),
      productionId: String(product.productionId ?? "").trim(),
      modelNumber: String(product.modelNumber ?? "").trim(),
    }))
    .filter((product) => product.id !== "" && product.productionId === id);
}