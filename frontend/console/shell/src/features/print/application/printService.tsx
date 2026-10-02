// frontend/console/shell/src/features/print/application/printService.tsx

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";
import {
  buildQrPdfBlobA4,
  type QrPdfItem,
} from "../utils/qrPdfBuilder";

export type { PrintLogForPrint };

export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];
  return listPrintLogsByProductionIdApi(productionId);
}

/**
 * productionId に紐づく print_log を取得する。
 *
 * 既存 print_log が存在しない場合だけ、
 * POST /products/print-logs により products / print_log / inspections を作成する。
 *
 * この関数自身は QR / CSV の出力を行わない。
 */
async function ensurePrintLogsForProduction(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) {
    throw new Error("productionId is required");
  }

  const existingLogs = await listPrintLogsByProductionIdApi(productionId);

  if (existingLogs.length > 0) {
    return existingLogs;
  }

  return createProductsForPrintApi({
    productionId,
  });
}

/**
 * 初回印刷用。
 *
 * print_log が存在しない場合だけ products / print_log / inspections を作成し、
 * productions.printed を更新する。
 * QR PDF / CSV の出力は行わない。
 */
export async function preparePrintForProduction(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  return ensurePrintLogsForProduction(productionId);
}

function prepareQrPreviewWindow(): Window {
  const previewWindow = window.open("", "_blank");

  if (!previewWindow) {
    throw new Error("QR PDF preview window could not be opened");
  }

  previewWindow.opener = null;

  previewWindow.document.open();
  previewWindow.document.write(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>QR出力</title>
<style>
html,body{width:100%;height:100%;margin:0}
body{display:flex;align-items:center;justify-content:center;background:#fff;color:#111827;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans JP","Yu Gothic",Meiryo,sans-serif}
main{display:flex;align-items:center;gap:12px;font-size:14px}
.spinner{width:18px;height:18px;box-sizing:border-box;border:2px solid #d1d5db;border-top-color:#111827;border-radius:50%;animation:spin .8s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<main><span class="spinner" aria-hidden="true"></span><span>QRコードを作成中です…</span></main>
</body>
</html>`);
  previewWindow.document.close();

  return previewWindow;
}

async function buildQrPdfFromLogs(
  logs: PrintLogForPrint[],
): Promise<Blob | null> {
  const qrItems: QrPdfItem[] = [];

  for (const log of logs) {
    for (const item of log.items) {
      if (!item.qrPayload) continue;

      qrItems.push({
        payload: item.qrPayload,
        label: item.modelNumber,
      });
    }
  }

  if (qrItems.length === 0) {
    return null;
  }

  return buildQrPdfBlobA4(qrItems, {
    cols: 5,
    cellHeight: 100,
  });
}

function downloadProductIdsCsv(
  logs: PrintLogForPrint[],
  productionId: string,
): void {
  const productIds: string[] = [];

  for (const log of logs) {
    for (const item of log.items) {
      if (!item.productId) continue;
      productIds.push(item.productId);
    }
  }

  if (productIds.length === 0) return;

  const csv = [
    "productId",
    ...productIds,
  ].join("\r\n");

  const blob = new Blob(
    [`\uFEFF${csv}`],
    {
      type: "text/csv;charset=utf-8",
    },
  );

  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");

  anchor.href = url;
  anchor.download = `product-ids-${productionId}.csv`;

  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();

  URL.revokeObjectURL(url);
}

/**
 * QR 出力用。
 *
 * 1. ユーザー操作直後に QR PDF 表示用のタブを開く
 * 2. GET /products/print-logs?productionId=... で既存 print_log を確認する
 * 3. 存在しない場合だけ POST /products/print-logs を実行する
 * 4. qrPayload を QR コード化して PDF を生成する
 * 5. 先に開いておいたタブを生成済み PDF の Blob URL へ遷移させる
 *
 * GET が失敗した場合は POST にフォールバックしない。
 */
export async function outputQrForProduction(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  const previewWindow = prepareQrPreviewWindow();

  try {
    const logs = await ensurePrintLogsForProduction(productionId);

    if (logs.length === 0) {
      previewWindow.close();
      return [];
    }

    const pdfBlob = await buildQrPdfFromLogs(logs);

    if (!pdfBlob) {
      previewWindow.close();
      return logs;
    }

    const pdfUrl = URL.createObjectURL(pdfBlob);

    if (previewWindow.closed) {
      throw new Error("QR PDF preview window was closed");
    }

    previewWindow.location.replace(pdfUrl);

    return logs;
  } catch (error) {
    if (!previewWindow.closed) {
      previewWindow.close();
    }

    throw error;
  }
}

/**
 * CSV 出力用。
 *
 * 1. GET /products/print-logs?productionId=... で既存 print_log を確認する
 * 2. 存在しない場合だけ POST /products/print-logs を実行する
 * 3. 各 productId を CSV ファイルとしてダウンロードする
 *
 * GET が失敗した場合は POST にフォールバックしない。
 */
export async function outputProductIdsCsvForProduction(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  const logs = await ensurePrintLogsForProduction(productionId);

  if (logs.length === 0) return [];

  downloadProductIdsCsv(
    logs,
    productionId,
  );

  return logs;
}

/**
 * 既存の呼び出し元との互換用。
 *
 * 従来の「印刷」は QR 出力として扱う。
 */
export async function printOrCreateProductsForPrint(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  return outputQrForProduction(params);
}