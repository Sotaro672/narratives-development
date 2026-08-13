// frontend/console/shell/src/features/mint/infrastructure/repository/http/productBlueprintPatch.ts

import {
  API_BASE,
} from "../../../../../shared/http/apiBase";

import {
  getAuthHeadersOrThrow,
} from "../../../../../shared/http/authHeaders";

import type {
  ProductBlueprintPatchDTO,
} from "../../dto/mintRequestLocal.dto";

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const normalizedProductBlueprintId =
    String(
      productBlueprintId ?? "",
    ).trim();

  if (!normalizedProductBlueprintId) {
    throw new Error(
      "productBlueprintId が空です",
    );
  }

  const authHeaders =
    await getAuthHeadersOrThrow();

  const url =
    `${API_BASE}/mint/product_blueprints/` +
    encodeURIComponent(
      normalizedProductBlueprintId,
    );

  const response =
    await fetch(
      url,
      {
        method: "GET",
        headers: authHeaders,
      },
    );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body =
      await response.text();

    throw new Error(
      `Failed to fetch productBlueprint: ` +
        `${response.status} ` +
        `${response.statusText}` +
        (
          body
            ? ` body=${body.slice(0, 400)}`
            : ""
        ),
    );
  }

  return await response.json() as
    ProductBlueprintPatchDTO;
}