// frontend/amol/src/features/resale/api/resaleConditionImageApi.ts

import {
  fetchResaleWithAuth,
  type ApiDataResponse,
} from "./resaleHttpClient";

import type {
  ResaleConditionImage,
  ResaleImageIdentifier,
} from "../../shared/types/resaleTypes";

type CreateResaleConditionImageParams = Pick<
  ResaleConditionImage,
  "id" | "resaleId" | "url" | "displayOrder"
>;

export async function createResaleConditionImage(
  image: CreateResaleConditionImageParams,
): Promise<ResaleConditionImage> {
  const resaleId = image.resaleId.trim();

  if (!resaleId) {
    throw new Error("resaleId is required");
  }

  const result = await fetchResaleWithAuth<
    ApiDataResponse<ResaleConditionImage>
  >(
    `/mall/me/resales/${encodeURIComponent(resaleId)}/images`,
    {
      method: "POST",
      body: JSON.stringify({
        id: image.id,
        url: image.url,
        displayOrder: image.displayOrder,
      }),
    },
  );

  return result.data;
}

export async function listMyResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const normalizedResaleId = resaleId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  const result = await fetchResaleWithAuth<
    ApiDataResponse<ResaleConditionImage[]>
  >(
    `/mall/me/resales/${encodeURIComponent(normalizedResaleId)}/images`,
    {
      method: "GET",
    },
  );

  return result.data;
}

export async function deleteMyResaleConditionImage({
  resaleId,
  imageId,
}: ResaleImageIdentifier): Promise<void> {
  const normalizedResaleId = resaleId.trim();
  const normalizedImageId = imageId.trim();

  if (!normalizedResaleId) {
    throw new Error("resaleId is required");
  }

  if (!normalizedImageId) {
    throw new Error("imageId is required");
  }

  await fetchResaleWithAuth<{
    ok: boolean;
  }>(
    `/mall/me/resales/${encodeURIComponent(
      normalizedResaleId,
    )}/images/${encodeURIComponent(normalizedImageId)}`,
    {
      method: "DELETE",
    },
  );
}