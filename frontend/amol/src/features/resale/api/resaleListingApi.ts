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
} from "../../shared/types/resale";

export async function createResaleListingRecord(
  params: CreateResaleListingRecordParams,
): Promise<ResaleListing> {
  const result = await fetchResaleWithAuth<ApiDataResponse<ResaleListing>>(
    "/mall/me/resales",
    {
      method: "POST",
      json: {
        assetId: params.assetId,
        tokenBlueprintId: params.tokenBlueprintId,
        productId: params.productId,
        brandId: params.brandId,
        productBlueprintId: params.productBlueprintId,
        price: params.price,
        condition: params.condition,
        description: params.description,
      },
    },
  );

  return result.data;
}

export async function updateResaleListing(
  params: UpdateResaleListingParams,
): Promise<ResaleListing> {
  const resaleId = params.resaleId.trim();

  if (!resaleId) {
    throw new Error("resaleId is required");
  }

  const body: {
    price?: number;
    condition?: string;
    description?: string;
    status?: string;
  } = {};

  if (params.price !== undefined) {
    body.price = params.price;
  }

  if (params.condition !== undefined) {
    body.condition = params.condition;
  }

  if (params.description !== undefined) {
    body.description = params.description;
  }

  if (params.status !== undefined) {
    body.status = params.status;
  }

  const result = await fetchResaleWithAuth<ApiDataResponse<ResaleListing>>(
    `/mall/me/resales/${encodeURIComponent(resaleId)}`,
    {
      method: "PUT",
      json: body,
    },
  );

  return result.data;
}

export async function getMyResaleListing(
  resaleId: string,
): Promise<ResaleListing> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  const result = await fetchResaleWithAuth<ApiDataResponse<ResaleListing>>(
    `/mall/me/resales/${encodeURIComponent(normalizedResaleId)}`,
    {
      method: "GET",
    },
  );

  return result.data;
}

export async function listMyResaleListings(
  params: ListMyResaleListingsParams = {},
): Promise<ListMyResaleListingsResponse> {
  return fetchResaleWithAuth<ListMyResaleListingsResponse>(
    "/mall/me/resales",
    {
      method: "GET",
      query: {
        page: params.page ?? 1,
        perPage: params.perPage ?? 50,
      },
    },
  );
}

export async function updatePrimaryResaleImage({
  resaleId,
  imageId,
}: ResaleImageIdentifier): Promise<ResaleListing> {
  const normalizedResaleId = resaleId.trim();
  const normalizedImageId = imageId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  if (!normalizedImageId) {
    throw new Error("imageId is required");
  }

  const result = await fetchResaleWithAuth<ApiDataResponse<ResaleListing>>(
    `/mall/me/resales/${encodeURIComponent(normalizedResaleId)}/primary-image`,
    {
      method: "PUT",
      json: {
        imageId: normalizedImageId,
      },
    },
  );

  return result.data;
}

export async function deleteResaleListing(
  resaleId: string,
): Promise<void> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  await fetchResaleWithAuth<{
    ok: boolean;
    resaleId: string;
  }>(
    `/mall/me/resales/${encodeURIComponent(normalizedResaleId)}`,
    {
      method: "DELETE",
    },
  );
}