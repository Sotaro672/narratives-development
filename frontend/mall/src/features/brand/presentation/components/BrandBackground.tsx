// frontend/mall/src/features/brand/presentation/components/BrandBackground.tsx

import {
  useEffect,
  useState,
} from "react";

import Media from "../../../../components/ui/Media";
import type {
  BrandDetail,
} from "../../../shared/types/brand";

type BrandBackgroundProps = {
  brand: BrandDetail;
};

export default function BrandBackground({
  brand,
}: BrandBackgroundProps) {
  const backgroundImage =
    brand.brandBackgroundImage.trim();

  const brandName =
    brand.brandName.trim() ||
    "ブランド";

  const [
    failed,
    setFailed,
  ] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [backgroundImage]);

  if (
    !backgroundImage ||
    failed
  ) {
    return null;
  }

  return (
    <div className="brand-page-hero">
      <Media
        src={backgroundImage}
        alt={`${brandName}の背景画像`}
        loading="lazy"
        fit="cover"
        onError={() => {
          setFailed(true);
        }}
      />
    </div>
  );
}