// frontend/admin/shell/src/features/resale/presentation/components/ResaleMediaSection.tsx

import type { AvatarResale } from "../../../../shared/type/avatar";
import MediaGallery from "../../../../shared/ui/MediaGallery/MediaGallery";
import { createResaleGalleryItems } from "../model/resalePresentation";

type ResaleMediaSectionProps = {
  resale: AvatarResale | null;
};

export default function ResaleMediaSection({
  resale,
}: ResaleMediaSectionProps) {
  if (!resale) {
    return null;
  }

  const galleryItems = createResaleGalleryItems(resale.images);

  return (
    <section className="ui-detail-section">
      <MediaGallery
        items={galleryItems}
        altFallback={resale.productName || "再販商品画像"}
        placeholderText="再販画像はありません。"
      />
    </section>
  );
}