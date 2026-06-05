// frontend/console/product/src/infrastructure/api/printApi.ts

import {
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
 * 初回印刷処理
 *
 * NOTE:
 * 関数名は既存呼び出し互換のため createProductsForPrint のまま。
 * ただし、frontend から個別に POST /products は呼ばない。
 *
 * 実際には POST /products/print-logs を呼び、
 * backend 側で以下をまとめて処理する:
 *   - production.models から products 作成
 *   - print_log 作成
 *   - inspections 作成
 *   - productions.printed = true
 *
 * rows は backend には送らない。
 * rows は printService 層で QR PDF のラベル生成に使う。
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;
  const id = productionId.trim();

  if (!id) {
    throw new Error("productionId is required");
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