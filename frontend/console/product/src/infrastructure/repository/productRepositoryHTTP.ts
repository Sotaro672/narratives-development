// frontend/console/product/src/infrastructure/repository/productRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

/* ---------------------------------------------------------
 * Product 作成API（印刷用）: 1件分
 *   POST /products
 * --------------------------------------------------------- */
export async function createProductHTTP(payload: {
  modelId: string;
  productionId: string;
  printedAt: string; // ISO 文字列
  printedBy?: string | null;
}): Promise<void> {
  const res = await fetch(`${API_BASE}/products`, {
    method: "POST",
    headers: await getAuthJsonHeadersOrThrow(),
    body: JSON.stringify({
      modelId: payload.modelId,
      productionId: payload.productionId,
      printedAt: payload.printedAt,
      printedBy: payload.printedBy ?? null,
      // inspectionResult / connectedToken などは
      // バックエンド側でデフォルト値(notYet/null)を設定する想定
    }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Product create failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }
}

/* ---------------------------------------------------------
 * print_log 作成 API:
 *   POST /products/print-logs
 *   body: { productionId }
 *   → バックエンド側で
 *     - 対象 productionId の products をもとに print_log を作成
 *     - BuildProductQRValue を実行して QR ペイロードを生成
 * --------------------------------------------------------- */
export async function createPrintLogsHTTP(productionId: string): Promise<void> {
  const id = productionId.trim();
  if (!id) {
    throw new Error("productionId is required for print_log creation");
  }

  const res = await fetch(`${API_BASE}/products/print-logs`, {
    method: "POST",
    headers: await getAuthJsonHeadersOrThrow(),
    body: JSON.stringify({ productionId: id }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `PrintLog create failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }
}

/* ---------------------------------------------------------
 * print_log 取得 API（生 JSON を返す）:
 *   GET /products/print-logs?productionId={id}
 *   → any[] を返し、マッピングは application 層で実施
 * --------------------------------------------------------- */
export async function fetchPrintLogsByProductionId(
  productionId: string,
): Promise<any[]> {
  const id = productionId.trim();
  if (!id) return [];

  const safeId = encodeURIComponent(id);

  const res = await fetch(
    `${API_BASE}/products/print-logs?productionId=${safeId}`,
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

  const raw = (await res.json()) as any[];
  return raw;
}

/* ---------------------------------------------------------
 * products 取得 API（生 JSON を返す）:
 *   GET /products?productionId={id}
 *   → any[] を返し、マッピングは application 層で実施
 * --------------------------------------------------------- */
export async function fetchProductsByProductionId(
  productionId: string,
): Promise<any[]> {
  const id = productionId.trim();
  if (!id) return [];

  const safeId = encodeURIComponent(id);

  const res = await fetch(`${API_BASE}/products?productionId=${safeId}`, {
    method: "GET",
    headers: await getAuthJsonHeadersOrThrow(),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `List products failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = (await res.json()) as any[];
  return raw;
}