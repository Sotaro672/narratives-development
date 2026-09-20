// frontend/amol/src/features/brand/presentation/components/BrandListSection.tsx

import { useNavigate } from "react-router-dom";

import SectionHeader from "../../../../components/ui/SectionHeader";
import TextState from "../../../../components/ui/TextState";
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
        <SectionHeader title="出品中のリスト" titleAs="h2" />

        <TextState variant="empty" className="brand-page-empty">
          現在このブランドの出品中リストはありません。
        </TextState>
      </section>
    );
  }

  if (listItems.length === 0) {
    return (
      <section className="brand-page-section">
        <SectionHeader
          title="出品中のリスト"
          titleAs="h2"
          right={<span>{listIds.length}件</span>}
        />

        <TextState variant="empty" className="brand-page-empty">
          リスト情報を取得できませんでした。
        </TextState>
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
      <SectionHeader
        title="出品中のリスト"
        titleAs="h2"
        right={<span>{listItems.length}件</span>}
      />

      <ProductListingGrid
        items={listingItems}
        onOpen={handleOpenItem}
        className="brand-page-list-grid"
      />
    </section>
  );
}