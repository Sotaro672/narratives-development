// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/brands.ts

import { API_BASE } from "../../../../../shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shared/http/authHeaders";

import type {
  ItemsResult,
} from "../../../../../shared/types/common/common";

import type {
  BrandSummary,
} from "../../../application/port/MintRequestRepository";

type BrandRecordRaw = {
  id?: unknown;
  name?: unknown;
};

function toText(
  value: unknown,
): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

export async function fetchBrandsForMintHTTP(): Promise<
  BrandSummary[]
> {
  const authHeaders =
    await getAuthHeadersOrThrow();

  const url =
    `${API_BASE}/mint/brands`;

  const response =
    await fetch(
      url,
      {
        method: "GET",
        headers: authHeaders,
      },
    );

  if (!response.ok) {
    const body =
      await response
        .text()
        .catch(
          () => "",
        );

    throw new Error(
      "Failed to fetch brands (mint): " +
        `${response.status} ${response.statusText}` +
        (
          body
            ? ` body=${body.slice(0, 400)}`
            : ""
        ),
    );
  }

  const responsePayload =
    await response.json() as
      | ItemsResult<BrandRecordRaw>
      | null
      | undefined;

  const rawItems =
    Array.isArray(
      responsePayload?.items,
    )
      ? responsePayload.items
      : [];

  return rawItems
    .map(
      (
        brand,
      ): BrandSummary => ({
        id:
          toText(
            brand.id,
          ),

        name:
          toText(
            brand.name,
          ),
      }),
    )
    .filter(
      (brand) =>
        Boolean(
          brand.id &&
            brand.name,
        ),
    );
}