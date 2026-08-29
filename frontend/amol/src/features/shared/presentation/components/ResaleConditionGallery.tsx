// frontend/amol/src/features/shared/presentation/componentns/ResaleConditionGallery.tsx

import MediaGallery, {
  type MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";

export type ResaleConditionGalleryProps = {
  items: MediaGalleryItem[];
  activeIndex: number;
  altFallback?: string;
  placeholderText?: string;
  onPrev: () => void;
  onNext: () => void;
  onSelect: (index: number) => void;
};

export default function ResaleConditionGallery({
  items,
  activeIndex,
  altFallback = "商品状態の写真",
  placeholderText = "商品状態の写真はありません。",
  onPrev,
  onNext,
  onSelect,
}: ResaleConditionGalleryProps) {
  return (
    <div className="resale-product-detail__image-wrap">
      <MediaGallery
        items={items}
        activeIndex={activeIndex}
        altFallback={altFallback}
        placeholderText={placeholderText}
        className="resale-product-detail__gallery"
        onPrev={onPrev}
        onNext={onNext}
        onSelect={onSelect}
      />
    </div>
  );
}