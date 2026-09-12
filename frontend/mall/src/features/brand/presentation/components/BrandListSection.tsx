// frontend/amol/src/features/brand/presentation/components/BrandListSection.tsx

import { useNavigate } from "react-router-dom";

import ProductListingGrid, {
  type ProductListingCardViewModel,
} from "../../../shared/presentation/components/ProductListingGrid";
import type { MallListItem } from "../../../shared/types/list";

import { formatBrandListPrice } from "../utils/formatBrandListPrice";

type BrandListSectionProps = {
  listIds: string[];
  listItems: MallListItem[];
};

export default function BrandListSection({
  listIds,
  listItems,
}: BrandListSectionProps) {
  const navigate = useNavigate();

  if (listIds.length === 0) {
    return (
      <section className="brand-page-section">
        <h2>出品中のリスト</h2>

        <div className="brand-page-empty">
          現在このブランドの出品中リストはありません。
        </div>
      </section>
    );
  }

  if (listItems.length === 0) {
    return (
      <section className="brand-page-section">
        <div className="brand-page-section-header">
          <h2>出品中のリスト</h2>
          <span>{listIds.length}件</span>
        </div>

        <div className="brand-page-empty">
          リスト情報を取得できませんでした。
        </div>
      </section>
    );
  }

  const listingItems: ProductListingCardViewModel[] = listItems.map((item) => ({
    id: item.id,
    title: item.title.trim() || "商品名未設定",
    imageUrl: item.image,
    priceLabel: formatBrandListPrice(item.prices),
  }));

  const handleOpenItem = (listId: string) => {
    const normalizedListId = listId.trim();
    if (!normalizedListId) {
      return;
    }

    navigate(`/lists/${encodeURIComponent(normalizedListId)}`);
  };

  return (
    <section className="brand-page-section">
      <div className="brand-page-section-header">
        <h2>出品中のリスト</h2>
        <span>{listItems.length}件</span>
      </div>

      <ProductListingGrid
        items={listingItems}
        onOpen={handleOpenItem}
        className="brand-page-list-grid"
      />
    </section>
  );
}