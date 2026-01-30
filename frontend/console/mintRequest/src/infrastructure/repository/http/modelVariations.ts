// frontend/console/mintRequest/src/infrastructure/repository/http/modelVariations.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";
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

  const authHeaders = await getAuthHeadersOrThrow();
  const authValue = String((authHeaders as any)?.Authorization ?? "").trim();
  if (!authValue) {
    throw new Error("Authorization header is missing (not logged in or token unavailable)");
  }

  // For logging only
  const m = authValue.match(/^Bearer\s+(.+)$/i);
  const idToken = (m?.[1] ?? "").trim();

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
          Authorization: idToken ? `Bearer ${safeTokenHint(idToken)}` : safeTokenHint(authValue),
          "Content-Type": "application/json",
        },
        variationId: vid,
      });

      const res = await fetch(url, { method: "GET", headers: authHeaders });

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
      const normalized = normalizeModelVariationForMintDTO(json);

      return normalized;
    } catch (_e: any) {
      // try next candidate
      continue;
    }
  }

  return null;
}
