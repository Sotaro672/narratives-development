// frontend/console/mintRequest/src/infrastructure/repository/http/inspections.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { InspectionBatchDTO } from "../../dto/inspectionBatch.dto";
import type { MintRequestDetailDTO } from "../../dto/mintRequestLocal.dto";

import { fetchProductionIdsForCurrentCompanyHTTP } from "./productions";
import { normalizeMintRequestDetail } from "../../normalizers/mintRequestDetail";

// ===============================
// helpers
// ===============================

function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray((x as any).inspections) ||
    Array.isArray((x as any).Inspections) ||
    Array.isArray((x as any).results) ||
    Array.isArray((x as any).Results) ||
    Array.isArray((x as any).items) ||
    Array.isArray((x as any).Items)
  );
}

function getAuthValueOrThrow(authHeaders: Record<string, string>): string {
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }
  return authValue;
}

function extractIdTokenForLog(authValue: string): string {
  const m = String(authValue ?? "").match(/^Bearer\s+(.+)$/i);
  return String(m?.[1] ?? "").trim();
}

// ===============================
// private: detail fetch (/mint/inspections/{productionId})
// - public API からは export しない
// ===============================

async function fetchMintRequestDetailByProductionIdHTTP(
  productionId: string,
): Promise<MintRequestDetailDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}`;

  logHttpRequest("fetchMintRequestDetailByProductionIdHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
    productionId: pid,
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

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

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = getAuthValueOrThrow(authHeaders);
  const idToken = extractIdTokenForLog(authValue);

  const url = `${API_BASE}/mint/inspections?productionIds=${encodeURIComponent(ids.join(","))}`;

  logHttpRequest("fetchInspectionBatchesByProductionIdsHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
    productionIds: ids,
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

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
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const detail = await fetchMintRequestDetailByProductionIdHTTP(pid);
  const inspection = (detail?.inspection ?? null) as any;

  if (!inspection) return null;
  if (!looksLikeInspectionBatchDTO(inspection)) return null;

  return inspection as InspectionBatchDTO;
}
