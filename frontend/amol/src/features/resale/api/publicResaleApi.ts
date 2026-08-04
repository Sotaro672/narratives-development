// frontend/amol/src/features/resale/api/publicResaleApi.ts

import {
  fetchPublicResale,
} from "./resaleHttpClient";

import type {
  ListMyResaleListingsResponse,
  ListResaleListingsByAvatarIdParams,
  ResaleConditionImage,
} from "../../shared/types/resaleTypes";

type PublicResaleConditionImageListResponse = {
  data?: ResaleConditionImage[] | null;
  items?: ResaleConditionImage[];
  error?: string;
};

export async function listResaleListingsByAvatarId(
  params: ListResaleListingsByAvatarIdParams,
): Promise<ListMyResaleListingsResponse> {
  const avatarId =
    params.avatarId.trim();

  const page =
    params.page ?? 1;

  const perPage =
    params.perPage ?? 50;

  if (!avatarId) {
    return {
      items: [],
      totalCount: 0,
      totalPages: 0,
      page,
      perPage,
    };
  }

  const searchParams =
    new URLSearchParams();

  searchParams.set(
    "page",
    String(page),
  );

  searchParams.set(
    "perPage",
    String(perPage),
  );

  return fetchPublicResale<
    ListMyResaleListingsResponse
  >(
    `/mall/resales/avatar/${encodeURIComponent(
      avatarId,
    )}?${searchParams.toString()}`,
    {
      method: "GET",
    },
  );
}

export async function listPublicResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const normalizedResaleId =
    resaleId.trim();

  if (!normalizedResaleId) {
    return [];
  }

  const result =
    await fetchPublicResale<
      PublicResaleConditionImageListResponse
    >(
      `/mall/resales/${encodeURIComponent(
        normalizedResaleId,
      )}/images`,
      {
        method: "GET",
      },
    );

  return (
    result.data ??
    result.items ??
    []
  );
}