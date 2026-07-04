// frontend/amol/src/pages/ResaleDetailPage.tsx
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import MediaIcon from "../components/ui/MediaIcon";
import SectionHeader from "../components/ui/SectionHeader";
import Textbox from "../components/ui/Textbox";
import {
  listMyResaleConditionImages,
  listMyResaleListings,
} from "../features/resale/api/resaleApi";
import type {
  ResaleConditionImage,
  ResaleListing,
} from "../features/resale/api/resaleApi";

import "../styles/page-layout.css";
import "../styles/resale-page.css";

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function formatPrice(value: number | undefined): string {
  const price = Number(value ?? 0);

  if (!Number.isFinite(price) || price <= 0) {
    return "-";
  }

  return `${price.toLocaleString("ja-JP")}円`;
}

function formatDateTime(value: string | undefined | null): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function sortImages(images: ResaleConditionImage[]): ResaleConditionImage[] {
  return [...images].sort((a, b) => {
    const aOrder = Number(a.displayOrder ?? 0);
    const bOrder = Number(b.displayOrder ?? 0);

    if (aOrder !== bOrder) {
      return aOrder - bOrder;
    }

    return String(a.id || "").localeCompare(String(b.id || ""), "ja");
  });
}

export default function ResaleDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const imageCarouselRef = useRef<HTMLDivElement>(null);

  const normalizedResaleId = normalizeText(resaleId);

  const [item, setItem] = useState<ResaleListing | null>(null);
  const [images, setImages] = useState<ResaleConditionImage[]>([]);
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const [loading, setLoading] = useState<boolean>(true);
  const [errorMessage, setErrorMessage] = useState("");

  const sortedImages = useMemo(() => sortImages(images), [images]);
  const hasImages = sortedImages.length > 0;

  const productName = normalizeText(item?.productName);
  const tokenName = normalizeText(item?.tokenName);
  const brandName = normalizeText(item?.brandName);
  const condition = normalizeText(item?.condition);
  const description = normalizeText(item?.description);
  const status = normalizeText(item?.status);

  const title = productName || tokenName || "出品詳細";

  const priceLabel = formatPrice(item?.price);
  const createdAtLabel = formatDateTime(item?.createdAt);
  const updatedAtLabel = formatDateTime(item?.updatedAt);

  const primaryImageUrl = sortedImages[0]?.url || "";

  const loadDetail = useCallback(async () => {
    if (!normalizedResaleId) {
      setErrorMessage("出品情報が見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setErrorMessage("");

    try {
      const result = await listMyResaleListings({
        page: 1,
        perPage: 100,
      });

      const nextItem =
        result.items?.find(
          (listing) => normalizeText(listing.id) === normalizedResaleId,
        ) ?? null;

      if (!nextItem) {
        setItem(null);
        setImages([]);
        setErrorMessage("出品情報が見つかりません。");
        return;
      }

      const nextImages = await listMyResaleConditionImages(normalizedResaleId);

      setItem(nextItem);
      setImages(nextImages);
      setCurrentImageIndex(0);
    } catch (error) {
      setItem(null);
      setImages([]);
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "出品情報の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [normalizedResaleId]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const handleImageCarouselScroll = () => {
    const carousel = imageCarouselRef.current;

    if (!carousel) {
      return;
    }

    const width = carousel.clientWidth;

    if (width <= 0) {
      return;
    }

    setCurrentImageIndex(Math.round(carousel.scrollLeft / width));
  };

  const handleMoveToImageSlide = (index: number) => {
    const carousel = imageCarouselRef.current;

    if (!carousel) {
      setCurrentImageIndex(index);
      return;
    }

    carousel.scrollTo({
      left: carousel.clientWidth * index,
      behavior: "smooth",
    });

    setCurrentImageIndex(index);
  };

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() => navigate(-1)}
      mode="mypage"
    >
      <section className="page-section">
        {loading ? (
          <div className="page-card">
            <p className="page-card__text">読み込み中です...</p>
          </div>
        ) : null}

        {!loading && errorMessage ? (
          <div className="page-card">
            <SectionHeader title="出品情報を表示できません" titleAs="h2">
              <p className="page-card__text">{errorMessage}</p>
            </SectionHeader>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={() => void loadDetail()}
              >
                再読み込み
              </button>

              <button
                type="button"
                className="page-button page-button--primary"
                onClick={() => navigate("/wallet")}
              >
                ウォレットへ戻る
              </button>
            </div>
          </div>
        ) : null}

        {!loading && !errorMessage && item ? (
          <div className="page-stack">
            <section className="page-card">
              <SectionHeader title="出品対象" titleAs="h2" />

              <div className="resale-token-summary">
                <MediaIcon
                  src={primaryImageUrl}
                  alt={productName || tokenName || "出品画像"}
                  fallback="◎"
                  size="lg"
                  shape="rounded"
                  className="resale-token-summary__icon"
                />

                <div className="resale-token-summary__body">
                  <p className="resale-token-summary__token-name">
                    {tokenName || "-"}
                  </p>

                  <p className="resale-token-summary__brand-name">
                    {brandName || "-"}
                  </p>

                  {productName ? (
                    <p className="resale-token-summary__product-name">
                      {productName}
                    </p>
                  ) : null}
                </div>
              </div>
            </section>

            <section className="page-card">
              <SectionHeader title="商品状態の写真" titleAs="h2" />

              {hasImages ? (
                <div className="resale-condition-media">
                  <div
                    ref={imageCarouselRef}
                    className="resale-condition-media__carousel"
                    onScroll={handleImageCarouselScroll}
                  >
                    {sortedImages.map((image) => (
                      <div
                        key={image.id}
                        className="resale-condition-media__slide"
                      >
                        <img
                          src={image.url}
                          alt={image.fileName || "商品状態の写真"}
                          className="resale-condition-media__image"
                        />
                      </div>
                    ))}
                  </div>

                  {sortedImages.length > 1 ? (
                    <div
                      className="resale-condition-media__dots"
                      aria-label="商品状態写真のページ送り"
                    >
                      {sortedImages.map((image, index) => (
                        <button
                          key={image.id}
                          type="button"
                          className={
                            index === currentImageIndex
                              ? "resale-condition-media__dot resale-condition-media__dot--active"
                              : "resale-condition-media__dot"
                          }
                          aria-label={`${index + 1}枚目の写真を表示`}
                          onClick={() => handleMoveToImageSlide(index)}
                        />
                      ))}
                    </div>
                  ) : null}
                </div>
              ) : (
                <p className="page-card__text">商品状態の写真はありません。</p>
              )}
            </section>

            <section className="page-card">
              <SectionHeader title="販売情報" titleAs="h2" />

              <div className="page-form">
                <Input
                  label="販売価格"
                  type="text"
                  value={priceLabel}
                  readOnly
                />

                <Input
                  label="商品の状態"
                  type="text"
                  value={condition || "-"}
                  readOnly
                />

                <Input
                  label="出品ステータス"
                  type="text"
                  value={status || "-"}
                  readOnly
                />

                <Input
                  label="出品日時"
                  type="text"
                  value={createdAtLabel}
                  readOnly
                />

                <Input
                  label="更新日時"
                  type="text"
                  value={updatedAtLabel}
                  readOnly
                />

                <Textbox
                  label="説明文"
                  value={description || "説明文はありません。"}
                  rows={6}
                  readOnly
                />
              </div>
            </section>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}