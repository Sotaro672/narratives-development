// frontend/amol/src/features/resale/infrastructure/resaleImageStorage.ts

import {
  getDownloadURL,
  ref,
  uploadBytesResumable,
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

export type ResaleImageUploadProgress = {
  transferredBytes: number;
  totalBytes: number;
  percentage: number;
};

export type ResaleImageUploadProgressHandler = (
  progress: ResaleImageUploadProgress,
) => void;

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

function calculateUploadPercentage(
  transferredBytes: number,
  totalBytes: number,
): number {
  if (totalBytes <= 0) {
    return 100;
  }

  return Math.min(
    100,
    Math.max(
      0,
      Math.round(
        (transferredBytes / totalBytes) * 100,
      ),
    ),
  );
}

export async function uploadResaleConditionImage(
  params: {
    resaleId: string;
    file: File;
    displayOrder: number;
    onProgress?: ResaleImageUploadProgressHandler;
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

  const uploadTask = uploadBytesResumable(
    storageRef,
    params.file,
    {
      contentType:
        params.file.type ||
        "application/octet-stream",
    },
  );

  await new Promise<void>((resolve, reject) => {
    uploadTask.on(
      "state_changed",
      (snapshot) => {
        params.onProgress?.({
          transferredBytes: snapshot.bytesTransferred,
          totalBytes: snapshot.totalBytes,
          percentage: calculateUploadPercentage(
            snapshot.bytesTransferred,
            snapshot.totalBytes,
          ),
        });
      },
      (error) => {
        reject(error);
      },
      () => {
        const snapshot = uploadTask.snapshot;

        params.onProgress?.({
          transferredBytes: snapshot.totalBytes,
          totalBytes: snapshot.totalBytes,
          percentage: 100,
        });

        resolve();
      },
    );
  });

  const url = await getDownloadURL(storageRef);

  return {
    id: imageId,
    resaleId: params.resaleId,
    url,
    displayOrder: params.displayOrder,
  };
}