// frontend/console/mintRequest/src/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { BrandForMintDTO } from "../../dto/mintRequestLocal.dto";

type BrandRecordRaw = {
  id?: unknown;
  name?: unknown;
};

type BrandPageResultRaw = {
  items?: BrandRecordRaw[] | null;
};

function toText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

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

  const json = (await res.json()) as BrandPageResultRaw | null | undefined;

  const rawItems = Array.isArray(json?.items) ? json.items : [];

  return rawItems
    .map((b): BrandForMintDTO => ({
      id: toText(b.id),
      name: toText(b.name),
    }))
    .filter((b) => b.id && b.name);
}