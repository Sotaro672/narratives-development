// frontend/admin/shell/src/features/resale/presentation/components/ResaleMediaSection.tsx

import type { Resale } from "../../../../shared/type/resale";
import MediaGallery from "../../../../shared/ui/MediaGallery/MediaGallery";
import { createResaleGalleryItems } from "../model/resalePresentation";

type ResaleMediaSectionProps = {
  resale: Resale | null;
};

export default function ResaleMediaSection({
  resale,
}: ResaleMediaSectionProps) {
  if (!resale) {
    return null;
  }

  const galleryItems = createResaleGalleryItems(resale.images);

  return (
    <>
      <section className="ui-detail-section">
        <MediaGallery
          items={galleryItems}
          altFallback={resale.productName || "再販商品画像"}
          placeholderText="再販画像はありません。"
        />
      </section>

      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">商品説明</h2>
        <p className="ui-detail-section__text">
          {resale.description || "-"}
        </p>
      </section>
    </>
  );
}