// frontend/console/product/src/application/printService.tsx

// このファイルは、アプリケーション層からインフラ層 API を呼び出しつつ、
// print_log → QR → PDF 生成までをまとめて扱うラッパです。

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

// ==============================
// 型の再エクスポート（既存の import を壊さないため）
// ==============================

export type { PrintRow, ProductSummaryForPrint, PrintLogForPrint };

// ==============================
// print_log 一覧取得（アプリ層ラッパ）
//   GET /products/print-logs?productionId=...
// ==============================

export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const logs = await listPrintLogsByProductionIdApi(id);
  return logs;
}

// ==============================
// products 一覧取得（アプリ層ラッパ）
//   GET /products?productionId=...
// ==============================

export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const products = await listProductsByProductionIdApi(id);
  return products;
}

// ==============================
// 印刷用: Product 作成 + print_log 作成 + QR PDF 表示
//
// 1. /products に対して Product を必要数作成
// 2. /products/print-logs に対して print_log 作成リクエストを送信
// 3. /products/print-logs?productionId=... で print_log を取得
// 4. 取得した print_log から QR JSON を取り出し、A4 1 行 4 つで PDF 生成
// 5. 新しいタブで PDF を開く
//
// → usePrintCard.tsx からは、この戻り値（printLogs）を受け取って
//   ダイアログ表示やデバッグに利用できます。
// ==============================

export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  // 1〜3. バックエンドを叩いて Product 作成 → print_log 作成 → print_log 取得
  const logs = await createProductsForPrintApi({
    productionId: id,
    rows,
  });

  // print_log が 0 件なら PDF 生成はスキップ
  if (logs.length === 0) {
    return logs;
  }

  // 4. print_log から QR PDF 用のアイテムを組み立て
  const qrItems: QrPdfItem[] = [];

  logs.forEach((log) => {
    const { productIds, qrPayloads } = log;

    productIds.forEach((pid, index) => {
      const payloadJson = qrPayloads[index];
      if (!pid || !payloadJson) return;

      // payloadJson は JSON 文字列としてそのまま QR に埋め込む想定。
      // productId をラベルにして PDF に表示。
      qrItems.push({
        payload: payloadJson,
        label: pid,
      });
    });
  });

  // QR 対象が 1 件もなければ終了
  if (qrItems.length === 0) {
    return logs;
  }

  // A4 縦・1 行 4 つの QR PDF を生成
  const pdfBlob = await buildQrPdfBlobA4(qrItems, {
    cols: 5,
    cellHeight: 100,
  });

  // 5. 新しいタブで表示
  openQrPdfInNewTab(pdfBlob);

  return logs;
}
