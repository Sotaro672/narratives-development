// frontend/console/brand/src/infrastructure/storage/brandAssetStorage.ts
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { storage } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

export type BrandAssetTarget = "brandIcon" | "brandBackgroundImage";

function getFileExtension(file: File): string {
  const fileName = file.name ?? "";
  const parts = fileName.split(".");
  const ext = parts.length > 1 ? parts.pop() : "";

  return ext ? `.${ext}` : "";
}

function buildBrandAssetPath(params: {
  companyId: string;
  brandId: string;
  target: BrandAssetTarget;
  file: File;
}): string {
  const extension = getFileExtension(params.file);
  const timestamp = Date.now();

  return [
    "brands",
    params.companyId,
    params.brandId,
    params.target,
    `${timestamp}${extension}`,
  ].join("/");
}

export async function uploadBrandAssetToFirebaseStorage(params: {
  companyId: string;
  brandId: string;
  target: BrandAssetTarget;
  file: File;
}): Promise<{
  downloadUrl: string;
  objectPath: string;
}> {
  if (!params.companyId) {
    throw new Error("companyId is required before uploading brand asset.");
  }

  if (!params.brandId) {
    throw new Error("brandId is required before uploading brand asset.");
  }

  const objectPath = buildBrandAssetPath(params);
  const storageRef = ref(storage, objectPath);

  await uploadBytes(storageRef, params.file, {
    contentType: params.file.type || "application/octet-stream",
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
  };
}