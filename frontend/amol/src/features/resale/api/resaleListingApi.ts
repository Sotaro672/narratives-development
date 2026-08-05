// frontend/amol/src/features/resale/api/resaleListingApi.ts

import {
  fetchResaleWithAuth,
  type ApiDataResponse,
} from "./resaleHttpClient";

import type {
  CreateResaleListingRecordParams,
  ListMyResaleListingsParams,
  ListMyResaleListingsResponse,
  ResaleImageIdentifier,
  ResaleListing,
  UpdateResaleListingParams,
} from "../../shared/types/resaleTypes";

function nonEmptyOrUndefined(
  value: string | undefined,
): string | undefined {
  const normalized =
    value?.trim();

  return normalized
    ? normalized
    : undefined;
}

export async function createResaleListingRecord(
  params: CreateResaleListingRecordParams,
): Promise<ResaleListing | null> {
  const result =
    await fetchResaleWithAuth<
      ApiDataResponse<ResaleListing>
    >(
      "/mall/me/resales",
      {
        method: "POST",
        body: JSON.stringify({
          mintAddress:
            params.mintAddress,
          tokenBlueprintId:
            params.tokenBlueprintId,
          productId:
            params.productId,
          brandId:
            nonEmptyOrUndefined(
              params.brandId,
            ),
          productBlueprintId:
            nonEmptyOrUndefined(
              params.productBlueprintId,
            ),
          price:
            params.price,
          condition:
            params.condition,
          description:
            params.description,
        }),
      },
    );

  return result.data ?? null;
}

export async function updateResaleListing(
  params: UpdateResaleListingParams,
): Promise<ResaleListing | null> {
  const resaleId =
    params.resaleId.trim();

  if (!resaleId) {
    throw new Error(
      "resaleId is required",
    );
  }

  const body: {
    price?: number;
    condition?: string;
    description?: string;
    status?: string;
  } = {};

  if (
    typeof params.price === "number" &&
    Number.isFinite(params.price)
  ) {
    body.price =
      params.price;
  }

  const condition =
    nonEmptyOrUndefined(
      params.condition,
    );

  if (condition) {
    body.condition =
      condition;
  }

  if (
    typeof params.description ===
    "string"
  ) {
    body.description =
      params.description.trim();
  }

  const status =
    nonEmptyOrUndefined(
      params.status,
    );

  if (status) {
    body.status =
      status;
  }

  const result =
    await fetchResaleWithAuth<
      ApiDataResponse<ResaleListing>
    >(
      `/mall/me/resales/${encodeURIComponent(
        resaleId,
      )}`,
      {
        method: "PUT",
        body: JSON.stringify(
          body,
        ),
      },
    );

  return result.data ?? null;
}

export async function getMyResaleListing(
  resaleId: string,
): Promise<ResaleListing | null> {
  const normalizedResaleId =
    resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error(
      "resaleId is required",
    );
  }

  const result =
    await fetchResaleWithAuth<
      ApiDataResponse<ResaleListing>
    >(
      `/mall/me/resales/${encodeURIComponent(
        normalizedResaleId,
      )}`,
      {
        method: "GET",
      },
    );

  return result.data ?? null;
}

export async function listMyResaleListings(
  params: ListMyResaleListingsParams = {},
): Promise<ListMyResaleListingsResponse> {
  return fetchResaleWithAuth<
    ListMyResaleListingsResponse
  >(
    "/mall/me/resales",
    {
      method: "GET",
      query: {
        page:
          params.page ?? 1,
        perPage:
          params.perPage ?? 50,
      },
    },
  );
}

export async function updatePrimaryResaleImage({
  resaleId,
  imageId,
}: ResaleImageIdentifier): Promise<ResaleListing | null> {
  const normalizedResaleId =
    resaleId.trim();

  const normalizedImageId =
    imageId.trim();

  if (!normalizedResaleId) {
    throw new Error(
      "resaleId is required",
    );
  }

  if (!normalizedImageId) {
    throw new Error(
      "imageId is required",
    );
  }

  const result =
    await fetchResaleWithAuth<
      ApiDataResponse<ResaleListing>
    >(
      `/mall/me/resales/${encodeURIComponent(
        normalizedResaleId,
      )}/primary-image`,
      {
        method: "PUT",
        body: JSON.stringify({
          imageId:
            normalizedImageId,
        }),
      },
    );

  return result.data ?? null;
}

export async function deleteResaleListing(
  resaleId: string,
): Promise<void> {
  const normalizedResaleId =
    resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error(
      "resaleId is required",
    );
  }

  await fetchResaleWithAuth<{
    ok?: boolean;
    resaleId?: string;
    error?: string;
  }>(
    `/mall/me/resales/${encodeURIComponent(
      normalizedResaleId,
    )}`,
    {
      method: "DELETE",
    },
  );
}