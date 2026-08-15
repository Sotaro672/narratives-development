// frontend/amol/src/features/resale/api/publicResaleApi.ts

import {
  fetchPublicResale,
  type ApiDataResponse,
} from "./resaleHttpClient";

import type {
  ListMyResaleListingsResponse,
  ListResaleListingsByAvatarIdParams,
  ResaleConditionImage,
} from "../../shared/types/resaleTypes";

export async function listResaleListingsByAvatarId(
  params: ListResaleListingsByAvatarIdParams,
): Promise<ListMyResaleListingsResponse> {
  const avatarId = params.avatarId.trim();

  if (!avatarId) {
    throw new Error("avatarId is required");
  }

  return fetchPublicResale<ListMyResaleListingsResponse>(
    `/mall/resales/avatar/${encodeURIComponent(avatarId)}`,
    {
      method: "GET",
      query: {
        page: params.page ?? 1,
        perPage: params.perPage ?? 50,
      },
    },
  );
}

export async function listPublicResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  const result = await fetchPublicResale<
    ApiDataResponse<ResaleConditionImage[]>
  >(
    `/mall/resales/${encodeURIComponent(normalizedResaleId)}/images`,
    {
      method: "GET",
    },
  );

  return result.data;
}