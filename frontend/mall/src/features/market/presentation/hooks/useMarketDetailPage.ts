// frontend/amol/src/features/market/presentation/hooks/useMarketDetailPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import type { MediaGalleryItem } from "../../../../components/ui/MediaGallery";
import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import {
  addResaleLike,
  fetchResaleLikeStatus,
  removeResaleLike,
} from "../../../like/infrastructure/likeApi";
import { createResaleConditionGalleryItems } from "../../../shared/presentation/utils/resaleConditionMedia";
import {
  createProductModelDisplay,
  type ProductModelDisplay,
} from "../../../shared/presentation/utils/productModelDisplay";
import type { MarketResaleListing } from "../../../shared/types/marketResale";
import type { ResaleConditionImage } from "../../../shared/types/resale";
import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

import { fetchMarketProductBlueprintReviews } from "../../infrastructure/marketReviewApi";
import { fetchMarketResaleById } from "../../infrastructure/marketResaleApi";
import { fetchMarketResaleConditionImages } from "../../infrastructure/marketResaleImageApi";
import { fetchMarketResaleComments } from "../../infrastructure/marketResaleReviewApi";

const DEFAULT_REVIEW_PAGE = 1;
const DEFAULT_REVIEW_PER_PAGE = 20;
const COMMENT_COUNT_PAGE = 1;
const COMMENT_COUNT_PER_PAGE = 1;

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
  commentCount: number;
  isLiked: boolean;
  loading: boolean;
  loadingReviews: boolean;
  loadingLike: boolean;
  addingToCart: boolean;
  updatingLike: boolean;
  error: string;
  reviewsError: string;
  likeErrorMessage: string;
  cartMessage: string;
  cartErrorMessage: string;
  title: string;
  priceLabel: string;
  model: ProductModelDisplay;
  tokenName: string;
  tokenIcon: string;
  tokenDescription: string;
  sellerAvatarId: string;
  avatarName: string;
  avatarIcon: string;
  galleryItems: MediaGalleryItem[];
  safeActiveMediaIndex: number;
  canAddToCart: boolean;
  handlePrevMedia: () => void;
  handleNextMedia: () => void;
  handleSelectMedia: (index: number) => void;
  handleToggleLike: () => Promise<void>;
  handleAddToCart: () => Promise<boolean>;
};

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
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
  const [commentCount, setCommentCount] = useState(0);
  const [isLiked, setIsLiked] = useState(false);
  const [activeMediaIndex, setActiveMediaIndex] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingReviews, setLoadingReviews] = useState(false);
  const [loadingLike, setLoadingLike] = useState(false);
  const [addingToCart, setAddingToCart] = useState(false);
  const [updatingLike, setUpdatingLike] = useState(false);
  const [error, setError] = useState("");
  const [reviewsError, setReviewsError] = useState("");
  const [likeErrorMessage, setLikeErrorMessage] = useState("");
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
      setCommentCount(0);
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

      const commentPagePromise = fetchMarketResaleComments({
        resaleId: normalizedResaleId,
        page: COMMENT_COUNT_PAGE,
        perPage: COMMENT_COUNT_PER_PAGE,
      }).catch(() => null);

      try {
        const [nextItem, nextImages, commentPage] = await Promise.all([
          fetchMarketResaleById(normalizedResaleId),
          fetchMarketResaleConditionImages(normalizedResaleId),
          commentPagePromise,
        ]);

        if (cancelled) return;

        setItem(nextItem);
        setImages(nextImages);
        setCommentCount(commentPage?.totalCount ?? 0);

        const productBlueprintId = textOrEmpty(nextItem.productBlueprintId);

        if (!productBlueprintId) return;

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
              getErrorMessage(
                reviewError,
                "レビューの取得に失敗しました。",
              ),
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
          setCommentCount(0);
          setError(
            getErrorMessage(
              loadError,
              "出品情報の取得に失敗しました。",
            ),
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

  useEffect(() => {
    let cancelled = false;

    async function loadLikeStatus() {
      setIsLiked(false);
      setLikeErrorMessage("");
      setUpdatingLike(false);

      if (!normalizedResaleId) {
        setLoadingLike(false);
        return;
      }

      setLoadingLike(true);

      try {
        const status = await fetchResaleLikeStatus(normalizedResaleId);

        if (cancelled) return;

        setIsLiked(status.liked);
      } catch (likeError) {
        if (cancelled) return;

        setIsLiked(false);
        setLikeErrorMessage(
          getErrorMessage(
            likeError,
            "お気に入り状態の取得に失敗しました。",
          ),
        );
      } finally {
        if (!cancelled) {
          setLoadingLike(false);
        }
      }
    }

    void loadLikeStatus();

    return () => {
      cancelled = true;
    };
  }, [normalizedResaleId]);

  const title = item?.productName || item?.tokenName || "マーケット詳細";
  const priceLabel = item
    ? `${item.price.toLocaleString("ja-JP")}円`
    : "価格未設定";

  const model = useMemo(
    () => createProductModelDisplay(item),
    [item],
  );

  const tokenName = textOrEmpty(item?.tokenName);
  const tokenIcon = textOrEmpty(item?.tokenIcon);
  const tokenDescription = textOrEmpty(item?.tokenDescription);
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
    if (galleryItems.length <= 1) return;

    setActiveMediaIndex((current) =>
      current <= 0 ? galleryItems.length - 1 : current - 1,
    );
  }, [galleryItems.length]);

  const handleNextMedia = useCallback(() => {
    if (galleryItems.length <= 1) return;

    setActiveMediaIndex((current) =>
      current >= galleryItems.length - 1 ? 0 : current + 1,
    );
  }, [galleryItems.length]);

  const handleSelectMedia = useCallback(
    (index: number) => {
      if (index < 0 || index >= galleryItems.length) return;

      setActiveMediaIndex(index);
    },
    [galleryItems.length],
  );

  const handleToggleLike = useCallback(async (): Promise<void> => {
    if (
      !normalizedResaleId ||
      loadingLike ||
      updatingLike
    ) {
      return;
    }

    setUpdatingLike(true);
    setLikeErrorMessage("");

    try {
      const status = isLiked
        ? await removeResaleLike(normalizedResaleId)
        : await addResaleLike(normalizedResaleId);

      setIsLiked(status.liked);
    } catch (likeError) {
      setLikeErrorMessage(
        getErrorMessage(
          likeError,
          isLiked
            ? "お気に入りの解除に失敗しました。"
            : "お気に入りの登録に失敗しました。",
        ),
      );
    } finally {
      setUpdatingLike(false);
    }
  }, [
    isLiked,
    loadingLike,
    normalizedResaleId,
    updatingLike,
  ]);

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
        getErrorMessage(
          cartError,
          "カートへの追加に失敗しました。",
        ),
      );
      return false;
    } finally {
      setAddingToCart(false);
    }
  }, [
    addResaleProductToCart,
    item?.id,
    item?.productId,
  ]);

  return {
    item,
    reviews,
    commentCount,
    isLiked,
    loading,
    loadingReviews,
    loadingLike,
    addingToCart,
    updatingLike,
    error,
    reviewsError,
    likeErrorMessage,
    cartMessage,
    cartErrorMessage,
    title,
    priceLabel,
    model,
    tokenName,
    tokenIcon,
    tokenDescription,
    sellerAvatarId,
    avatarName,
    avatarIcon,
    galleryItems,
    safeActiveMediaIndex,
    canAddToCart,
    handlePrevMedia,
    handleNextMedia,
    handleSelectMedia,
    handleToggleLike,
    handleAddToCart,
  };
}