// frontend/amol/src/features/resale/infrastructure/resaleImageStorage.ts

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import {
  storage,
} from "../../../lib/firebase";

import type {
  ResaleConditionImage,
} from "../../shared/types/resaleTypes";

function createUploadImageId(): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  ) {
    return crypto.randomUUID();
  }

  return `img_${Date.now()}_${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(
  fileName: string,
): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(
    /[^\w.\-()]/g,
    "_",
  );
}

export async function uploadResaleConditionImage(
  params: {
    resaleId: string;
    file: File;
    displayOrder: number;
  },
): Promise<ResaleConditionImage> {
  const imageId =
    createUploadImageId();

  const safeFileName =
    sanitizeStorageFileName(
      params.file.name,
    );

  const objectPath =
    `resale-condition-images/${params.resaleId}` +
    `/${imageId}/${safeFileName}`;

  const storageRef = ref(
    storage,
    objectPath,
  );

  const mimeType =
    params.file.type ||
    "application/octet-stream";

  await uploadBytes(
    storageRef,
    params.file,
    {
      contentType: mimeType,
    },
  );

  const url =
    await getDownloadURL(
      storageRef,
    );

  return {
    id: imageId,
    resaleId: params.resaleId,
    url,
    objectPath,
    fileName: params.file.name,
    fileSize: params.file.size,
    mimeType,
    displayOrder:
      params.displayOrder,
  };
}