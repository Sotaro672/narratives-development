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
} from "../../shared/types/resale";

export type CreateResaleListingProgressPhase =
  | "preparing"
  | "uploading"
  | "saving";

export type CreateResaleListingProgress = {
  phase: CreateResaleListingProgressPhase;
  fileName?: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

export type CreateResaleListingProgressHandler = (
  progress: CreateResaleListingProgress,
) => void;

export type CreateResaleListingOptions = {
  onProgress?: CreateResaleListingProgressHandler;
};

export async function createResaleListing(
  params: CreateResaleListingParams,
  options: CreateResaleListingOptions = {},
): Promise<ResaleListing> {
  const expectedUploadCount = params.conditionImages.length;
  const totalBytes = params.conditionImages.reduce(
    (total, file) => total + file.size,
    0,
  );

  options.onProgress?.({
    phase: "preparing",
    transferredBytes: 0,
    totalBytes,
    completedUploadCount: 0,
    expectedUploadCount,
  });

  const created = await createResaleListingRecord({
    assetId: params.assetId,
    tokenBlueprintId: params.tokenBlueprintId,
    productId: params.productId,
    price: params.price,
    condition: params.condition,
    description: params.description,
  });

  const resaleId = created.id;

  if (params.conditionImages.length === 0) {
    options.onProgress?.({
      phase: "saving",
      transferredBytes: 0,
      totalBytes: 0,
      completedUploadCount: 0,
      expectedUploadCount: 0,
    });

    return created;
  }

  const transferredBytesByIndex = new Map<number, number>();
  const completedIndexes = new Set<number>();

  const uploadedImages = await Promise.all(
    params.conditionImages.map((file, index) =>
      uploadResaleConditionImage({
        resaleId,
        file,
        displayOrder: index,
        onProgress: (uploadProgress) => {
          transferredBytesByIndex.set(
            index,
            Math.min(
              uploadProgress.transferredBytes,
              uploadProgress.totalBytes,
            ),
          );

          if (
            uploadProgress.totalBytes <= 0 ||
            uploadProgress.transferredBytes >= uploadProgress.totalBytes
          ) {
            completedIndexes.add(index);
          }

          const transferredBytes = Array.from(
            transferredBytesByIndex.values(),
          ).reduce((total, value) => total + value, 0);

          options.onProgress?.({
            phase: "uploading",
            fileName: file.name,
            transferredBytes: Math.min(
              transferredBytes,
              totalBytes,
            ),
            totalBytes,
            completedUploadCount: completedIndexes.size,
            expectedUploadCount,
          });
        },
      }),
    ),
  );

  options.onProgress?.({
    phase: "saving",
    transferredBytes: totalBytes,
    totalBytes,
    completedUploadCount: expectedUploadCount,
    expectedUploadCount,
  });

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