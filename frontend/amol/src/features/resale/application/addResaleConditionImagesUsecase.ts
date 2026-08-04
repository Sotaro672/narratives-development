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

type UploadedAndCreatedImage = {
  uploaded:
    ResaleConditionImage;
  created:
    ResaleConditionImage | null;
};

function mergeResaleConditionImage(
  uploaded: ResaleConditionImage,
  created: ResaleConditionImage,
): ResaleConditionImage {
  return {
    ...uploaded,
    ...created,

    id:
      created.id ||
      uploaded.id,

    resaleId:
      created.resaleId ||
      uploaded.resaleId,

    url:
      created.url ||
      uploaded.url,

    objectPath:
      created.objectPath ||
      uploaded.objectPath,

    fileName:
      created.fileName ||
      uploaded.fileName,

    fileSize:
      created.fileSize ||
      uploaded.fileSize,

    mimeType:
      created.mimeType ||
      uploaded.mimeType,

    displayOrder:
      typeof created.displayOrder ===
      "number"
        ? created.displayOrder
        : uploaded.displayOrder,
  };
}

export async function addMyResaleConditionImages(
  params: AddResaleConditionImagesParams,
): Promise<ResaleConditionImage[]> {
  const resaleId =
    params.resaleId.trim();

  if (!resaleId) {
    throw new Error(
      "resaleId is required",
    );
  }

  if (params.files.length === 0) {
    return [];
  }

  const startDisplayOrder =
    params.startDisplayOrder ?? 0;

  const uploadedImages =
    await Promise.all(
      params.files.map(
        (file, index) =>
          uploadResaleConditionImage({
            resaleId,
            file,
            displayOrder:
              startDisplayOrder +
              index,
          }),
      ),
    );

  const registeredImages =
    await Promise.all(
      uploadedImages.map(
        async (
          uploaded,
        ): Promise<UploadedAndCreatedImage> => {
          const created =
            await createResaleConditionImage(
              uploaded,
            );

          return {
            uploaded,
            created,
          };
        },
      ),
    );

  return registeredImages
    .filter(
      (
        result,
      ): result is {
        uploaded:
          ResaleConditionImage;
        created:
          ResaleConditionImage;
      } =>
        result.created !== null,
    )
    .map(
      ({
        uploaded,
        created,
      }) =>
        mergeResaleConditionImage(
          uploaded,
          created,
        ),
    );
}