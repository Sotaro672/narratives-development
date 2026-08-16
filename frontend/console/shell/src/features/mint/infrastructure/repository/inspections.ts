// frontend/console/shell/src/features/mint/infrastructure/repository/http/inspections.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";

import type { InspectionBatch } from "../../../../shared/types/inspections";
import type { MintRequestDetailDTO } from "../dto/mintRequestLocal.dto";

// ===============================
// detail: /mint/inspections/{productionId}
// ===============================

/**
 * GET /mint/inspections/{productionId}
 *
 * Backend BFF の MintRequestDetailDTO をそのまま返す。
 * Frontend側ではinspectionを独自DTOへ再構築しない。
 *
 * productBlueprintId / productName / modelMeta / inspectionは
 * Backend responseの構造を正とする。
 */
export async function fetchMintRequestDetailHTTP(
  productionId: string,
): Promise<MintRequestDetailDTO | null> {
  if (!productionId) {
    throw new Error("productionId が空です");
  }

  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(productionId)}`;

  const response = await fetch(url, {
    method: "GET",
    headers: authHeaders,
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body = await response.text();

    throw new Error(
      `Failed to fetch mint request detail: ${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  return (await response.json()) as MintRequestDetailDTO;
}

// ===============================
// complete: /products/inspections/complete
// ===============================

/**
 * productionIdに紐づく検品を完了する。
 *
 * このAPIはMint detail BFFとは別のCommand APIのため、
 * Backendが返すInspectionBatchをそのまま使用する。
 */
export async function completeInspectionHTTP(
  productionId: string,
): Promise<InspectionBatch | null> {
  if (!productionId) {
    throw new Error("productionId が空です");
  }

  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/products/inspections/complete`;

  const response = await fetch(url, {
    method: "PATCH",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ productionId }),
  });

  if (!response.ok) {
    const body = await response.text();

    throw new Error(
      `Failed to complete inspection: ${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  return (await response.json()) as InspectionBatch;
}