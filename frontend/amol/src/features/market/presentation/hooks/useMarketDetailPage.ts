// frontend/amol/src/features/market/presentation/hooks/useMarketDetailPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import type { MediaGalleryItem } from "../../../../components/ui/MediaGallery";
import { rgbToCssColor, toSafeColorRGB } from "../../../../components/utils/color";
import { textOrEmpty } from "../../../../components/utils/textOrEmpty";

import { fetchMarketProductBlueprintReviews } from "../../infrastructure/marketReviewApi";
import { fetchMarketResaleById } from "../../infrastructure/marketResaleApi";
import { fetchMarketResaleConditionImages } from "../../infrastructure/marketResaleImageApi";

import type { MarketResaleListing } from "../../../shared/types/marketResale";
import type {
  ResaleColor,
  ResaleConditionImage,
  ResaleVolume,
} from "../../../shared/types/resale";
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
  images: ResaleConditionImage[];
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

  modelId: string;
  modelKind: string;
  modelKindLabel: string;
  modelNumber: string;
  modelSize: string;

  modelColorName: string;
  modelColorCssValue: string;
  hasColorInfo: boolean;

  modelVolumeLabel: string;
  measurementsLabel: string;
  hasModelInfo: boolean;

  tokenName: string;
  tokenIcon: string;

  sellerAvatarId: string;
  avatarName: string;
  avatarIcon: string;

  galleryItems: MediaGalleryItem[];
  activeMediaIndex: number;
  safeActiveMediaIndex: number;

  canAddToCart: boolean;

  handlePrevMedia: () => void;
  handleNextMedia: () => void;
  handleSelectMedia: (index: number) => void;
  handleAddToCart: () => Promise<void>;
};

function formatModelKind(value: string): string {
  switch (value) {
    case "apparel":
      return "アパレル";
    case "alcohol":
      return "酒類";
    default:
      return value || "-";
  }
}

function getModelColorName(color: ResaleColor | null | undefined): string {
  return textOrEmpty(color?.name);
}

function getModelColorCssValue(color: ResaleColor | null | undefined): string {
  if (!color) {
    return "";
  }

  return rgbToCssColor(toSafeColorRGB(color.rgb));
}

function hasModelColor(color: ResaleColor | null | undefined): boolean {
  if (!color) {
    return false;
  }

  return Boolean(getModelColorName(color)) || Number.isFinite(Number(color.rgb));
}

function formatModelVolume(volume: ResaleVolume | null | undefined): string {
  if (!volume) {
    return "-";
  }

  const amount = Number(volume.amount ?? 0);
  const unit = textOrEmpty(volume.unit);

  if (!Number.isFinite(amount) || amount <= 0) {
    return unit || "-";
  }

  return unit ? `${amount.toLocaleString("ja-JP")}${unit}` : `${amount}`;
}

function formatMeasurements(
  measurements: Record<string, number> | null | undefined,
): string {
  if (!measurements) {
    return "-";
  }

  const entries = Object.entries(measurements).filter(([key, value]) => {
    return textOrEmpty(key) !== "" && Number.isFinite(Number(value));
  });

  if (entries.length === 0) {
    return "-";
  }

  return entries
    .sort(([a], [b]) => a.localeCompare(b, "ja"))
    .map(([key, value]) => `${key}: ${Number(value).toLocaleString("ja-JP")}`)
    .join(" / ");
}

function getFileTypeFromUrl(url: string): string {
  const normalizedUrl = url.toLowerCase();

  if (
    normalizedUrl.includes(".mp4") ||
    normalizedUrl.includes(".mov") ||
    normalizedUrl.includes(".webm")
  ) {
    return "video/mp4";
  }

  return "image/*";
}

function sortMarketResaleImages(
  images: ResaleConditionImage[],
): ResaleConditionImage[] {
  return [...images].sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }

    return a.id.localeCompare(b.id, "ja");
  });
}

function createGalleryItemFromImage(
  image: ResaleConditionImage,
): MediaGalleryItem {
  return {
    id: image.id,
    url: image.url,
    fileName: "出品画像",
    type: getFileTypeFromUrl(image.url),
  };
}

function createFallbackGalleryItem(
  item: MarketResaleListing,
): MediaGalleryItem | null {
  const imageUrl = textOrEmpty(item.imageUrl);

  if (!imageUrl) {
    return null;
  }

  return {
    id: item.id,
    url: imageUrl,
    fileName: item.productName || item.tokenName || "出品画像",
    type: getFileTypeFromUrl(imageUrl),
  };
}

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

  const modelId = textOrEmpty(item?.modelId);
  const modelKind = textOrEmpty(item?.kind);
  const modelKindLabel = formatModelKind(modelKind);
  const modelNumber = textOrEmpty(item?.modelNumber);
  const modelSize = textOrEmpty(item?.size);

  const modelColorName = getModelColorName(item?.color);
  const modelColorCssValue = getModelColorCssValue(item?.color);
  const hasColorInfo = hasModelColor(item?.color);

  const modelVolumeLabel = formatModelVolume(item?.volume);

  const measurementsLabel = useMemo(
    () => formatMeasurements(item?.measurements),
    [item?.measurements],
  );

  const hasModelInfo =
    Boolean(modelId) ||
    Boolean(modelKind) ||
    Boolean(modelNumber) ||
    Boolean(modelSize) ||
    hasColorInfo ||
    modelVolumeLabel !== "-" ||
    measurementsLabel !== "-";

  const tokenName = textOrEmpty(item?.tokenName);
  const tokenIcon = textOrEmpty(item?.tokenIcon);
  const sellerAvatarId = textOrEmpty(item?.avatarId);
  const avatarName = textOrEmpty(item?.avatarName);
  const avatarIcon = textOrEmpty(item?.avatarIcon);

  const galleryItems = useMemo<MediaGalleryItem[]>(() => {
    const imageItems = sortMarketResaleImages(images).map(
      createGalleryItemFromImage,
    );

    if (imageItems.length > 0) {
      return imageItems;
    }

    if (!item) {
      return [];
    }

    const fallbackItem = createFallbackGalleryItem(item);
    return fallbackItem ? [fallbackItem] : [];
  }, [images, item]);

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

  const handleAddToCart = useCallback(async () => {
    const targetResaleId = item?.id?.trim() ?? "";
    const targetProductId = item?.productId?.trim() ?? "";

    if (!targetResaleId || !targetProductId) {
      setCartMessage("");
      setCartErrorMessage("出品情報が不足しています。");
      return;
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
    } catch (cartError) {
      setCartErrorMessage(
        getErrorMessage(cartError, "カートへの追加に失敗しました。"),
      );
    } finally {
      setAddingToCart(false);
    }
  }, [addResaleProductToCart, item?.id, item?.productId]);

  return {
    item,
    images,
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

    modelId,
    modelKind,
    modelKindLabel,
    modelNumber,
    modelSize,

    modelColorName,
    modelColorCssValue,
    hasColorInfo,

    modelVolumeLabel,
    measurementsLabel,
    hasModelInfo,

    tokenName,
    tokenIcon,

    sellerAvatarId,
    avatarName,
    avatarIcon,

    galleryItems,
    activeMediaIndex,
    safeActiveMediaIndex,

    canAddToCart,

    handlePrevMedia,
    handleNextMedia,
    handleSelectMedia,
    handleAddToCart,
  };
}