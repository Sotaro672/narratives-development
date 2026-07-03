// frontend/amol/src/pages/LikesPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";

import "../styles/lists-page.css";

type LikePriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

type LikeListItem = {
  id: string;
  title?: string;
  description?: string;
  image?: string;
  imageUrl?: string;
  price?: number;
  prices?: LikePriceRow[];

  listId?: string;
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  productId?: string;
  brandId?: string;
  brandName?: string;
  productName?: string;
  likedAt?: string;
  createdAt?: string;
  updatedAt?: string;
};

type LikeIndexResponse = {
  items: LikeListItem[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

type CatalogProductBlueprint = {
  id?: string;
  productName?: string;
  brandName?: string;
};

type MallCatalogResponse = {
  productBlueprint?: CatalogProductBlueprint;
};

type LikeCardItem = LikeListItem & {
  productName?: string;
  brandName?: string;
};

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 20;

function formatPrice(item: LikeListItem): string {
  const prices = Array.isArray(item.prices) ? item.prices : [];

  const first = prices[0];
  const rawAmount = first?.amount ?? first?.price ?? item.price;
  const amount =
    typeof rawAmount === "number"
      ? rawAmount
      : typeof rawAmount === "string"
        ? Number(rawAmount)
        : NaN;

  const currency =
    typeof first?.currency === "string" && first.currency.trim() !== ""
      ? first.currency.toUpperCase()
      : "JPY";

  if (!Number.isFinite(amount)) {
    return "価格未設定";
  }

  if (currency === "JPY") {
    return `${amount.toLocaleString("ja-JP")}円`;
  }

  return `${amount.toLocaleString("ja-JP")} ${currency}`;
}

function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/$/, "");
  }

  return "";
}

function getItemImage(item: LikeListItem): string {
  if (typeof item.image === "string" && item.image.trim() !== "") {
    return item.image;
  }

  if (typeof item.imageUrl === "string" && item.imageUrl.trim() !== "") {
    return item.imageUrl;
  }

  return "";
}

function getItemTitle(item: LikeCardItem): string {
  if (typeof item.productName === "string" && item.productName.trim() !== "") {
    return item.productName;
  }

  if (typeof item.title === "string" && item.title.trim() !== "") {
    return item.title;
  }

  return "商品名未設定";
}

function getCatalogId(item: LikeListItem): string {
  if (typeof item.listId === "string" && item.listId.trim() !== "") {
    return item.listId;
  }

  if (typeof item.productId === "string" && item.productId.trim() !== "") {
    return item.productId;
  }

  return item.id;
}

function getNavigateId(item: LikeListItem): string {
  if (typeof item.listId === "string" && item.listId.trim() !== "") {
    return item.listId;
  }

  return item.id;
}

async function fetchCatalogCardItem(
  apiBaseUrl: string,
  item: LikeListItem,
): Promise<LikeCardItem> {
  const catalogId = getCatalogId(item);

  if (!catalogId) {
    return item;
  }

  try {
    const response = await fetch(
      `${apiBaseUrl}/mall/catalog/${encodeURIComponent(catalogId)}`,
      {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
        credentials: "include",
      },
    );

    const contentType = response.headers.get("content-type") ?? "";

    if (!response.ok || !contentType.includes("application/json")) {
      return item;
    }

    const data = (await response.json()) as MallCatalogResponse;
    const productBlueprint = data.productBlueprint;

    return {
      ...item,
      productName:
        typeof productBlueprint?.productName === "string"
          ? productBlueprint.productName
          : item.productName,
      brandName:
        typeof productBlueprint?.brandName === "string"
          ? productBlueprint.brandName
          : item.brandName,
    };
  } catch {
    return item;
  }
}

export default function LikesPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<LikeCardItem[]>([]);
  const [page, setPage] = useState(DEFAULT_PAGE);
  const [perPage] = useState(DEFAULT_PER_PAGE);
  const [totalPages, setTotalPages] = useState(1);
  const [isLoading, setIsLoading] = useState(true);

  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);

  useEffect(() => {
    let cancelled = false;

    async function fetchLikes() {
      setIsLoading(true);

      try {
        if (!apiBaseUrl) {
          throw new Error("API Base URLが未設定です。");
        }

        const searchParams = new URLSearchParams({
          page: String(page),
          perPage: String(perPage),
        });

        const response = await fetch(
          `${apiBaseUrl}/mall/likes?${searchParams.toString()}`,
          {
            method: "GET",
            headers: {
              Accept: "application/json",
            },
            credentials: "include",
          },
        );

        const contentType = response.headers.get("content-type") ?? "";

        if (!contentType.includes("application/json")) {
          throw new Error("お気に入り一覧APIがJSON以外を返しました。");
        }

        const data = (await response.json()) as Partial<LikeIndexResponse>;

        if (!response.ok) {
          throw new Error("お気に入り一覧の取得に失敗しました。");
        }

        if (!Array.isArray(data.items)) {
          throw new Error("お気に入り一覧APIのitemsが配列ではありません。");
        }

        const catalogItems = await Promise.all(
          data.items.map((item) => fetchCatalogCardItem(apiBaseUrl, item)),
        );

        if (cancelled) {
          return;
        }

        setItems(catalogItems);
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

    void fetchLikes();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl, page, perPage]);

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
              const cardBrandName = item.brandName || "";
              const image = getItemImage(item);
              const navigateId = getNavigateId(item);

              return (
                <button
                  key={item.id}
                  type="button"
                  className="lists-page-card"
                  onClick={() => navigate(`/favorites/${navigateId}`)}
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