// frontend/amol/src/pages/MarketDetailPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import MediaGallery, {
  type MediaGalleryItem,
} from "../components/ui/MediaGallery";
import {
  fetchMarketResaleById,
  fetchMarketResaleConditionImages,
  type MarketResaleConditionImage,
  type MarketResaleListing,
} from "../features/market/marketApi";
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

function hasModelColor(color: MarketResaleModelColor | null | undefined): boolean {
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

  const idToken = await currentUser.getIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/cart/resales`, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
    body: JSON.stringify({
      resaleId: args.resaleId,
      productId: args.productId,
    }),
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "カートへの追加に失敗しました。");
  }
}

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const [item, setItem] = useState<MarketResaleListingWithModel | null>(null);
  const [images, setImages] = useState<MarketResaleConditionImage[]>([]);
  const [activeMediaIndex, setActiveMediaIndex] = useState<number>(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [addingToCart, setAddingToCart] = useState<boolean>(false);
  const [error, setError] = useState<string>("");
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
      setError("");
      setCartMessage("");
      setCartErrorMessage("");
      setActiveMediaIndex(0);

      try {
        const data = (await fetchMarketResaleById(
          resaleId,
        )) as MarketResaleListingWithModel;

        const nextImages = await fetchMarketResaleConditionImages(resaleId);

        if (!cancelled) {
          setItem(data);
          setImages(nextImages);
          setActiveMediaIndex(0);
        }
      } catch (err) {
        if (!cancelled) {
          setItem(null);
          setImages([]);
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

              {avatarName || avatarIcon ? (
                <div className="market-detail-page__seller">
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
                      {avatarName || "アバター名未設定"}
                    </span>
                  </div>
                </div>
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