// frontend/amol/src/features/market/presentation/components/MarketResaleGallery.tsx

import MediaGallery, {
  type MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";

type MarketResaleGalleryProps = {
  items:
    MediaGalleryItem[];
  activeIndex: number;
  altFallback: string;
  onPrev: () => void;
  onNext: () => void;
  onSelect: (
    index: number,
  ) => void;
};

export default function MarketResaleGallery({
  items,
  activeIndex,
  altFallback,
  onPrev,
  onNext,
  onSelect,
}: MarketResaleGalleryProps) {
  return (
    <div className="market-detail-page__image-wrap">
      <MediaGallery
        items={items}
        activeIndex={
          activeIndex
        }
        altFallback={
          altFallback
        }
        placeholderText="No Image"
        className="market-detail-page__media-gallery"
        onPrev={onPrev}
        onNext={onNext}
        onSelect={
          onSelect
        }
      />
    </div>
  );
}