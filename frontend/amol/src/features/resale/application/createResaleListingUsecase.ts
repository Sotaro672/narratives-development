// frontend/amol/src/features/resale/application/createResaleListingUsecase.ts

import {
  createResaleListingRecord,
  updatePrimaryResaleImage,
} from "../api/resaleListingApi";

import {
  createResaleConditionImage,
} from "../api/resaleConditionImageApi";

import {
  uploadResaleConditionImage,
} from "../infrastructure/resaleImageStorage";

import type {
  CreateResaleListingParams,
  ResaleListing,
} from "../../shared/types/resaleTypes";

export async function createResaleListing(
  params: CreateResaleListingParams,
): Promise<ResaleListing> {
  const created = await createResaleListingRecord({
    assetId: params.assetId,
    tokenBlueprintId: params.tokenBlueprintId,
    productId: params.productId,
    brandId: params.brandId,
    productBlueprintId: params.productBlueprintId,
    price: params.price,
    condition: params.condition,
    description: params.description,
  });

  const resaleId = created.id;

  if (params.conditionImages.length === 0) {
    return created;
  }

  const uploadedImages = await Promise.all(
    params.conditionImages.map((file, index) =>
      uploadResaleConditionImage({
        resaleId,
        file,
        displayOrder: index,
      }),
    ),
  );

  await Promise.all(
    uploadedImages.map(createResaleConditionImage),
  );

  const primaryImage = uploadedImages[0];

  if (!primaryImage) {
    return created;
  }

  return updatePrimaryResaleImage({
    resaleId,
    imageId: primaryImage.id,
  });
}