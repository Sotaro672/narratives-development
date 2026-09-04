// frontend/amol/src/pages/LikesPage.tsx

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import ListPagination from "../components/ui/Pagination";
import { formatPrice } from "../components/utils/price";

import { fetchCatalogDetail } from "../features/catalog/infrastructure/catalogRepository";
import {
  DEFAULT_PAGE,
  DEFAULT_PER_PAGE,
} from "../features/like/constants";
import { fetchLikes } from "../features/like/infrastructure/likeApi";
import { fetchMarketResaleById } from "../features/market/infrastructure/marketResaleApi";
import ProductListingGrid, {
  type ProductListingCardViewModel,
} from "../features/shared/presentation/components/ProductListingGrid";
import type {
  LikeEntity,
  LikeTargetType,
} from "../features/shared/types/like";

import { getApiBaseUrl } from "../lib/apiBaseUrl";

import "../styles/lists-page.css";

type LikeDisplayItem = {
  key: string;
  targetType: LikeTargetType;
  targetId: string;
  card: ProductListingCardViewModel;
};

function buildLikeCardKey(
  targetType: LikeTargetType,
  targetId: string,
): string {
  return `${targetType}:${targetId}`;
}

function buildFallbackLikeDisplayItem(
  like: LikeEntity,
): LikeDisplayItem {
  const key = buildLikeCardKey(
    like.targetType,
    like.targetId,
  );

  return {
    key,
    targetType: like.targetType,
    targetId: like.targetId,
    card: {
      id: key,
      title:
        like.targetType === "list"
          ? "商品情報を取得できませんでした。"
          : "出品情報を取得できませんでした。",
      metaLines: [
        like.targetType === "list"
          ? "通常販売"
          : "二次流通",
      ],
    },
  };
}

async function fetchListLikeDisplayItem(
  apiBaseUrl: string,
  like: LikeEntity,
): Promise<LikeDisplayItem> {
  const catalog = await fetchCatalogDetail({
    apiBaseUrl,
    listId: like.targetId,
  });

  const key = buildLikeCardKey(
    like.targetType,
    like.targetId,
  );

  const sortedImages = [...catalog.listImages].sort(
    (a, b) => a.displayOrder - b.displayOrder,
  );

  const imageUrl =
    sortedImages[0]?.url?.trim() ||
    catalog.list.image?.trim() ||
    "";

  const title =
    catalog.productBlueprint.productName?.trim() ||
    catalog.list.title?.trim() ||
    "商品名未設定";

  const firstPrice = catalog.list.prices[0]?.price;

  return {
    key,
    targetType: like.targetType,
    targetId: like.targetId,
    card: {
      id: key,
      title,
      imageUrl,
      imageAlt: title,
      brandName: catalog.productBlueprint.brandName,
      metaLines: ["通常販売"],
      priceLabel: formatPrice(firstPrice),
      reviewAverage:
        catalog.productReviewSummary?.averageRating,
      reviewCount:
        catalog.productReviewSummary?.totalCount,
    },
  };
}

async function fetchResaleLikeDisplayItem(
  like: LikeEntity,
): Promise<LikeDisplayItem> {
  const resale = await fetchMarketResaleById(
    like.targetId,
  );

  const key = buildLikeCardKey(
    like.targetType,
    like.targetId,
  );

  const title =
    resale.productName?.trim() ||
    resale.tokenName?.trim() ||
    "商品名未設定";

  return {
    key,
    targetType: like.targetType,
    targetId: like.targetId,
    card: {
      id: key,
      title,
      imageUrl: resale.imageUrl,
      imageAlt: title,
      brandName: resale.brandName,
      metaLines: [
        "二次流通",
        resale.condition,
      ],
      priceLabel: formatPrice(resale.price),
    },
  };
}

async function fetchLikeDisplayItem(
  apiBaseUrl: string,
  like: LikeEntity,
): Promise<LikeDisplayItem> {
  try {
    if (like.targetType === "list") {
      return await fetchListLikeDisplayItem(
        apiBaseUrl,
        like,
      );
    }

    return await fetchResaleLikeDisplayItem(
      like,
    );
  } catch {
    return buildFallbackLikeDisplayItem(like);
  }
}

export default function LikesPage() {
  const navigate = useNavigate();

  const [items, setItems] =
    useState<LikeDisplayItem[]>([]);
  const [page, setPage] =
    useState(DEFAULT_PAGE);
  const [perPage] =
    useState(DEFAULT_PER_PAGE);
  const [totalPages, setTotalPages] =
    useState(1);
  const [isLoading, setIsLoading] =
    useState(true);

  const apiBaseUrl = useMemo(
    () => getApiBaseUrl(),
    [],
  );

  useEffect(() => {
    let cancelled = false;

    async function loadLikes() {
      setIsLoading(true);

      try {
        if (!apiBaseUrl) {
          throw new Error(
            "API Base URLが未設定です。",
          );
        }

        const result = await fetchLikes({
          page,
          perPage,
        });

        const displayItems =
          await Promise.all(
            result.items.map(
              (like) =>
                fetchLikeDisplayItem(
                  apiBaseUrl,
                  like,
                ),
            ),
          );

        if (cancelled) {
          return;
        }

        setItems(displayItems);

        setTotalPages(
          Number.isFinite(result.totalPages) &&
            result.totalPages > 0
            ? result.totalPages
            : 1,
        );

        if (
          Number.isFinite(result.page) &&
          result.page > 0
        ) {
          setPage(result.page);
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

    void loadLikes();

    return () => {
      cancelled = true;
    };
  }, [
    apiBaseUrl,
    page,
    perPage,
  ]);

  const listingItems:
    ProductListingCardViewModel[] =
      items.map((item) => item.card);

  const canGoPrev =
    page > DEFAULT_PAGE &&
    !isLoading;

  const canGoNext =
    page < totalPages &&
    !isLoading;

  function handleCartButtonClick() {
    navigate("/cart");
  }

  function handleOpenItem(
    cardId: string,
  ) {
    const item = items.find(
      (candidate) =>
        candidate.key === cardId,
    );

    if (!item) {
      return;
    }

    const targetId =
      item.targetId.trim();

    if (!targetId) {
      return;
    }

    if (item.targetType === "list") {
      navigate(
        `/lists/${encodeURIComponent(targetId)}`,
      );
      return;
    }

    navigate(
      `/market/${encodeURIComponent(targetId)}`,
    );
  }

  function handlePrevPage() {
    setPage(
      (current) =>
        Math.max(
          DEFAULT_PAGE,
          current - 1,
        ),
    );
  }

  function handleNextPage() {
    setPage(
      (current) =>
        Math.min(
          totalPages,
          current + 1,
        ),
    );
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
            emptyText="お気に入りはありません。"
          />
        ) : null}

        {!isLoading &&
        totalPages > 1 ? (
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