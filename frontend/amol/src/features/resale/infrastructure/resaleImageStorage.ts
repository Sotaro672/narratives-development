// frontend/amol/src/features/resale/infrastructure/resaleImageStorage.ts

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import {
  storage,
} from "../../../lib/firebase";

export type UploadedResaleConditionImage = {
  id: string;
  resaleId: string;
  url: string;
  displayOrder: number;
};

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
  if (!fileName) {
    return "image";
  }

  return fileName.replace(
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
): Promise<UploadedResaleConditionImage> {
  const imageId = createUploadImageId();
  const safeFileName = sanitizeStorageFileName(params.file.name);

  const objectPath =
    `resale-condition-images/${params.resaleId}` +
    `/${imageId}/${safeFileName}`;

  const storageRef = ref(
    storage,
    objectPath,
  );

  await uploadBytes(
    storageRef,
    params.file,
    {
      contentType:
        params.file.type ||
        "application/octet-stream",
    },
  );

  const url = await getDownloadURL(storageRef);

  return {
    id: imageId,
    resaleId: params.resaleId,
    url,
    displayOrder: params.displayOrder,
  };
}