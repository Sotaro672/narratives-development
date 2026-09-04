// frontend/amol/src/features/market/presentation/components/MarketListView.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import { formatYen } from "../../../../components/utils/price";

import ProductListingGrid, {
  type ProductListingCardViewModel,
} from "../../../shared/presentation/components/ProductListingGrid";
import ListPagination from "../../../../components/ui/Pagination";

import { fetchMarketResales } from "../../infrastructure/marketResaleApi";

import type { MarketResaleListing } from "../../../shared/types/marketResale";

import "../../../../styles/lists-page.css";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

export default function MarketListView() {
  const navigate = useNavigate();

  const [items, setItems] = useState<MarketResaleListing[]>([]);
  const [page, setPage] = useState(DEFAULT_PAGE);
  const [totalPages, setTotalPages] = useState(1);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadMarketItems() {
      setIsLoading(true);

      try {
        const data = await fetchMarketResales({
          page,
          perPage: DEFAULT_PER_PAGE,
          sort: "createdAt",
          order: "desc",
        });

        if (cancelled) return;

        setItems(data.items);
        setTotalPages(data.totalPages);

        if (data.page !== page) {
          setPage(data.page);
        }
      } catch {
        if (cancelled) return;

        setItems([]);
        setTotalPages(1);
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }

    void loadMarketItems();

    return () => {
      cancelled = true;
    };
  }, [page]);

  const listingItems: ProductListingCardViewModel[] = items.map((item) => ({
    id: item.id,
    title: item.productName?.trim() || item.tokenName?.trim() || "商品名未設定",
    imageUrl: item.imageUrl,
    brandName: item.brandName,
    metaLines: item.condition ? [item.condition] : [],
    priceLabel: formatYen(item.price),
  }));

  const canGoPrev = page > 1 && !isLoading;
  const canGoNext = page < totalPages && !isLoading;

  function handleCartClick() {
    navigate("/cart");
  }

  function handleItemClick(resaleId: string) {
    const normalizedResaleId = resaleId.trim();
    if (!normalizedResaleId) return;

    navigate(`/market/${encodeURIComponent(normalizedResaleId)}`);
  }

  function handlePrevPage() {
    setPage((currentPage) => Math.max(1, currentPage - 1));
  }

  function handleNextPage() {
    setPage((currentPage) => Math.min(totalPages, currentPage + 1));
  }

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={handleCartClick}
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading ? (
          <ProductListingGrid
            items={listingItems}
            onOpen={handleItemClick}
            emptyText="現在、マーケットに出品されている商品はありません。"
          />
        ) : null}

        {!isLoading && totalPages > 1 ? (
          <ListPagination
            page={page}
            totalPages={totalPages}
            canGoPrev={canGoPrev}
            canGoNext={canGoNext}
            onPrev={handlePrevPage}
            onNext={handleNextPage}
          />
        ) : null}
      </section>
    </Layout>
  );
}