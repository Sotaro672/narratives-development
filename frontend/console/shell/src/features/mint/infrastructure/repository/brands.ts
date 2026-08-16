// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";

import type { ItemsResult } from "../../../../shared/types/common/common";
import type { BrandSummary } from "../dto/MintRequestRepository";

export async function fetchBrandsForMintHTTP(): Promise<BrandSummary[]> {
  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/mint/brands`;

  const response = await fetch(url, {
    method: "GET",
    headers: authHeaders,
  });

  if (!response.ok) {
    const body = await response.text();

    throw new Error(
      "Failed to fetch brands (mint): " +
        `${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  const responsePayload = (await response.json()) as ItemsResult<BrandSummary>;

  return responsePayload.items;
}