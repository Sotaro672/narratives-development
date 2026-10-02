// frontend/console/shell/src/features/print/application/printService.tsx

import {
  createPrintPDFURL as createPrintPDFURLApi,
  createProductsForPrint as createProductsForPrintApi,
  listPrintLogsByProductionId as listPrintLogsByProductionIdApi,
  type PrintLogForPrint,
} from "../infrastructure/api/printApi";

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

/**
 * Popup blocker を避けるため、ユーザー操作と同じ同期処理内で
 * PDF 表示先の空タブを先に確保する。
 *
 * noopener を window.open の features に指定すると Window reference を
 * 取得できない browser があるため、open 後に opener を切り離す。
 */
function openPrintPDFPreviewWindow(): Window {
  const previewWindow = window.open("about:blank", "_blank");

  if (!previewWindow) {
    throw new Error("PDF preview popup was blocked");
  }

  try {
    previewWindow.document.title = "PDFを準備中です…";
    previewWindow.document.body.style.margin = "0";
    previewWindow.document.body.style.padding = "24px";
    previewWindow.document.body.style.fontFamily = "sans-serif";
    previewWindow.document.body.textContent = "PDFを準備中です…";
  } catch {
    // about:blank の初期表示設定に失敗しても PDF 遷移自体は継続する。
  }

  try {
    previewWindow.opener = null;
  } catch {
    // opener の切り離しに失敗しても PDF 遷移自体は継続する。
  }

  return previewWindow;
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
 * 1. ユーザー操作直後に PDF 表示用の空タブを同期で開く
 * 2. GET /products/print-logs?productionId=... で既存 print_log を確認する
 * 3. 存在しない場合だけ POST /products/print-logs を実行する
 * 4. POST /products/print-pdf-url で短時間有効な署名付き HTTPS URL を取得する
 * 5. 先に確保したタブを backend の application/pdf response へ遷移させる
 *
 * browser 側では QR / PDF を生成しない。
 * GET が失敗した場合は POST /products/print-logs にフォールバックしない。
 */
export async function outputQrForProduction(params: {
  productionId: string;
}): Promise<PrintLogForPrint[]> {
  const { productionId } = params;

  if (!productionId) {
    throw new Error("productionId is required");
  }

  let previewWindow: Window | null = null;

  try {
    previewWindow = openPrintPDFPreviewWindow();

    const logs = await ensurePrintLogsForProduction(productionId);

    if (logs.length === 0) {
      if (!previewWindow.closed) {
        previewWindow.close();
      }
      return [];
    }

    const pdfURL = await createPrintPDFURLApi({
      productionId,
    });

    if (previewWindow.closed) {
      throw new Error("PDF preview window was closed");
    }

    previewWindow.location.replace(pdfURL);

    return logs;
  } catch (error) {
    if (previewWindow && !previewWindow.closed) {
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