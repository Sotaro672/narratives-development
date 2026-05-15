// frontend/console/mintRequest/src/infrastructure/repository/http/modelVariations.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { ModelVariationForMintDTO } from "../../dto/mintRequestLocal.dto";

export async function fetchModelVariationByIdForMintHTTP(
  variationId: string,
): Promise<ModelVariationForMintDTO | null> {
  const vid = String(variationId ?? "").trim();
  if (!vid) throw new Error("variationId が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const candidates = [
    `${API_BASE}/models/variations/${encodeURIComponent(vid)}`,
    `${API_BASE}/model/variations/${encodeURIComponent(vid)}`,
  ];

  for (const url of candidates) {
    try {
      const res = await fetch(url, { method: "GET", headers: authHeaders });

      if (res.status === 404 || res.status === 405) continue;
      if (res.status >= 500) continue;

      if (!res.ok) {
        const body = await res.text().catch(() => "");
        throw new Error(
          `Failed to fetch model variation: ${res.status} ${res.statusText}${
            body ? ` body=${body.slice(0, 400)}` : ""
          }`,
        );
      }

      const json = (await res.json()) as ModelVariationForMintDTO | null | undefined;
      return json ?? null;
    } catch (_e: any) {
      // try next candidate
      continue;
    }
  }

  return null;
}