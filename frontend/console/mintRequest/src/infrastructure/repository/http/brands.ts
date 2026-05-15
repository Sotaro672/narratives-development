// frontend/console/mintRequest/src/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { BrandForMintDTO } from "../../dto/mintRequestLocal.dto";
import type { BrandPageResultDTO, BrandRecordRaw } from "../../dto/mintRequestRaw.dto";

export async function fetchBrandsForMintHTTP(): Promise<BrandForMintDTO[]> {
  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/mint/brands`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
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