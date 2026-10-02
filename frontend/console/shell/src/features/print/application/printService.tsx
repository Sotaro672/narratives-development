// frontend/console/shell/src/features/print/application/printService.tsx

import {
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";
import {
  buildQrPdfBlobA4,
  type QrPdfItem,
  type QrPdfProgress,
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
body{background:#fff;color:#111827;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans JP","Yu Gothic",Meiryo,sans-serif}
#qr-preview-status{display:flex;width:100%;height:100%;box-sizing:border-box;align-items:center;justify-content:center}
#qr-preview-status__content{display:flex;align-items:center;gap:12px;font-size:14px}
#qr-preview-status__text{white-space:nowrap}
#qr-preview-error{max-width:720px;padding:24px;font-size:14px;line-height:1.7}
#qr-preview-error strong{display:block;margin-bottom:8px;font-size:16px}
#qr-preview-error p{margin:0;color:#6b7280;word-break:break-word}
.spinner{width:18px;height:18px;box-sizing:border-box;flex:0 0 auto;border:2px solid #d1d5db;border-top-color:#111827;border-radius:50%;animation:spin .8s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<div id="qr-preview-status">
  <div id="qr-preview-status__content">
    <span class="spinner" aria-hidden="true"></span>
    <span id="qr-preview-status__text">印刷データを取得中です…</span>
  </div>
</div>
</body>
</html>`);
  previewWindow.document.close();

  return previewWindow;
}

function updateQrPreviewStatus(
  previewWindow: Window,
  message: string,
): void {
  if (previewWindow.closed) return;

  const statusText = previewWindow.document.getElementById(
    "qr-preview-status__text",
  );

  if (statusText) {
    statusText.textContent = message;
  }
}

function updateQrPreviewProgress(
  previewWindow: Window,
  progress: QrPdfProgress,
): void {
  if (progress.stage === "generating") {
    updateQrPreviewStatus(
      previewWindow,
      `QRコードを生成中です… ${progress.completed} / ${progress.total}`,
    );
    return;
  }

  if (progress.stage === "saving") {
    updateQrPreviewStatus(
      previewWindow,
      "PDFを作成中です…",
    );
    return;
  }

  updateQrPreviewStatus(
    previewWindow,
    "PDFを表示中です…",
  );
}

function showQrPreviewError(
  previewWindow: Window,
  error: unknown,
): void {
  if (previewWindow.closed) return;

  const message =
    error instanceof Error && error.message.trim()
      ? error.message
      : "Unknown error";

  const document = previewWindow.document;

  document.body.innerHTML = "";
  document.body.style.margin = "0";
  document.body.style.width = "100%";
  document.body.style.height = "100%";
  document.body.style.overflow = "auto";

  const errorContainer = document.createElement("div");
  errorContainer.id = "qr-preview-error";

  const title = document.createElement("strong");
  title.textContent = "QR出力に失敗しました。";

  const detail = document.createElement("p");
  detail.textContent = message;

  errorContainer.appendChild(title);
  errorContainer.appendChild(detail);
  document.body.appendChild(errorContainer);
}

/**
 * 生成済み PDF Blob を、クリック直後に開いておいたプレビュータブへ表示する。
 *
 * Blob URL は表示先の previewWindow 側で生成し、
 * 同じ Window 内の iframe から参照する。
 *
 * PDF Viewer が Blob URL を参照している間に revoke しないよう、
 * この処理内では URL.revokeObjectURL を実行しない。
 */
function displayQrPdfInPreviewWindow(
  previewWindow: Window,
  pdfBlob: Blob,
): void {
  if (previewWindow.closed) {
    throw new Error("QR PDF preview window was closed");
  }

  if (pdfBlob.size === 0) {
    throw new Error("Generated QR PDF is empty");
  }

  if (pdfBlob.type !== "application/pdf") {
    throw new Error(
      `Generated QR PDF has invalid MIME type: ${pdfBlob.type || "unknown"}`,
    );
  }

  const previewGlobal =
    previewWindow as Window & typeof globalThis;

  const pdfUrl =
    previewGlobal.URL.createObjectURL(pdfBlob);

  const document = previewWindow.document;

  document.body.innerHTML = "";
  document.body.style.margin = "0";
  document.body.style.width = "100%";
  document.body.style.height = "100%";
  document.body.style.overflow = "hidden";

  const iframe = document.createElement("iframe");

  iframe.src = pdfUrl;
  iframe.title = "QR PDF";
  iframe.style.display = "block";
  iframe.style.width = "100%";
  iframe.style.height = "100%";
  iframe.style.border = "0";

  document.body.appendChild(iframe);
}

async function buildQrPdfFromLogs(
  logs: PrintLogForPrint[],
  onProgress?: (progress: QrPdfProgress) => void,
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
    onProgress,
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
 * 4. qrPayload を QR コード化し、進捗をプレビュー画面へ表示する
 * 5. PDF を生成する
 * 6. previewWindow 側で Blob URL を生成し、同じ Window 内の iframe へ表示する
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
    updateQrPreviewStatus(
      previewWindow,
      "印刷データを取得中です…",
    );

    const logs =
      await ensurePrintLogsForProduction(productionId);

    if (logs.length === 0) {
      throw new Error("QR出力対象の商品がありません");
    }

    updateQrPreviewStatus(
      previewWindow,
      "QRコードを生成する準備をしています…",
    );

    const pdfBlob = await buildQrPdfFromLogs(
      logs,
      (progress) => {
        updateQrPreviewProgress(
          previewWindow,
          progress,
        );
      },
    );

    if (!pdfBlob) {
      throw new Error("QR出力対象のQRペイロードがありません");
    }

    if (previewWindow.closed) {
      throw new Error("QR PDF preview window was closed");
    }

    updateQrPreviewStatus(
      previewWindow,
      "PDFを表示中です…",
    );

    displayQrPdfInPreviewWindow(
      previewWindow,
      pdfBlob,
    );

    return logs;
  } catch (error) {
    console.error(
      "Failed to output QR PDF",
      error,
    );

    showQrPreviewError(
      previewWindow,
      error,
    );

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

  const logs =
    await ensurePrintLogsForProduction(productionId);

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