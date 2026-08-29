// frontend/amol/src/pages/LikesPage.tsx

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import ListPagination from "../components/ui/Pagination";
import { formatPrice } from "../components/utils/price";
import {
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
} from "../features/like/constants";
import type {
  LikeCardItem,
  LikeCatalogResponse,
  LikeIndexResponse,
  LikeListItem,
} from "../features/like/types";
import ProductListingGrid, {
  type ProductListingCardViewModel,
} from "../features/shared/presentation/components/ProductListingGrid";
import { getApiBaseUrl } from "../lib/apiBaseUrl";

import "../styles/lists-page.css";

function formatItemPrice(item: LikeListItem): string {
  const prices = Array.isArray(item.prices) ? item.prices : [];
  const first = prices[0];
  const amount = first?.amount ?? first?.price ?? item.price;
  const currency =
    typeof first?.currency === "string" && first.currency.trim() !== ""
      ? first.currency
      : "JPY";

  return formatPrice(amount, { currency });
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

    const data = (await response.json()) as LikeCatalogResponse;
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
          typeof data.page === "number" && data.page > 0
            ? data.page
            : page,
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

  const listingItems: ProductListingCardViewModel[] = items.map((item) => ({
    id: getNavigateId(item),
    title: getItemTitle(item),
    imageUrl: getItemImage(item),
    brandName: item.brandName,
    priceLabel: formatItemPrice(item),
  }));

  const canGoPrev = page > 1 && !isLoading;
  const canGoNext = page < totalPages && !isLoading;

  function handleCartButtonClick() {
    navigate("/cart");
  }

  function handleOpenItem(listId: string) {
    const normalizedListId = listId.trim();

    if (!normalizedListId) {
      return;
    }

    navigate(`/favorites/${encodeURIComponent(normalizedListId)}`);
  }

  function handlePrevPage() {
    setPage((current) => Math.max(DEFAULT_PAGE, current - 1));
  }

  function handleNextPage() {
    setPage((current) => Math.min(totalPages, current + 1));
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
            emptyText="お気に入りの商品はありません。"
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