// frontend/amol/src/features/brand/presentation/components/BrandIcon.tsx

import MediaIcon from "../../../../components/ui/MediaIcon";

import type {
  BrandDetail,
} from "../../types/brand";

import {
  buildBrandInitial,
} from "../utils/buildBrandInitial";

type BrandIconProps = {
  brand: BrandDetail;
};

export default function BrandIcon({
  brand,
}: BrandIconProps) {
  const brandName =
    brand.brandName.trim();

  const brandIcon =
    brand.brandIcon.trim();

  return (
    <MediaIcon
      src={brandIcon}
      alt={
        brandName
          ? `${brandName}のブランドアイコン`
          : "ブランドアイコン"
      }
      fallback={buildBrandInitial(
        brandName,
      )}
      size="lg"
      shape="circle"
      className="brand-page-icon"
    />
  );
}