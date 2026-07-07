// frontend/amol/src/pages/MarketDetailPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import MediaGallery, {
  type MediaGalleryItem,
} from "../components/ui/MediaGallery";
import {
  fetchMarketProductBlueprintReviews,
  fetchMarketResaleById,
  fetchMarketResaleConditionImages,
  type MarketProductBlueprintReview,
  type MarketProductBlueprintReviewPage,
  type MarketResaleConditionImage,
  type MarketResaleListing,
} from "../features/market/marketApi";
import { fetchCurrentAvatarId } from "../features/catalog/infrastructure/avatarStateRepository";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { auth } from "../lib/firebase";
import { rgbToCssColor, toSafeColorRGB } from "../components/utils/color";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

type MarketResaleModelColor = {
  name?: string;
  rgb?: number;
};

type MarketResaleModelVolume = {
  amount?: number;
  value?: number;
  unit?: string;
};

type MarketResaleListingWithModel = MarketResaleListing & {
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: MarketResaleModelColor | null;
  measurements?: Record<string, number> | null;
  volume?: MarketResaleModelVolume | null;
};

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

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

function formatReviewDate(value: string | undefined): string {
  const text = normalizeText(value);

  if (!text) {
    return "";
  }

  const date = new Date(text);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

function getRatingStars(value: number | undefined): string {
  const rating = Math.max(0, Math.min(5, Math.trunc(Number(value ?? 0))));

  if (rating <= 0) {
    return "評価なし";
  }

  return "★".repeat(rating) + "☆".repeat(5 - rating);
}

function getModelColorName(
  color: MarketResaleModelColor | null | undefined,
): string {
  return normalizeText(color?.name);
}

function getModelColorCssValue(
  color: MarketResaleModelColor | null | undefined,
): string {
  if (!color) {
    return "";
  }

  return rgbToCssColor(toSafeColorRGB(color.rgb));
}

function hasModelColor(
  color: MarketResaleModelColor | null | undefined,
): boolean {
  if (!color) {
    return false;
  }

  const name = getModelColorName(color);
  const rgb = Number(color.rgb);

  return Boolean(name) || Number.isFinite(rgb);
}

function formatModelVolume(
  volume: MarketResaleModelVolume | null | undefined,
): string {
  if (!volume) {
    return "-";
  }

  const amount = Number(volume.amount ?? volume.value ?? 0);
  const unit = normalizeText(volume.unit);

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
    const label = normalizeText(key);
    const numericValue = Number(value);

    return label !== "" && Number.isFinite(numericValue);
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
  images: MarketResaleConditionImage[],
): MarketResaleConditionImage[] {
  return [...images].sort((a, b) => {
    const aOrder = Number(a.displayOrder ?? 0);
    const bOrder = Number(b.displayOrder ?? 0);

    if (aOrder !== bOrder) {
      return aOrder - bOrder;
    }

    return String(a.id || "").localeCompare(String(b.id || ""), "ja");
  });
}

function createGalleryItemFromImage(
  image: MarketResaleConditionImage,
): MediaGalleryItem {
  return {
    id: image.id,
    url: image.url,
    fileName: image.fileName || "出品画像",
    type: image.mimeType || image.type || getFileTypeFromUrl(image.url),
  };
}

function createFallbackGalleryItem(
  item: MarketResaleListingWithModel,
): MediaGalleryItem | null {
  const imageUrl = normalizeText(item.imageUrl);

  if (!imageUrl) {
    return null;
  }

  return {
    id: normalizeText(item.imageId) || normalizeText(item.id) || imageUrl,
    url: imageUrl,
    fileName: item.productName || item.tokenName || "出品画像",
    type: getFileTypeFromUrl(imageUrl),
  };
}

async function readResponseErrorMessage(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";

  if (contentType.includes("application/json")) {
    const data = (await response.json().catch(() => null)) as
      | { error?: unknown; message?: unknown }
      | null;

    if (typeof data?.error === "string" && data.error.trim() !== "") {
      return data.error;
    }

    if (typeof data?.message === "string" && data.message.trim() !== "") {
      return data.message;
    }
  }

  const text = await response.text().catch(() => "");

  if (text.trim() !== "") {
    return text;
  }

  return "リクエストに失敗しました。";
}

async function addResaleProductToCart(args: {
  resaleId: string;
  productId: string;
}): Promise<void> {
  const currentUser = auth.currentUser;

  if (!currentUser) {
    throw new Error("カートに追加するにはログインが必要です。");
  }

  const apiBaseUrl = getApiBaseUrl();

  if (!apiBaseUrl) {
    throw new Error("APIの接続先が設定されていません。");
  }

  const normalizedApiBaseUrl = apiBaseUrl.replace(/\/+$/, "");
  const idToken = await currentUser.getIdToken();
  const avatarId = await fetchCurrentAvatarId(normalizedApiBaseUrl);

  const response = await fetch(`${normalizedApiBaseUrl}/mall/me/cart/resales`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify({
      avatarId,
      resaleId: args.resaleId,
      productId: args.productId,
    }),
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートへの追加に失敗しました。");
  }
}

function ReviewAvatar({
  review,
}: {
  review: MarketProductBlueprintReview;
}) {
  const avatarName = normalizeText(review.avatarName);
  const avatarIcon = normalizeText(review.avatarIcon);
  const avatarId = normalizeText(review.avatarId);

  return (
    <div className="market-detail-page__review-author">
      {avatarIcon ? (
        <img
          src={avatarIcon}
          alt={avatarName || avatarId || "レビュー投稿者"}
          className="market-detail-page__review-author-icon"
        />
      ) : (
        <span
          className="market-detail-page__review-author-icon market-detail-page__review-author-icon--placeholder"
          aria-hidden="true"
        >
          ◎
        </span>
      )}

      <span className="market-detail-page__review-author-name">
        {avatarName || avatarId || "匿名"}
      </span>
    </div>
  );
}

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const [item, setItem] = useState<MarketResaleListingWithModel | null>(null);
  const [images, setImages] = useState<MarketResaleConditionImage[]>([]);
  const [reviews, setReviews] =
    useState<MarketProductBlueprintReviewPage | null>(null);
  const [activeMediaIndex, setActiveMediaIndex] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [loadingReviews, setLoadingReviews] = useState<boolean>(false);
  const [addingToCart, setAddingToCart] = useState<boolean>(false);
  const [error, setError] = useState<string>("");
  const [reviewsError, setReviewsError] = useState<string>("");
  const [cartMessage, setCartMessage] = useState<string>("");
  const [cartErrorMessage, setCartErrorMessage] = useState<string>("");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!resaleId) {
        setError("出品情報が見つかりません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setLoadingReviews(false);
      setError("");
      setReviewsError("");
      setCartMessage("");
      setCartErrorMessage("");
      setActiveMediaIndex(0);

      try {
        const data = (await fetchMarketResaleById(
          resaleId,
        )) as MarketResaleListingWithModel;

        const nextImages = await fetchMarketResaleConditionImages(resaleId);

        if (cancelled) {
          return;
        }

        setItem(data);
        setImages(nextImages);
        setActiveMediaIndex(0);

        const productBlueprintId = normalizeText(data.productBlueprintId);

        if (!productBlueprintId) {
          setReviews(null);
          return;
        }

        setLoadingReviews(true);

        try {
          const nextReviews = await fetchMarketProductBlueprintReviews({
            productBlueprintId,
            page: 1,
            perPage: 20,
          });

          if (!cancelled) {
            setReviews(nextReviews);
          }
        } catch (reviewErr) {
          if (!cancelled) {
            setReviews(null);
            setReviewsError(
              reviewErr instanceof Error
                ? reviewErr.message
                : "レビューの取得に失敗しました。",
            );
          }
        } finally {
          if (!cancelled) {
            setLoadingReviews(false);
          }
        }
      } catch (err) {
        if (!cancelled) {
          setItem(null);
          setImages([]);
          setReviews(null);
          setActiveMediaIndex(0);
          setError(
            err instanceof Error
              ? err.message
              : "出品情報の取得に失敗しました。",
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
  }, [resaleId]);

  const title = item?.productName || item?.tokenName || "マーケット詳細";

  const priceLabel =
    typeof item?.price === "number"
      ? `${item.price.toLocaleString("ja-JP")}円`
      : "価格未設定";

  const modelId = normalizeText(item?.modelId);
  const modelKind = normalizeText(item?.kind);
  const modelNumber = normalizeText(item?.modelNumber);
  const modelSize = normalizeText(item?.size);
  const tokenName = normalizeText(item?.tokenName);
  const tokenIcon = normalizeText(item?.tokenIcon);
  const sellerAvatarId = normalizeText(item?.avatarId);
  const avatarName = normalizeText(item?.avatarName);
  const avatarIcon = normalizeText(item?.avatarIcon);

  const galleryItems = useMemo<MediaGalleryItem[]>(() => {
    const fromImages = sortMarketResaleImages(images).map(
      createGalleryItemFromImage,
    );

    if (fromImages.length > 0) {
      return fromImages;
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

  const modelKindLabel = formatModelKind(modelKind);
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

  const canAddToCart = Boolean(
    item?.id && item?.productId && !loading && !error && !addingToCart,
  );

  function handlePrevMedia() {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveMediaIndex((current) =>
      current <= 0 ? galleryItems.length - 1 : current - 1,
    );
  }

  function handleNextMedia() {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveMediaIndex((current) =>
      current >= galleryItems.length - 1 ? 0 : current + 1,
    );
  }

  function handleSelectMedia(index: number) {
    if (index < 0 || index >= galleryItems.length) {
      return;
    }

    setActiveMediaIndex(index);
  }

  function handleOpenSellerAvatar() {
    if (!sellerAvatarId) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(sellerAvatarId)}`);
  }

  async function handleAddToCart() {
    const targetResaleId = item?.id?.trim();
    const targetProductId = item?.productId?.trim();

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
    } catch (err) {
      setCartErrorMessage(
        err instanceof Error ? err.message : "カートへの追加に失敗しました。",
      );
    } finally {
      setAddingToCart(false);
    }
  }

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() => navigate(-1)}
      hideAnnouncementButton
      hideSettingsButton
      hideHamburgerMenu
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() => navigate("/cart")}
      actionButtonLabel={addingToCart ? "追加中" : "カートに入れる"}
      onActionButtonClick={handleAddToCart}
      actionButtonDisabled={!canAddToCart}
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel: addingToCart ? "追加中" : "カートに入れる",
        disabled: !canAddToCart,
        onButtonClick: handleAddToCart,
      }}
    >
      <div className="page-layout market-detail-page">
        {loading ? (
          <div className="market-detail-page__state">
            <p>読み込み中です...</p>
          </div>
        ) : null}

        {!loading && error ? (
          <div className="market-detail-page__state market-detail-page__state--error">
            <p>{error}</p>
          </div>
        ) : null}

        {!loading && !error && item ? (
          <section className="market-detail-page__card">
            <div className="market-detail-page__image-wrap">
              <MediaGallery
                items={galleryItems}
                activeIndex={safeActiveMediaIndex}
                altFallback={item.productName || item.tokenName || "出品画像"}
                placeholderText="No Image"
                className="market-detail-page__media-gallery"
                onPrev={handlePrevMedia}
                onNext={handleNextMedia}
                onSelect={handleSelectMedia}
              />
            </div>

            <div className="market-detail-page__content">
              <p className="market-detail-page__brand">
                {item.brandName || "ブランド名未設定"}
              </p>

              <h1 className="market-detail-page__title">
                {item.productName || item.tokenName || "商品名未設定"}
              </h1>

              {avatarName || avatarIcon || sellerAvatarId ? (
                <button
                  type="button"
                  className="market-detail-page__seller market-detail-page__seller--button"
                  onClick={handleOpenSellerAvatar}
                  disabled={!sellerAvatarId}
                >
                  {avatarIcon ? (
                    <img
                      src={avatarIcon}
                      alt={avatarName || "出品者アイコン"}
                      className="market-detail-page__seller-icon"
                    />
                  ) : (
                    <span
                      className="market-detail-page__seller-icon market-detail-page__seller-icon--placeholder"
                      aria-hidden="true"
                    >
                      ◎
                    </span>
                  )}

                  <div className="market-detail-page__seller-body">
                    <span className="market-detail-page__seller-label">
                      出品者
                    </span>
                    <span className="market-detail-page__seller-name">
                      {avatarName || sellerAvatarId || "アバター名未設定"}
                    </span>
                  </div>

                  {sellerAvatarId ? (
                    <span
                      className="market-detail-page__seller-arrow"
                      aria-hidden="true"
                    >
                      ›
                    </span>
                  ) : null}
                </button>
              ) : null}

              {tokenName || tokenIcon ? (
                <div className="market-detail-page__token">
                  {tokenIcon ? (
                    <img
                      src={tokenIcon}
                      alt={tokenName || "トークンアイコン"}
                      className="market-detail-page__token-icon"
                    />
                  ) : null}

                  <div className="market-detail-page__token-body">
                    <span className="market-detail-page__token-label">
                      トークン
                    </span>
                    <span className="market-detail-page__token-name">
                      {tokenName || "トークン名未設定"}
                    </span>
                  </div>
                </div>
              ) : null}

              <p className="market-detail-page__price">{priceLabel}</p>

              <dl className="market-detail-page__meta">
                {item.condition ? (
                  <div className="market-detail-page__meta-row">
                    <dt>状態</dt>
                    <dd>{item.condition}</dd>
                  </div>
                ) : null}

                {hasModelInfo ? (
                  <>
                    {modelKind ? (
                      <div className="market-detail-page__meta-row">
                        <dt>種別</dt>
                        <dd>{modelKindLabel}</dd>
                      </div>
                    ) : null}

                    {modelNumber ? (
                      <div className="market-detail-page__meta-row">
                        <dt>モデル番号</dt>
                        <dd>{modelNumber}</dd>
                      </div>
                    ) : null}

                    {modelSize ? (
                      <div className="market-detail-page__meta-row">
                        <dt>サイズ</dt>
                        <dd>{modelSize}</dd>
                      </div>
                    ) : null}

                    {hasColorInfo ? (
                      <div className="market-detail-page__meta-row">
                        <dt>カラー</dt>
                        <dd>
                          <span className="market-detail-page__color-value">
                            {modelColorCssValue ? (
                              <span
                                className="market-detail-page__color-swatch"
                                style={{
                                  backgroundColor: modelColorCssValue,
                                }}
                                aria-hidden="true"
                              />
                            ) : null}

                            <span>
                              {modelColorName ||
                                modelColorCssValue ||
                                "カラー未設定"}
                            </span>
                          </span>
                        </dd>
                      </div>
                    ) : null}

                    {measurementsLabel !== "-" ? (
                      <div className="market-detail-page__meta-row">
                        <dt>採寸</dt>
                        <dd>{measurementsLabel}</dd>
                      </div>
                    ) : null}

                    {modelVolumeLabel !== "-" ? (
                      <div className="market-detail-page__meta-row">
                        <dt>容量</dt>
                        <dd>{modelVolumeLabel}</dd>
                      </div>
                    ) : null}
                  </>
                ) : null}
              </dl>

              {item.description ? (
                <div className="market-detail-page__description">
                  <h2>商品説明</h2>
                  <p>{item.description}</p>
                </div>
              ) : null}

              <section className="market-detail-page__reviews">
                <div className="market-detail-page__reviews-header">
                  <h2>レビュー</h2>

                  {loadingReviews ? (
                    <span className="market-detail-page__reviews-status">
                      読み込み中...
                    </span>
                  ) : null}
                </div>

                {reviewsError ? (
                  <p className="market-detail-page__reviews-error" role="alert">
                    {reviewsError}
                  </p>
                ) : null}

                {!loadingReviews && !reviewsError && reviews?.items.length ? (
                  <div className="market-detail-page__review-list">
                    {reviews.items.map((review) => {
                      const title = normalizeText(review.title);
                      const body = normalizeText(review.body);
                      const reviewedAt = formatReviewDate(review.reviewedAt);

                      return (
                        <article
                          className="market-detail-page__review"
                          key={review.id}
                        >
                          <div className="market-detail-page__review-top">
                            <ReviewAvatar review={review} />

                            <span className="market-detail-page__review-rating">
                              {getRatingStars(review.rating)}
                            </span>
                          </div>

                          {title ? (
                            <h3 className="market-detail-page__review-title">
                              {title}
                            </h3>
                          ) : null}

                          {body ? (
                            <p className="market-detail-page__review-body">
                              {body}
                            </p>
                          ) : null}

                          {reviewedAt ? (
                            <time className="market-detail-page__review-date">
                              {reviewedAt}
                            </time>
                          ) : null}
                        </article>
                      );
                    })}
                  </div>
                ) : null}

                {!loadingReviews &&
                !reviewsError &&
                (!reviews || reviews.items.length === 0) ? (
                  <p className="market-detail-page__reviews-empty">
                    まだレビューはありません。
                  </p>
                ) : null}
              </section>

              {cartMessage ? (
                <p className="market-detail-page__cart-message">
                  {cartMessage}
                </p>
              ) : null}

              {cartErrorMessage ? (
                <p className="market-detail-page__cart-error" role="alert">
                  {cartErrorMessage}
                </p>
              ) : null}
            </div>
          </section>
        ) : null}
      </div>
    </Layout>
  );
}