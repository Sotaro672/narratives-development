// frontend/amol/src/features/market/presentation/components/MarketListView.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import { formatYen } from "../../../../components/utils/price";
import { fetchMarketResales } from "../../infrastructure/marketResaleApi";

import type { MarketResaleListing } from "../../../shared/types/marketResale";

import "../../../../styles/lists-page.css";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

function getItemTitle(item: MarketResaleListing): string {
  return item.productName || item.tokenName || "商品名未設定";
}

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

        if (cancelled) {
          return;
        }

        setItems(data.items);
        setTotalPages(data.totalPages);

        if (data.page !== page) {
          setPage(data.page);
        }
      } catch {
        if (cancelled) {
          return;
        }

        setItems([]);
        setTotalPages(1);
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadMarketItems();

    return () => {
      cancelled = true;
    };
  }, [page]);

  const canGoPrev = page > 1 && !isLoading;
  const canGoNext = page < totalPages && !isLoading;

  const handleCartClick = () => {
    navigate("/cart");
  };

  const handleItemClick = (resaleId: string) => {
    navigate(`/market/${encodeURIComponent(resaleId)}`);
  };

  const handlePrevPage = () => {
    setPage((currentPage) => Math.max(1, currentPage - 1));
  };

  const handleNextPage = () => {
    setPage((currentPage) => Math.min(totalPages, currentPage + 1));
  };

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={handleCartClick}
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading && items.length > 0 ? (
          <div className="lists-page-grid">
            {items.map((item) => {
              const cardTitle = getItemTitle(item);
              const imageUrl = item.imageUrl || "";

              return (
                <button
                  key={item.id}
                  type="button"
                  className="lists-page-card"
                  onClick={() => handleItemClick(item.id)}
                >
                  <div className="lists-page-card-image-wrap">
                    {imageUrl ? (
                      <img
                        src={imageUrl}
                        alt={cardTitle}
                        className="lists-page-card-image"
                        loading="lazy"
                      />
                    ) : (
                      <div className="lists-page-card-image-placeholder">
                        No Image
                      </div>
                    )}
                  </div>

                  <div className="lists-page-card-body">
                    <h2 className="lists-page-card-title">{cardTitle}</h2>

                    {item.brandName ? (
                      <p className="lists-page-card-description">
                        {item.brandName}
                      </p>
                    ) : null}

                    {item.condition ? (
                      <p className="lists-page-card-description">
                        {item.condition}
                      </p>
                    ) : null}

                    <div className="lists-page-card-footer">
                      <span className="lists-page-card-price">
                        {formatYen(item.price)}
                      </span>
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        ) : null}

        {!isLoading && items.length === 0 ? (
          <div className="lists-page-empty">
            <p>現在、マーケットに出品されている商品はありません。</p>
          </div>
        ) : null}

        {!isLoading && totalPages > 1 ? (
          <div className="lists-page-pagination" aria-label="ページ送り">
            <button
              type="button"
              className="lists-page-pagination-button"
              disabled={!canGoPrev}
              onClick={handlePrevPage}
            >
              前へ
            </button>

            <span className="lists-page-pagination-status">
              {page} / {totalPages}
            </span>

            <button
              type="button"
              className="lists-page-pagination-button"
              disabled={!canGoNext}
              onClick={handleNextPage}
            >
              次へ
            </button>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}