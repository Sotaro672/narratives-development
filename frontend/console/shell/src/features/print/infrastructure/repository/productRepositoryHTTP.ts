// frontend/console/shell/src/features/print/infrastructure/repository/productRepositoryHTTP.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";

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

export type PrintPDFURLResponse = {
  url: string;
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

  const authHeaders = await getAuthHeaders();

  const res = await fetch(`${API_BASE}/products/print-logs`, {
    method: "POST",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
    },
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
  if (!productionId) {
    return [];
  }

  const authHeaders = await getAuthHeaders();

  const res = await fetch(
    `${API_BASE}/products/print-logs?productionId=${encodeURIComponent(productionId)}`,
    {
      method: "GET",
      headers: authHeaders,
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

/**
 * POST /products/print-pdf-url
 *
 * Firebase Auth 済みの Console request から、
 * QR PDF を直接表示するための短時間有効な署名 URL を取得する。
 *
 * backend response:
 * {
 *   "url": "/products/print-pdf?ticket=..."
 * }
 *
 * backend は Host header を信用せず相対 URL を返すため、
 * frontend 側で API_BASE を基準に絶対 URL へ変換する。
 */
export async function createPrintPDFURLHTTP(
  productionId: string,
): Promise<PrintPDFURLResponse> {
  if (!productionId) {
    throw new Error("productionId is required for print PDF URL creation");
  }

  const authHeaders = await getAuthHeaders();

  const res = await fetch(`${API_BASE}/products/print-pdf-url`, {
    method: "POST",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ productionId }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");

    throw new Error(
      `Print PDF URL create failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const response = (await res.json()) as PrintPDFURLResponse;

  if (!response.url || typeof response.url !== "string") {
    throw new Error("Print PDF URL response does not contain a valid url");
  }

  let absoluteURL: string;

  try {
    absoluteURL = new URL(response.url, API_BASE).toString();
  } catch {
    throw new Error("Print PDF URL response contains an invalid url");
  }

  return {
    url: absoluteURL,
  };
}