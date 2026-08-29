// frontend/amol/src/features/market/presentation/hooks/useMarketDetailPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import type { MediaGalleryItem } from "../../../../components/ui/MediaGallery";
import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import { fetchMarketProductBlueprintReviews } from "../../infrastructure/marketReviewApi";
import { fetchMarketResaleById } from "../../infrastructure/marketResaleApi";
import { fetchMarketResaleConditionImages } from "../../infrastructure/marketResaleImageApi";

import {
  createResaleConditionGalleryItems,
} from "../../../shared/presentation/utils/resaleConditionMedia";
import {
  createResaleModelDisplay,
  type ResaleModelDisplay,
} from "../../../shared/presentation/utils/resaleModelDisplay";
import type { MarketResaleListing } from "../../../shared/types/marketResale";
import type { ResaleConditionImage } from "../../../shared/types/resale";
import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

const DEFAULT_REVIEW_PAGE = 1;
const DEFAULT_REVIEW_PER_PAGE = 20;

export type AddResaleProductToCart = (args: {
  resaleId: string;
  productId: string;
}) => Promise<void>;

export type UseMarketDetailPageParams = {
  resaleId?: string;
  addResaleProductToCart: AddResaleProductToCart;
};

export type UseMarketDetailPageResult = {
  item: MarketResaleListing | null;
  reviews: ProductBlueprintReviewPage | null;

  loading: boolean;
  loadingReviews: boolean;
  addingToCart: boolean;

  error: string;
  reviewsError: string;
  cartMessage: string;
  cartErrorMessage: string;

  title: string;
  priceLabel: string;
  model: ResaleModelDisplay;

  tokenName: string;
  tokenIcon: string;

  sellerAvatarId: string;
  avatarName: string;
  avatarIcon: string;

  galleryItems: MediaGalleryItem[];
  safeActiveMediaIndex: number;

  canAddToCart: boolean;

  handlePrevMedia: () => void;
  handleNextMedia: () => void;
  handleSelectMedia: (index: number) => void;
  handleAddToCart: () => Promise<boolean>;
};

function getErrorMessage(error: unknown, fallbackMessage: string): string {
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }

  return fallbackMessage;
}

export function useMarketDetailPage({
  resaleId,
  addResaleProductToCart,
}: UseMarketDetailPageParams): UseMarketDetailPageResult {
  const normalizedResaleId = resaleId?.trim() ?? "";

  const [item, setItem] = useState<MarketResaleListing | null>(null);
  const [images, setImages] = useState<ResaleConditionImage[]>([]);
  const [reviews, setReviews] = useState<ProductBlueprintReviewPage | null>(null);

  const [activeMediaIndex, setActiveMediaIndex] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingReviews, setLoadingReviews] = useState(false);
  const [addingToCart, setAddingToCart] = useState(false);

  const [error, setError] = useState("");
  const [reviewsError, setReviewsError] = useState("");
  const [cartMessage, setCartMessage] = useState("");
  const [cartErrorMessage, setCartErrorMessage] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setLoadingReviews(false);
      setItem(null);
      setImages([]);
      setReviews(null);
      setError("");
      setReviewsError("");
      setCartMessage("");
      setCartErrorMessage("");
      setActiveMediaIndex(0);

      if (!normalizedResaleId) {
        setError("出品情報が見つかりません。");
        setLoading(false);
        return;
      }

      try {
        const [nextItem, nextImages] = await Promise.all([
          fetchMarketResaleById(normalizedResaleId),
          fetchMarketResaleConditionImages(normalizedResaleId),
        ]);

        if (cancelled) {
          return;
        }

        setItem(nextItem);
        setImages(nextImages);

        const productBlueprintId = textOrEmpty(nextItem.productBlueprintId);

        if (!productBlueprintId) {
          return;
        }

        setLoadingReviews(true);

        try {
          const nextReviews = await fetchMarketProductBlueprintReviews({
            productBlueprintId,
            page: DEFAULT_REVIEW_PAGE,
            perPage: DEFAULT_REVIEW_PER_PAGE,
          });

          if (!cancelled) {
            setReviews(nextReviews);
          }
        } catch (reviewError) {
          if (!cancelled) {
            setReviews(null);
            setReviewsError(
              getErrorMessage(reviewError, "レビューの取得に失敗しました。"),
            );
          }
        } finally {
          if (!cancelled) {
            setLoadingReviews(false);
          }
        }
      } catch (loadError) {
        if (!cancelled) {
          setItem(null);
          setImages([]);
          setReviews(null);
          setError(
            getErrorMessage(loadError, "出品情報の取得に失敗しました。"),
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [normalizedResaleId]);

  const title = item?.productName || item?.tokenName || "マーケット詳細";
  const priceLabel = item
    ? `${item.price.toLocaleString("ja-JP")}円`
    : "価格未設定";

  const model = useMemo(
    () => createResaleModelDisplay(item),
    [item],
  );

  const tokenName = textOrEmpty(item?.tokenName);
  const tokenIcon = textOrEmpty(item?.tokenIcon);
  const sellerAvatarId = textOrEmpty(item?.avatarId);
  const avatarName = textOrEmpty(item?.avatarName);
  const avatarIcon = textOrEmpty(item?.avatarIcon);

  const galleryItems = useMemo<MediaGalleryItem[]>(
    () =>
      createResaleConditionGalleryItems(images, {
        fallback: item
          ? {
              id: item.id,
              url: item.imageUrl,
              fileName: item.productName || item.tokenName || "出品画像",
            }
          : null,
      }),
    [images, item],
  );

  const safeActiveMediaIndex =
    activeMediaIndex >= 0 && activeMediaIndex < galleryItems.length
      ? activeMediaIndex
      : 0;

  const canAddToCart = Boolean(
    item?.id &&
      item.productId &&
      !loading &&
      !error &&
      !addingToCart,
  );

  const handlePrevMedia = useCallback(() => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveMediaIndex((current) =>
      current <= 0 ? galleryItems.length - 1 : current - 1,
    );
  }, [galleryItems.length]);

  const handleNextMedia = useCallback(() => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveMediaIndex((current) =>
      current >= galleryItems.length - 1 ? 0 : current + 1,
    );
  }, [galleryItems.length]);

  const handleSelectMedia = useCallback(
    (index: number) => {
      if (index < 0 || index >= galleryItems.length) {
        return;
      }

      setActiveMediaIndex(index);
    },
    [galleryItems.length],
  );

  const handleAddToCart = useCallback(async (): Promise<boolean> => {
    const targetResaleId = item?.id?.trim() ?? "";
    const targetProductId = item?.productId?.trim() ?? "";

    if (!targetResaleId || !targetProductId) {
      setCartMessage("");
      setCartErrorMessage("出品情報が不足しています。");
      return false;
    }

    setAddingToCart(true);
    setCartMessage("");
    setCartErrorMessage("");

    try {
      await addResaleProductToCart({
        resaleId: targetResaleId,
        productId: targetProductId,
      });

      setCartMessage("カートに追加しました。");
      return true;
    } catch (cartError) {
      setCartErrorMessage(
        getErrorMessage(cartError, "カートへの追加に失敗しました。"),
      );
      return false;
    } finally {
      setAddingToCart(false);
    }
  }, [addResaleProductToCart, item?.id, item?.productId]);

  return {
    item,
    reviews,
    loading,
    loadingReviews,
    addingToCart,
    error,
    reviewsError,
    cartMessage,
    cartErrorMessage,
    title,
    priceLabel,
    model,
    tokenName,
    tokenIcon,
    sellerAvatarId,
    avatarName,
    avatarIcon,
    galleryItems,
    safeActiveMediaIndex,
    canAddToCart,
    handlePrevMedia,
    handleNextMedia,
    handleSelectMedia,
    handleAddToCart,
  };
}