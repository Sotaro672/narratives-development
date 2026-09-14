// frontend/console/shell/src/features/brand/infrastructure/storage/brandAssetStorage.ts

import {
  getDownloadURL,
  ref,
  uploadBytesResumable,
} from "firebase/storage";

import { storage } from "../../../../auth/infrastructure/config/firebaseClient";
import {
  assertImageForStorage,
  getImageContentType,
  type ImageStorageTarget,
} from "../../../../shared/storage/imageStoragePolicy";

type BrandImageTarget = Extract<
  ImageStorageTarget,
  "brandIcon" | "brandBackgroundImage"
>;

export type BrandAssetUploadProgress = {
  transferredBytes: number;
  totalBytes: number;
  percentage: number;
};

export type BrandAssetUploadProgressHandler = (
  progress: BrandAssetUploadProgress,
) => void;

export type UploadBrandAssetToFirebaseStorageParams = {
  companyId: string;
  brandId: string;
  target: BrandImageTarget;
  file: File;
  onProgress?: BrandAssetUploadProgressHandler;
};

export type UploadedBrandAsset = {
  downloadUrl: string;
  objectPath: string;
};

function buildBrandAssetPath(params: {
  companyId: string;
  brandId: string;
  target: BrandImageTarget;
}): string {
  return [
    "brands",
    params.companyId,
    params.brandId,
    params.target,
  ].join("/");
}

function calculateUploadPercentage(
  transferredBytes: number,
  totalBytes: number,
): number {
  if (totalBytes <= 0) {
    return 0;
  }

  return Math.min(
    100,
    Math.max(
      0,
      Math.round((transferredBytes / totalBytes) * 100),
    ),
  );
}

function emitUploadProgress(
  handler: BrandAssetUploadProgressHandler | undefined,
  transferredBytes: number,
  totalBytes: number,
): void {
  handler?.({
    transferredBytes,
    totalBytes,
    percentage: calculateUploadPercentage(
      transferredBytes,
      totalBytes,
    ),
  });
}

export async function uploadBrandAssetToFirebaseStorage(
  params: UploadBrandAssetToFirebaseStorageParams,
): Promise<UploadedBrandAsset> {
  if (!params.companyId) {
    throw new Error(
      "companyId is required before uploading brand asset.",
    );
  }

  if (!params.brandId) {
    throw new Error(
      "brandId is required before uploading brand asset.",
    );
  }

  assertImageForStorage(
    params.file,
    params.target,
  );

  const contentType = getImageContentType(
    params.file,
  );

  const objectPath = buildBrandAssetPath({
    companyId: params.companyId,
    brandId: params.brandId,
    target: params.target,
  });

  const storageRef = ref(
    storage,
    objectPath,
  );

  emitUploadProgress(
    params.onProgress,
    0,
    params.file.size,
  );

  const uploadTask = uploadBytesResumable(
    storageRef,
    params.file,
    {
      contentType,
      cacheControl: "public,max-age=3600",
    },
  );

  await new Promise<void>((resolve, reject) => {
    uploadTask.on(
      "state_changed",
      (snapshot) => {
        emitUploadProgress(
          params.onProgress,
          snapshot.bytesTransferred,
          snapshot.totalBytes,
        );
      },
      (error) => {
        reject(error);
      },
      () => {
        const snapshot = uploadTask.snapshot;

        emitUploadProgress(
          params.onProgress,
          snapshot.totalBytes,
          snapshot.totalBytes,
        );

        resolve();
      },
    );
  });

  const downloadUrl = await getDownloadURL(
    uploadTask.snapshot.ref,
  );

  return {
    downloadUrl,
    objectPath,
  };
}