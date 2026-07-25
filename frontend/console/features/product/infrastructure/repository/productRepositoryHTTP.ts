// frontend/console/product/src/infrastructure/repository/productRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthJsonHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

/* ---------------------------------------------------------
 * print_log 作成 API:
 *   POST /products/print-logs
 *   body: { productionId }
 *
 * バックエンド側で以下をまとめて実行する:
 *   - production.models から products を作成
 *   - print_log を作成
 *   - inspections を作成
 *   - productions.printed を true に更新
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
 *
 * NOTE:
 * 初回印刷前は print_log が存在しないため、404 は空配列として扱う。
 * 認証エラーや 500 は通常どおり throw する。
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

  if (res.status === 404) {
    return [];
  }

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `List print_logs failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = await res.json();

  if (!Array.isArray(raw)) {
    return [];
  }

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

  const raw = await res.json();

  if (!Array.isArray(raw)) {
    return [];
  }

  return raw;
}