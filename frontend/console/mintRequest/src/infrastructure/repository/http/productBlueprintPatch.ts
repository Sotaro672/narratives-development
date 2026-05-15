// frontend/console/mintRequest/src/infrastructure/repository/http/productBlueprintPatch.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { ProductBlueprintPatchDTO } from "../../dto/mintRequestLocal.dto";

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const pbid = String(productBlueprintId ?? "").trim();
  if (!pbid) throw new Error("productBlueprintId が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/mint/product_blueprints/${encodeURIComponent(pbid)}/patch`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as ProductBlueprintPatchDTO | null | undefined;
  return json ?? null;
}