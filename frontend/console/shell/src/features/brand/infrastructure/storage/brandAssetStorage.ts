// frontend/console/shell/src/features/brand/infrastructure/storage/brandAssetStorage.ts

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import { storage } from "../../../../auth/infrastructure/config/firebaseClient";

import type { BrandImageTarget } from "../../config/brandImagePolicy.generated";
import { validateBrandImage } from "../../application/brandImageValidation";

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

export async function uploadBrandAssetToFirebaseStorage(
  params: {
    companyId: string;
    brandId: string;
    target: BrandImageTarget;
    file: File;
  },
): Promise<{
  downloadUrl: string;
  objectPath: string;
}> {
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

  const validation = validateBrandImage(
    params.file,
    params.target,
  );

  if (!validation.valid) {
    throw new Error(validation.message);
  }

  if (!params.file.type) {
    throw new Error(
      "画像のMIMEタイプを取得できません。",
    );
  }

  const objectPath = buildBrandAssetPath({
    companyId: params.companyId,
    brandId: params.brandId,
    target: params.target,
  });

  const storageRef = ref(storage, objectPath);

  await uploadBytes(
    storageRef,
    params.file,
    {
      contentType: params.file.type,
      cacheControl: "public,max-age=3600",
    },
  );

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
  };
}