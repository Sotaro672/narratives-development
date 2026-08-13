// frontend/console/shell/src/features/print/infrastructure/repository/productRepositoryHTTP.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../shared/http/authHeaders";

export type PrintedItemForPrint = {
  productId: string;
  displayOrder: number;
  qrPayload: string;
  modelNumber: string;
};

export type PrintLogForPrint = {
  id: string;
  productionId: string;
  items: PrintedItemForPrint[];
};

/**
 * POST /products/print-logs
 *
 * backend 側で以下をまとめて実行する:
 * - production.models から products を作成
 * - print_log 作成
 * - inspections 作成
 * - productions.printed = true
 * - QR PDF に必要な qrPayload / modelNumber を解決
 *
 * backend が返す印刷用 BFF response をそのまま正として扱う。
 */
export async function createPrintLogsHTTP(
  productionId: string,
): Promise<PrintLogForPrint> {
  if (!productionId) {
    throw new Error("productionId is required for print_log creation");
  }

  const res = await fetch(`${API_BASE}/products/print-logs`, {
    method: "POST",
    headers: await getAuthJsonHeadersOrThrow(),
    body: JSON.stringify({ productionId }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `PrintLog create failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  return res.json();
}

/**
 * GET /products/print-logs?productionId={productionId}
 *
 * backend の PrintQueryService が返す印刷用 BFF response を
 * そのまま正として扱う。
 */
export async function fetchPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  if (!productionId) return [];

  const res = await fetch(
    `${API_BASE}/products/print-logs?productionId=${encodeURIComponent(productionId)}`,
    {
      method: "GET",
      headers: await getAuthJsonHeadersOrThrow(),
    },
  );

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `List print_logs failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  return res.json();
}