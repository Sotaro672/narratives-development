// frontend/console/product/src/application/printService.tsx

// このファイルは、アプリケーション層からインフラ層 API を呼び出しつつ、
// print_log → QR → PDF 生成までをまとめて扱うラッパです。

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  listProductsByProductionId as listProductsByProductionIdApi,
  createInspectionBatchForProduction as createInspectionBatchForProductionApi,
  type PrintRow,
  type ProductSummaryForPrint,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";

import {
  buildQrPdfBlobA4,
  openQrPdfInNewTab,
  type QrPdfItem,
} from "../utils/qrPdfBuilder";

// ★ 印刷完了シグナルを送るために ProductionDetailService を import
import { notifyPrintLogCompleted } from "../../../production/src/application/productionDetailService";

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
// ヘルパー: Product 一覧から productId -> ラベル のマップを作成
//   ※ ラベルは modelNumber のみを使う（無い場合はラベル無し）
// ==============================

function buildProductLabelMap(
  products: ProductSummaryForPrint[] | undefined,
): Map<string, string> {
  const map = new Map<string, string>();
  if (!products || products.length === 0) return map;

  for (const p of products) {
    // ProductSummaryForPrint は { id, modelId, productionId, modelNumber? }
    const productId = p.id;
    if (!productId) continue;

    const modelNumber = (p.modelNumber ?? "").trim();
    if (!modelNumber) {
      // modelNumber が無いものはラベルを付けない
      continue;
    }

    // productId -> modelNumber
    map.set(productId, modelNumber);
  }

  return map;
}

// ==============================
// ヘルパー: PrintLog 一覧から QR PDF を生成して開く
//   ★ 第3引数に products を追加し、modelNumber をラベルに利用
// ==============================

async function buildAndOpenQrPdfFromLogs(
  logs: PrintLogForPrint[],
  productionId: string,
  products?: ProductSummaryForPrint[],
): Promise<number> {
  const qrItems: QrPdfItem[] = [];

  // productId -> ラベル(modelNumber) のマップ
  const productLabelMap = buildProductLabelMap(products);

  logs.forEach((log) => {
    const { productIds, qrPayloads } = log;

    // qrPayloads が null / undefined / 配列以外のケースに備えて安全に扱う
    const payloadList = Array.isArray(qrPayloads) ? qrPayloads : [];

    productIds.forEach((pid, index) => {
      const payloadJson = payloadList[index];
      if (!pid || !payloadJson) return;

      // ★ ラベルは modelNumber のみ。無ければラベル無し。
      const label = productLabelMap.get(pid);

      // payloadJson は JSON 文字列としてそのまま QR に埋め込む想定。
      qrItems.push({
        payload: payloadJson,
        label,
      });
    });
  });

  // QR 対象が 0 件なら何もしない
  if (qrItems.length === 0) {
    return 0;
  }

  // A4 縦・1 行 5 つの QR PDF を生成
  const pdfBlob = await buildQrPdfBlobA4(qrItems, {
    cols: 5,
    cellHeight: 100,
  });

  // 新しいタブで表示
  openQrPdfInNewTab(pdfBlob);

  return qrItems.length;
}

// ==============================
// 印刷用: Product 作成 + print_log 作成 + QR PDF 表示
//
// 仕様拡張：
// - まず既存の print_log を検索
//   - あれば、それを使って QR PDF を再生成するだけ（print_log は増やさない）
//   - なければ、従来通り Product 作成 → print_log 作成 → print_log 取得 → PDF 生成
//
// 1. /products?productionId=... で既存 print_log を確認
// 2. あれば buildAndOpenQrPdfFromLogs で PDF 表示のみ
// 3. なければ /products に対して Product を必要数作成
// 4. /products/print-logs に対して print_log 作成リクエストを送信
//    （★ 同じタイミングで /products/inspections も作成要求）
// 5. /products/print-logs?productionId=... で print_log を取得
// 6. 取得した print_log から QR JSON を取り出し、A4 1 行 5 つで PDF 生成
// 7. 新しいタブで PDF を開く
//
// ★ ProductBlueprint の printed 遷移（notYet → printed）は
//    バックエンド側の PrintUsecase / ProductBlueprintUsecase に任せる。
//    フロントから mark-printed API は呼ばない。
// ==============================

export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  // ------------------------------------------------------
  // 0. 既存の print_log があるかどうかを確認
  // ------------------------------------------------------
  const existingLogs = await listPrintLogsByProductionIdApi(id);

  if (existingLogs.length > 0) {
    // ★ 既存 Product 一覧を取得し、modelNumber をラベルに使う
    const products = await listProductsByProductionId(id);

    const totalQrCount = await buildAndOpenQrPdfFromLogs(
      existingLogs,
      id,
      products,
    );

    // ★ 既存ログを使って PDF を開いたことをシグナルとして通知
    await notifyPrintLogCompleted({
      productionId: id,
      logCount: existingLogs.length,
      totalQrCount,
      reusedExistingLogs: true,
    });

    // printed 状態は過去の初回印刷時にバックエンド側で更新済みの想定。
    // 再印刷時は printed を変更しない。
    return existingLogs;
  }

  // ------------------------------------------------------
  // 1〜3. バックエンドを叩いて Product 作成 → print_log 作成 → print_log 取得
  // ------------------------------------------------------
  const logs = await createProductsForPrintApi({
    productionId: id,
    rows,
  });

  // print_log が 0 件なら PDF 生成はスキップ
  if (logs.length === 0) {
    await notifyPrintLogCompleted({
      productionId: id,
      logCount: 0,
      totalQrCount: 0,
      reusedExistingLogs: false,
    });
    // printed の遷移も発生しない（バックエンド側も何もしない想定）
    return logs;
  }

  // ★ 新規 print_log 作成と同じタイミングで inspection テーブル作成を要求
  await createInspectionBatchForProductionApi(id);

  // ★ 新規作成した場合も Product 一覧を取得して modelNumber をラベルに使う
  const products = await listProductsByProductionId(id);

  // ------------------------------------------------------
  // 4〜6. print_log から QR PDF を生成
  // ------------------------------------------------------
  const totalQrCount = await buildAndOpenQrPdfFromLogs(logs, id, products);

  // ★ 最後に「print_log 作成 + QR PDF 生成完了」のシグナルを送る
  await notifyPrintLogCompleted({
    productionId: id,
    logCount: logs.length,
    totalQrCount,
    reusedExistingLogs: false,
  });

  // ★ ProductBlueprint の printed: notYet → printed への遷移は
  //    バックエンド側（PrintUsecase / ProductBlueprintUsecase）に任せる。
  //    ここでは mark-printed API を呼ばない。

  return logs;
}
