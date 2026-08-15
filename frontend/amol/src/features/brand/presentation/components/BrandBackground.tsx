// frontend/amol/src/features/brand/presentation/components/BrandBackground.tsx

import {
  useEffect,
  useState,
} from "react";

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
      <img
        className="brand-page-hero-image"
        src={backgroundImage}
        alt={`${brandName}の背景画像`}
        loading="lazy"
        onError={() => {
          setFailed(true);
        }}
      />
    </div>
  );
}