// frontend/console/mintRequest/src/infrastructure/repository/http/productBlueprintPatch.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { ProductBlueprintPatchDTO } from "../../dto/mintRequestLocal.dto";
import { normalizeProductBlueprintPatch } from "../../normalizers/productBlueprintPatch";

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const pbid = String(productBlueprintId ?? "").trim();
  if (!pbid) throw new Error("productBlueprintId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/product_blueprints/${encodeURIComponent(pbid)}/patch`;

  logHttpRequest("fetchProductBlueprintPatchHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productBlueprintId: pbid,
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  logHttpResponse("fetchProductBlueprintPatchHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchProductBlueprintPatchHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch productBlueprintPatch: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as any;
  return normalizeProductBlueprintPatch(json) ?? null;
}
