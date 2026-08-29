// frontend\amol\src\features\shared\presentation\components\ProductMediaGallery.tsx

import MediaGallery, { type MediaGalleryItem } from "../../../../components/ui/MediaGallery";

export type ProductMediaGalleryProps = {
  items: MediaGalleryItem[];
  activeIndex: number;
  altFallback?: string;
  placeholderText?: string;
  onPrev: () => void;
  onNext: () => void;
  onSelect: (index: number) => void;
};

export default function ProductMediaGallery({
  items,
  activeIndex,
  altFallback = "商品画像",
  placeholderText = "商品画像はありません。",
  onPrev,
  onNext,
  onSelect,
}: ProductMediaGalleryProps) {
  return (
    <div className="product-detail__image-wrap">
      <MediaGallery
        items={items}
        activeIndex={activeIndex}
        altFallback={altFallback}
        placeholderText={placeholderText}
        className="product-detail__gallery"
        onPrev={onPrev}
        onNext={onNext}
        onSelect={onSelect}
      />
    </div>
  );
}