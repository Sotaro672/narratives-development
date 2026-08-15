// frontend/amol/src/features/market/presentation/components/MarketListView.tsx

import {
  useEffect,
  useState,
} from "react";
import {
  useNavigate,
} from "react-router-dom";

import Layout from "../../../../components/layout/Layout";
import {
  formatYen,
} from "../../../../components/utils/price";

import {
  fetchMarketResales,
} from "../../api/marketResaleApi";

import type {
  MarketResaleListing,
} from "../../../shared/types/marketResale";

import "../../../../styles/lists-page.css";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

function getItemTitle(
  item: MarketResaleListing,
): string {
  if (
    typeof item.productName ===
      "string" &&
    item.productName.trim() !== ""
  ) {
    return item.productName;
  }

  if (
    typeof item.tokenName ===
      "string" &&
    item.tokenName.trim() !== ""
  ) {
    return item.tokenName;
  }

  return "商品名未設定";
}

function getItemBrandName(
  item: MarketResaleListing,
): string {
  if (
    typeof item.brandName ===
      "string" &&
    item.brandName.trim() !== ""
  ) {
    return item.brandName;
  }

  return "";
}

function getItemImage(
  item: MarketResaleListing,
): string {
  const imageSource =
    item as MarketResaleListing & {
      image?: unknown;
      imageUrl?: unknown;
      url?: unknown;
    };

  if (
    typeof imageSource.image ===
      "string" &&
    imageSource.image.trim() !== ""
  ) {
    return imageSource.image;
  }

  if (
    typeof imageSource.imageUrl ===
      "string" &&
    imageSource.imageUrl.trim() !== ""
  ) {
    return imageSource.imageUrl;
  }

  if (
    typeof imageSource.url ===
      "string" &&
    imageSource.url.trim() !== ""
  ) {
    return imageSource.url;
  }

  return "";
}

export default function MarketListView() {
  const navigate =
    useNavigate();

  const [
    items,
    setItems,
  ] = useState<
    MarketResaleListing[]
  >([]);

  const [
    page,
    setPage,
  ] = useState(
    DEFAULT_PAGE,
  );

  const [
    totalPages,
    setTotalPages,
  ] = useState(1);

  const [
    isLoading,
    setIsLoading,
  ] = useState(true);

  useEffect(() => {
    let cancelled =
      false;

    async function loadMarketItems() {
      setIsLoading(true);

      try {
        const data =
          await fetchMarketResales({
            page,
            perPage:
              DEFAULT_PER_PAGE,
            sort:
              "createdAt",
            order:
              "desc",
          });

        if (cancelled) {
          return;
        }

        setItems(
          Array.isArray(
            data.items,
          )
            ? data.items
            : [],
        );

        setTotalPages(
          typeof data.totalPages ===
            "number" &&
          data.totalPages > 0
            ? data.totalPages
            : 1,
        );

        if (
          typeof data.page ===
            "number" &&
          data.page > 0 &&
          data.page !== page
        ) {
          setPage(
            data.page,
          );
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
      cancelled =
        true;
    };
  }, [
    page,
  ]);

  const canGoPrev =
    page > 1 &&
    !isLoading;

  const canGoNext =
    page < totalPages &&
    !isLoading;

  const handleCartClick =
    () => {
      navigate("/cart");
    };

  const handleItemClick =
    (
      resaleId: string,
    ) => {
      navigate(
        `/market/${encodeURIComponent(
          resaleId,
        )}`,
      );
    };

  const handlePrevPage =
    () => {
      setPage(
        (currentPage) =>
          Math.max(
            1,
            currentPage - 1,
          ),
      );
    };

  const handleNextPage =
    () => {
      setPage(
        (currentPage) =>
          Math.min(
            totalPages,
            currentPage + 1,
          ),
      );
    };

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={
        handleCartClick
      }
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading &&
        items.length > 0 ? (
          <div className="lists-page-grid">
            {items.map(
              (item) => {
                const cardTitle =
                  getItemTitle(
                    item,
                  );

                const cardBrandName =
                  getItemBrandName(
                    item,
                  );

                const imageUrl =
                  getItemImage(
                    item,
                  );

                return (
                  <button
                    key={
                      item.id
                    }
                    type="button"
                    className="lists-page-card"
                    onClick={() => {
                      handleItemClick(
                        item.id,
                      );
                    }}
                  >
                    <div className="lists-page-card-image-wrap">
                      {imageUrl ? (
                        <img
                          src={
                            imageUrl
                          }
                          alt={
                            cardTitle
                          }
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
                      <h2 className="lists-page-card-title">
                        {
                          cardTitle
                        }
                      </h2>

                      {cardBrandName ? (
                        <p className="lists-page-card-description">
                          {
                            cardBrandName
                          }
                        </p>
                      ) : null}

                      {item.condition ? (
                        <p className="lists-page-card-description">
                          {
                            item.condition
                          }
                        </p>
                      ) : null}

                      <div className="lists-page-card-footer">
                        <span className="lists-page-card-price">
                          {formatYen(
                            item.price,
                          )}
                        </span>
                      </div>
                    </div>
                  </button>
                );
              },
            )}
          </div>
        ) : null}

        {!isLoading &&
        items.length === 0 ? (
          <div className="lists-page-empty">
            <p>
              現在、マーケットに出品されている商品はありません。
            </p>
          </div>
        ) : null}

        {!isLoading &&
        totalPages > 1 ? (
          <div
            className="lists-page-pagination"
            aria-label="ページ送り"
          >
            <button
              type="button"
              className="lists-page-pagination-button"
              disabled={
                !canGoPrev
              }
              onClick={
                handlePrevPage
              }
            >
              前へ
            </button>

            <span className="lists-page-pagination-status">
              {page} /{" "}
              {totalPages}
            </span>

            <button
              type="button"
              className="lists-page-pagination-button"
              disabled={
                !canGoNext
              }
              onClick={
                handleNextPage
              }
            >
              次へ
            </button>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}