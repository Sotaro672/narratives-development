// frontend/amol/src/pages/ListsPage.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { formatPrice } from "../components/utils/price";

import ProductListingGrid, {
  type ProductListingCardViewModel,
} from "../features/shared/presentation/components/ProductListingGrid";
import ListPagination from "../components/ui/Pagination";

import { useListsPage } from "../features/list/presentation/hooks/useListsPage";

import "../styles/lists-page.css";

export default function ListsPage() {
  const navigate = useNavigate();

  const {
    items,
    page,
    totalPages,
    isLoading,
    canGoPrev,
    canGoNext,
    goPrev,
    goNext,
  } = useListsPage();

  const listingItems: ProductListingCardViewModel[] = items.map((item) => {
    const firstPrice = Array.isArray(item.prices) ? item.prices[0] : undefined;
    const priceAmount = firstPrice?.amount ?? firstPrice?.price;

    return {
      id: item.id,
      title: item.productName?.trim() || item.title.trim() || "商品名未設定",
      imageUrl: item.image,
      brandName: item.brandName,
      reviewAverage: item.reviewAverage,
      reviewCount: item.reviewCount,
      priceLabel: formatPrice(priceAmount, {
        currency: firstPrice?.currency,
      }),
    };
  });

  function handleCartButtonClick() {
    navigate("/cart");
  }

  function handleOpenItem(listId: string) {
    const normalizedListId = listId.trim();
    if (!normalizedListId) return;

    navigate(`/lists/${encodeURIComponent(normalizedListId)}`);
  }

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={handleCartButtonClick}
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading ? (
          <ProductListingGrid
            items={listingItems}
            onOpen={handleOpenItem}
            emptyText="現在、販売中の商品はありません。"
          />
        ) : null}

        {!isLoading && totalPages > 1 ? (
          <ListPagination
            page={page}
            totalPages={totalPages}
            canGoPrev={canGoPrev}
            canGoNext={canGoNext}
            onPrev={goPrev}
            onNext={goNext}
          />
        ) : null}
      </section>
    </Layout>
  );
}