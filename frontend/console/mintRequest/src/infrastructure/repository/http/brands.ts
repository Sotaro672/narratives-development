// frontend/console/mintRequest/src/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { BrandForMintDTO } from "../../dto/mintRequestLocal.dto";
import type {
  BrandPageResultDTO,
  BrandRecordRaw
} from "../../dto/mintRequestRaw.dto";

export async function fetchBrandsForMintHTTP(): Promise<BrandForMintDTO[]> {
  const idToken = await getIdTokenOrThrow();
  const url = `${API_BASE}/mint/brands`;

  logHttpRequest("fetchBrandsForMintHTTP", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

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

  const rawItems: BrandRecordRaw[] = json?.items ?? json?.Items ?? [];

  return rawItems
    .map((b) => ({
      id: String(b.id ?? b.ID ?? "").trim(),
      name: String(b.name ?? b.Name ?? "").trim(),
    }))
    .filter((b) => b.id && b.name);
}
