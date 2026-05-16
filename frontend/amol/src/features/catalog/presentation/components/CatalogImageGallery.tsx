// frontend/amol/src/features/catalog/components/CatalogImageGallery.tsx
import type { TouchEvent } from "react";

import MediaGallery, {
  type MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";
import type {
  CatalogListImage,
  CatalogProductBlueprint,
} from "../../types";

type CatalogImageGalleryProps = {
  activeImage: CatalogListImage | undefined;
  activeImageIndex: number;
  catalogImages: CatalogListImage[];
  productBlueprint: CatalogProductBlueprint;
  hasMultipleImages: boolean;
  onPrevImage: () => void;
  onNextImage: () => void;
  onSelectImage: (index: number) => void;
  onTouchStart: (event: TouchEvent<HTMLDivElement>) => void;
  onTouchEnd: (event: TouchEvent<HTMLDivElement>) => void;
};

export default function CatalogImageGallery({
  activeImage,
  activeImageIndex,
  catalogImages,
  productBlueprint,
  hasMultipleImages,
  onPrevImage,
  onNextImage,
  onSelectImage,
  onTouchStart,
  onTouchEnd,
}: CatalogImageGalleryProps) {
  const mediaItems: MediaGalleryItem[] = catalogImages.map((image) => ({
    id: image.id,
    url: image.url,
    fileName: image.fileName,
  }));

  return (
    <MediaGallery
      items={activeImage?.url ? mediaItems : []}
      activeIndex={activeImageIndex}
      altFallback={productBlueprint.productName}
      placeholderText="No Image"
      className="catalog-page-media"
      onPrev={onPrevImage}
      onNext={onNextImage}
      onSelect={onSelectImage}
      onTouchStart={hasMultipleImages ? onTouchStart : undefined}
      onTouchEnd={hasMultipleImages ? onTouchEnd : undefined}
    />
  );
}