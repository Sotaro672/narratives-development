// frontend/console/mintRequest/src/infrastructure/repository/http/inspections.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { InspectionBatchDTO } from "../../api/mintRequestApi";
import type { MintRequestDetailDTO } from "../../dto/mintRequestLocal.dto";

import { fetchProductionIdsForCurrentCompanyHTTP } from "./productions";
import { normalizeMintRequestDetail } from "../../normalizers/mintRequestDetail";

// ===============================
// helpers
// ===============================

function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray(x.inspections) ||
    Array.isArray(x.Inspections) ||
    Array.isArray(x.results) ||
    Array.isArray(x.Results) ||
    Array.isArray(x.items) ||
    Array.isArray(x.Items)
  );
}

// ===============================
// detail: /mint/inspections/{productionId}
// ===============================

export async function fetchMintRequestDetailByProductionIdHTTP(
  productionId: string,
): Promise<MintRequestDetailDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();
  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}`;

  logHttpRequest("fetchMintRequestDetailByProductionIdHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionId: pid,
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  logHttpResponse("fetchMintRequestDetailByProductionIdHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchMintRequestDetailByProductionIdHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch mint request detail: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as any;
  return normalizeMintRequestDetail(json) ?? null;
}

// ===============================
// list: /mint/inspections?productionIds=...
// ===============================

export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const productionIds = await fetchProductionIdsForCurrentCompanyHTTP();
  if (productionIds.length === 0) return [];
  return await fetchInspectionBatchesByProductionIdsHTTP(productionIds);
}

export async function fetchInspectionBatchesByProductionIdsHTTP(
  productionIds: string[],
): Promise<InspectionBatchDTO[]> {
  const ids = (productionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return [];

  const idToken = await getIdTokenOrThrow();
  const url = `${API_BASE}/mint/inspections?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;

  logHttpRequest("fetchInspectionBatchesByProductionIdsHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionIds: ids,
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  logHttpResponse("fetchInspectionBatchesByProductionIdsHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchInspectionBatchesByProductionIdsHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  return json ?? [];
}

// ===============================
// single: inspection by productionId
// ===============================

export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = String(productionId ?? "").trim();
  if (!trimmed) throw new Error("productionId が空です");

  // ✅ detail を優先（batch-shape のときだけ採用）
  try {
    const detail = await fetchMintRequestDetailByProductionIdHTTP(trimmed);
    const inspection = (detail?.inspection ?? null) as any;

    if (looksLikeInspectionBatchDTO(inspection)) {
      return inspection as InspectionBatchDTO;
    }
  } catch (_e: any) {
    // noop: fallbackへ
  }

  // 🔙 fallback: list ルート
  const batches = await fetchInspectionBatchesByProductionIdsHTTP([trimmed]);
  const hit =
    batches.find(
      (b: any) =>
        String((b as any)?.productionId ?? (b as any)?.ProductionID ?? "").trim() ===
        trimmed,
    ) ?? null;

  return hit ?? null;
}
