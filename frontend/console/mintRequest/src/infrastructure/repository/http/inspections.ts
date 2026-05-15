// frontend/console/mintRequest/src/infrastructure/repository/http/inspections.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { InspectionBatchDTO } from "../../../domain/entity/inspections";
import type { MintRequestDetailDTO } from "../../dto/mintRequestLocal.dto";

import { fetchProductionIdsForCurrentCompanyHTTP } from "./productions";

// ===============================
// helpers
// ===============================

function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray((x as any).inspections) ||
    Array.isArray((x as any).results) ||
    Array.isArray((x as any).items)
  );
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

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch mint request detail: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as MintRequestDetailDTO | null | undefined;
  return json ?? null;
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

  const url = `${API_BASE}/mint/inspections?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
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
  const inspection = detail?.inspection ?? null;

  if (!inspection) return null;
  if (!looksLikeInspectionBatchDTO(inspection)) return null;

  return inspection as InspectionBatchDTO;
}

// ===============================
// complete: /products/inspections/complete
// ===============================

export async function completeInspectionHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) throw new Error("productionId が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/products/inspections/complete`;

  const headers = {
    ...authHeaders,
    "Content-Type": "application/json",
  };

  const res = await fetch(url, {
    method: "PATCH",
    headers,
    body: JSON.stringify({
      productionId: pid,
    }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to complete inspection: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json().catch(() => null)) as InspectionBatchDTO | null;

  if (!json) return null;
  if (!looksLikeInspectionBatchDTO(json)) return null;

  return json;
}