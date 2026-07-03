// frontend/amol/src/pages/MarketPage.tsx
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchMarketResales,
  type MarketResaleListing,
} from "../features/market/marketApi";

import "../styles/lists-page.css";

type MarketCardItem = MarketResaleListing;

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

function formatPrice(item: MarketCardItem): string {
  const amount =
    typeof item.price === "number"
      ? item.price
      : typeof item.price === "string"
        ? Number(item.price)
        : NaN;

  if (!Number.isFinite(amount)) {
    return "価格未設定";
  }

  return `${amount.toLocaleString("ja-JP")}円`;
}

function getItemTitle(item: MarketCardItem): string {
  if (typeof item.productName === "string" && item.productName.trim() !== "") {
    return item.productName;
  }

  if (typeof item.tokenName === "string" && item.tokenName.trim() !== "") {
    return item.tokenName;
  }

  return "商品名未設定";
}

function getItemBrandName(item: MarketCardItem): string {
  if (typeof item.brandName === "string" && item.brandName.trim() !== "") {
    return item.brandName;
  }

  return "";
}

function getItemImage(item: MarketCardItem): string {
  const image = item as MarketCardItem & {
    image?: unknown;
    imageUrl?: unknown;
    url?: unknown;
  };

  if (typeof image.image === "string" && image.image.trim() !== "") {
    return image.image;
  }

  if (typeof image.imageUrl === "string" && image.imageUrl.trim() !== "") {
    return image.imageUrl;
  }

  if (typeof image.url === "string" && image.url.trim() !== "") {
    return image.url;
  }

  return "";
}

export default function MarketPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<MarketCardItem[]>([]);
  const [page, setPage] = useState(DEFAULT_PAGE);
  const [perPage] = useState(DEFAULT_PER_PAGE);
  const [totalPages, setTotalPages] = useState(1);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function fetchMarketItems() {
      setIsLoading(true);

      try {
        const data = await fetchMarketResales({
          page,
          perPage,
          sort: "updatedAt",
          order: "desc",
        });

        if (cancelled) {
          return;
        }

        setItems(Array.isArray(data.items) ? data.items : []);
        setTotalPages(
          typeof data.totalPages === "number" && data.totalPages > 0
            ? data.totalPages
            : 1,
        );
        setPage(
          typeof data.page === "number" && data.page > 0 ? data.page : page,
        );
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

    void fetchMarketItems();

    return () => {
      cancelled = true;
    };
  }, [page, perPage]);

  const canGoPrev = page > 1 && !isLoading;
  const canGoNext = page < totalPages && !isLoading;

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() => navigate("/cart")}
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading && items.length > 0 && (
          <div className="lists-page-grid">
            {items.map((item) => {
              const cardTitle = getItemTitle(item);
              const cardBrandName = getItemBrandName(item);
              const image = getItemImage(item);

              return (
                <button
                  key={item.id}
                  type="button"
                  className="lists-page-card"
                  onClick={() => navigate(`/market/${item.id}`)}
                >
                  <div className="lists-page-card-image-wrap">
                    {image ? (
                      <img
                        src={image}
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

                    {cardBrandName ? (
                      <p className="lists-page-card-description">
                        {cardBrandName}
                      </p>
                    ) : null}

                    {item.condition ? (
                      <p className="lists-page-card-description">
                        {item.condition}
                      </p>
                    ) : null}

                    <div className="lists-page-card-footer">
                      <span className="lists-page-card-price">
                        {formatPrice(item)}
                      </span>
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        )}

        {!isLoading && items.length === 0 && (
          <div className="lists-page-empty">
            <p>現在、マーケットに出品されている商品はありません。</p>
          </div>
        )}

        {!isLoading && totalPages > 1 && (
          <div className="lists-page-pagination" aria-label="ページ送り">
            <button
              type="button"
              className="lists-page-pagination-button"
              disabled={!canGoPrev}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
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
              onClick={() =>
                setPage((current) => Math.min(totalPages, current + 1))
              }
            >
              次へ
            </button>
          </div>
        )}
      </section>
    </Layout>
  );
}