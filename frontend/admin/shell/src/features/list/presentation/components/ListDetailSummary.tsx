// frontend/admin/shell/src/features/company/presentation/components/ListDetailSummary.tsx

import { useMemo } from "react";

import type { ContractListDetail } from "../../model/listDetail";
import MediaGallery, {
  type MediaGalleryItem,
} from "../../../../shared/ui/MediaGallery/MediaGallery";

type ListDetailSummaryProps = {
  list: ContractListDetail;
};

export default function ListDetailSummary({
  list,
}: ListDetailSummaryProps) {
  const galleryItems = useMemo<MediaGalleryItem[]>(() => {
    const images = Array.isArray(list.images) ? list.images : [];

    return [...images]
      .filter((image) => Boolean(image.id && image.url))
      .sort((a, b) => {
        if (a.displayOrder !== b.displayOrder) {
          return a.displayOrder - b.displayOrder;
        }

        return a.id.localeCompare(b.id);
      })
      .map((image) => ({
        id: image.id,
        url: image.url,
      }));
  }, [list.images]);

  return (
    <>
      <section className="ui-detail-section">
        <MediaGallery
          items={galleryItems}
          altFallback={list.title || list.productName || "出品画像"}
          placeholderText="出品画像はありません。"
        />
      </section>

      <section className="ui-detail-section">
        <p className="ui-detail-section__text">
          {list.description || "-"}
        </p>
      </section>
    </>
  );
}