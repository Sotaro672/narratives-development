// frontend/console/shell/src/features/mint/infrastructure/repository/http/mintProductBlueprint.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";
import type { MintProductBlueprintDTO } from "../dto/mintRequestLocal.dto";

/**
 * GET /mint/product_blueprints/{id}
 *
 * Backend BFF の MintProductBlueprintDTO を正としてそのまま返す。
 * Frontend側ではpatch形式への変換やfallbackは行わない。
 */
export async function fetchMintProductBlueprintHTTP(
  productBlueprintId: string,
): Promise<MintProductBlueprintDTO | null> {
  const normalizedProductBlueprintId = String(productBlueprintId ?? "").trim();

  if (!normalizedProductBlueprintId) {
    throw new Error("productBlueprintId が空です");
  }

  const authHeaders = await getAuthHeaders();
  const url =
    `${API_BASE}/mint/product_blueprints/` +
    encodeURIComponent(normalizedProductBlueprintId);

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
      `Failed to fetch mint product blueprint: ` +
        `${response.status} ${response.statusText}` +
        (body ? ` body=${body.slice(0, 400)}` : ""),
    );
  }

  return (await response.json()) as MintProductBlueprintDTO;
}