// frontend/console/mintRequest/src/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { BrandForMintDTO } from "../../dto/mintRequestLocal.dto";
import type { BrandPageResultDTO, BrandRecordRaw } from "../../dto/mintRequestRaw.dto";

export async function fetchBrandsForMintHTTP(): Promise<BrandForMintDTO[]> {
  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }

  const url = `${API_BASE}/mint/brands`;

  // For logging only: extract raw token if possible
  const m = authValue.match(/^Bearer\s+(.+)$/i);
  const idToken = (m?.[1] ?? "").trim();

  logHttpRequest("fetchBrandsForMintHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  logHttpResponse("fetchBrandsForMintHTTP", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchBrandsForMintHTTP", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch brands (mint): ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as BrandPageResultDTO | null | undefined;

  const rawItems: BrandRecordRaw[] = json?.items ?? (json as any)?.Items ?? [];

  return rawItems
    .map((b) => ({
      id: String((b as any).id ?? (b as any).ID ?? "").trim(),
      name: String((b as any).name ?? (b as any).Name ?? "").trim(),
    }))
    .filter((b) => b.id && b.name);
}
