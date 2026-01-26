// frontend/console/mintRequest/src/infrastructure/repository/http/modelVariations.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpRequest,
  logHttpResponse,
  logHttpError,
  safeTokenHint,
} from "../../http/httpLogger";

import type { ModelVariationForMintDTO } from "../../dto/mintRequestLocal.dto";
import { normalizeModelVariationForMintDTO } from "../../normalizers/modelVariation";

export async function fetchModelVariationByIdForMintHTTP(
  variationId: string,
): Promise<ModelVariationForMintDTO | null> {
  const vid = String(variationId ?? "").trim();
  if (!vid) throw new Error("variationId が空です");

  const idToken = await getIdTokenOrThrow();

  const candidates = [
    `${API_BASE}/models/variations/${encodeURIComponent(vid)}`,
    `${API_BASE}/model/variations/${encodeURIComponent(vid)}`,
  ];

  for (const url of candidates) {
    try {
      logHttpRequest("fetchModelVariationByIdForMintHTTP", {
        method: "GET",
        url,
        headers: {
          Authorization: `Bearer ${safeTokenHint(idToken)}`,
          "Content-Type": "application/json",
        },
        variationId: vid,
      });

      const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

      logHttpResponse("fetchModelVariationByIdForMintHTTP", {
        method: "GET",
        url,
        status: res.status,
        statusText: res.statusText,
      });

      if (res.status === 404 || res.status === 405) continue;
      if (res.status >= 500) continue;

      if (!res.ok) {
        const body = await res.text().catch(() => "");
        logHttpError("fetchModelVariationByIdForMintHTTP", {
          method: "GET",
          url,
          status: res.status,
          statusText: res.statusText,
          bodyPreview: body ? body.slice(0, 800) : "",
        });
        throw new Error(
          `Failed to fetch model variation: ${res.status} ${res.statusText}${
            body ? ` body=${body.slice(0, 400)}` : ""
          }`,
        );
      }

      const json = (await res.json()) as any;
      return normalizeModelVariationForMintDTO(json);
    } catch (_e: any) {
      // try next candidate
      continue;
    }
  }

  return null;
}
