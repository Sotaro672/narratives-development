// frontend/admin/shell/src/features/brand/infrastructure/brandAssetStorage.ts

import {
  getDownloadURL,
  ref,
} from "firebase/storage";

import { storage } from "../../../auth/infrastructure/firebaseClient";

export type BrandAssetTarget =
  | "brandIcon"
  | "brandBackgroundImage";

type GetBrandAssetDownloadUrlParams = {
  companyId: string;
  brandId: string;
  target: BrandAssetTarget;
};

function buildBrandAssetPath({
  companyId,
  brandId,
  target,
}: GetBrandAssetDownloadUrlParams): string {
  return [
    "brands",
    companyId,
    brandId,
    target,
  ].join("/");
}

export async function getBrandAssetDownloadUrl(
  params: GetBrandAssetDownloadUrlParams,
): Promise<string> {
  const companyId = params.companyId.trim();
  const brandId = params.brandId.trim();

  if (!companyId || !brandId) {
    return "";
  }

  const objectPath = buildBrandAssetPath({
    companyId,
    brandId,
    target: params.target,
  });

  try {
    return await getDownloadURL(
      ref(storage, objectPath),
    );
  } catch {
    return "";
  }
}

export function getBrandIconUrl(
  companyId: string,
  brandId: string,
): Promise<string> {
  return getBrandAssetDownloadUrl({
    companyId,
    brandId,
    target: "brandIcon",
  });
}

export function getBrandBackgroundImageUrl(
  companyId: string,
  brandId: string,
): Promise<string> {
  return getBrandAssetDownloadUrl({
    companyId,
    brandId,
    target: "brandBackgroundImage",
  });
}