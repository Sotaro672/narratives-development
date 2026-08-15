// frontend/amol/src/features/resale/application/addResaleConditionImagesUsecase.ts

import {
  createResaleConditionImage,
} from "../api/resaleConditionImageApi";

import {
  uploadResaleConditionImage,
} from "../infrastructure/resaleImageStorage";

import type {
  AddResaleConditionImagesParams,
  ResaleConditionImage,
} from "../../shared/types/resaleTypes";

export async function addMyResaleConditionImages(
  params: AddResaleConditionImagesParams,
): Promise<ResaleConditionImage[]> {
  const resaleId = params.resaleId.trim();

  if (!resaleId) {
    throw new Error("resaleId is required");
  }

  if (params.files.length === 0) {
    return [];
  }

  const startDisplayOrder = params.startDisplayOrder ?? 0;

  const uploadedImages = await Promise.all(
    params.files.map((file, index) =>
      uploadResaleConditionImage({
        resaleId,
        file,
        displayOrder: startDisplayOrder + index,
      }),
    ),
  );

  return Promise.all(
    uploadedImages.map((uploaded) =>
      createResaleConditionImage({
        id: uploaded.id,
        resaleId: uploaded.resaleId,
        url: uploaded.url,
        displayOrder: uploaded.displayOrder,
      }),
    ),
  );
}