//frontend\amol\src\pages\ResalePage.tsx
import { useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import MediaIcon from "../components/ui/MediaIcon";
import MediaUploader, {
  type MediaUploaderItem,
} from "../components/ui/MediaUploader";
import SectionHeader from "../components/ui/SectionHeader";
import Textbox from "../components/ui/Textbox";
import { createResaleListing } from "../features/resale/api/resaleApi";

import "../styles/page-layout.css";
import "../styles/resale-page.css";

type ResalePageLocationState = {
  mintAddress?: string;
  productId?: string;
  brandId?: string;
  brandName?: string;
  productName?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  tokenName?: string;
  tokenIconUrl?: string;
  currentAvatarId?: string;
};

type ConditionMediaItem = MediaUploaderItem & {
  file: File;
};

function getLocationState(value: unknown): ResalePageLocationState {
  if (!value || typeof value !== "object") {
    return {};
  }

  return value as ResalePageLocationState;
}

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function formatPrice(value: string): string {
  const digits = value.replace(/[^\d]/g, "");

  if (!digits) {
    return "";
  }

  return Number(digits).toLocaleString("ja-JP");
}

function createConditionMediaItem(file: File): ConditionMediaItem {
  return {
    id: `${file.name}-${file.size}-${file.lastModified}-${crypto.randomUUID()}`,
    type: "image",
    previewUrl: URL.createObjectURL(file),
    title: file.name,
    fileName: file.name,
    file,
  };
}

export default function ResalePage() {
  const navigate = useNavigate();
  const location = useLocation();

  const conditionMediaInputRef = useRef<HTMLInputElement>(null);
  const conditionMediaCarouselRef = useRef<HTMLDivElement>(null);

  const locationState = useMemo(
    () => getLocationState(location.state),
    [location.state],
  );

  const mintAddress = normalizeText(locationState.mintAddress);
  const productId = normalizeText(locationState.productId);
  const brandId = normalizeText(locationState.brandId);
  const brandName = normalizeText(locationState.brandName);
  const productName = normalizeText(locationState.productName);
  const productBlueprintId = normalizeText(locationState.productBlueprintId);
  const tokenBlueprintId = normalizeText(locationState.tokenBlueprintId);
  const tokenName = normalizeText(locationState.tokenName);
  const tokenIconUrl = normalizeText(locationState.tokenIconUrl);
  const currentAvatarId = normalizeText(locationState.currentAvatarId);

  const [price, setPrice] = useState("");
  const [description, setDescription] = useState("");
  const [condition, setCondition] = useState("未使用に近い");
  const [conditionMediaItems, setConditionMediaItems] = useState<
    ConditionMediaItem[]
  >([]);
  const [conditionMediaCurrentIndex, setConditionMediaCurrentIndex] =
    useState(0);
  const [errorMessage, setErrorMessage] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const formattedPrice = useMemo(() => formatPrice(price), [price]);

  const hasRequiredListingTarget =
    Boolean(productId) && Boolean(tokenBlueprintId);

  const hasConditionMedia = conditionMediaItems.length > 0;

  const canSubmit =
    hasRequiredListingTarget &&
    Boolean(currentAvatarId) &&
    Boolean(price.replace(/[^\d]/g, "")) &&
    hasConditionMedia;

  const handleChangePrice = (value: string) => {
    setPrice(value.replace(/[^\d]/g, ""));
    setErrorMessage("");
  };

  const handleConditionMediaSelected = (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const files = Array.from(event.currentTarget.files ?? []).filter((file) =>
      file.type.startsWith("image/"),
    );

    if (files.length === 0) {
      event.currentTarget.value = "";
      return;
    }

    const nextItems = files.map(createConditionMediaItem);

    setConditionMediaItems((current) => [...current, ...nextItems]);
    setErrorMessage("");
    event.currentTarget.value = "";
  };

  const handleRemoveConditionMedia = (id: string) => {
    setConditionMediaItems((current) => {
      const removingItem = current.find((item) => item.id === id);

      if (removingItem?.previewUrl) {
        URL.revokeObjectURL(removingItem.previewUrl);
      }

      const nextItems = current.filter((item) => item.id !== id);

      setConditionMediaCurrentIndex((currentIndex) => {
        if (nextItems.length === 0) {
          return 0;
        }

        return Math.min(currentIndex, nextItems.length - 1);
      });

      return nextItems;
    });
  };

  const handleConditionMediaCarouselScroll = () => {
    const carousel = conditionMediaCarouselRef.current;

    if (!carousel) {
      return;
    }

    const width = carousel.clientWidth;

    if (width <= 0) {
      return;
    }

    setConditionMediaCurrentIndex(Math.round(carousel.scrollLeft / width));
  };

  const handleMoveToConditionMediaSlide = (index: number) => {
    const carousel = conditionMediaCarouselRef.current;

    if (!carousel) {
      setConditionMediaCurrentIndex(index);
      return;
    }

    carousel.scrollTo({
      left: carousel.clientWidth * index,
      behavior: "smooth",
    });

    setConditionMediaCurrentIndex(index);
  };

  const handleSubmit = async () => {
    if (isSubmitting) {
      return;
    }

    if (!canSubmit) {
      setErrorMessage("販売価格と商品状態の写真を入力してください。");
      return;
    }

    setIsSubmitting(true);
    setErrorMessage("");

    try {
      const created = await createResaleListing({
        mintAddress,
        tokenBlueprintId,
        productId,
        brandId,
        productBlueprintId,
        avatarId: currentAvatarId,
        price: Number(price),
        condition,
        description,
        conditionImages: conditionMediaItems.map((item) => item.file),
      });

      navigate("/wallet", {
        replace: true,
        state: {
          resaleCreated: true,
          resaleId: created?.id,
        },
      });
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "出品に失敗しました。時間をおいてもう一度お試しください。",
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  useEffect(() => {
    return () => {
      conditionMediaItems.forEach((item) => {
        if (item.previewUrl) {
          URL.revokeObjectURL(item.previewUrl);
        }
      });
    };
  }, [conditionMediaItems]);

  return (
    <Layout
      title="出品"
      showBackButton
      mode="mypage"
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel: isSubmitting ? "出品中..." : "出品する",
        disabled: !canSubmit || isSubmitting,
        onButtonClick: handleSubmit,
      }}
    >
      <section className="page-section">
        {!hasRequiredListingTarget ? (
          <div className="page-card">
            <SectionHeader title="出品情報が見つかりません" titleAs="h2">
              <p className="page-card__text">
                ウォレットまたはトークン詳細から、もう一度出品ボタンを押してください。
              </p>
            </SectionHeader>

            <button
              type="button"
              className="page-button page-button--primary"
              onClick={() => navigate("/wallet")}
            >
              ウォレットへ戻る
            </button>
          </div>
        ) : (
          <div className="page-stack">
            <section className="page-card">
              <SectionHeader title="出品対象" titleAs="h2" />

              <div className="resale-token-summary">
                <MediaIcon
                  src={tokenIconUrl}
                  alt={tokenName || "トークンアイコン"}
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

              {!currentAvatarId ? (
                <p className="page-error" role="alert">
                  出品者情報が取得できていません。再度ウォレットから開き直してください。
                </p>
              ) : null}
            </section>

            <section className="page-card">
              <SectionHeader title="販売情報" titleAs="h2" />

              <div className="page-form">
                <Input
                  label="販売価格"
                  type="text"
                  inputMode="numeric"
                  value={formattedPrice}
                  placeholder="例：12,000"
                  helperText="半角数字で入力してください。"
                  required
                  onChange={(event) =>
                    handleChangePrice(event.currentTarget.value)
                  }
                />

                <label className="page-form__field">
                  <span className="page-form__label">商品の状態</span>
                  <select
                    value={condition}
                    onChange={(event) => setCondition(event.currentTarget.value)}
                  >
                    <option value="新品・未使用">新品・未使用</option>
                    <option value="未使用に近い">未使用に近い</option>
                    <option value="目立った傷や汚れなし">
                      目立った傷や汚れなし
                    </option>
                    <option value="やや傷や汚れあり">
                      やや傷や汚れあり
                    </option>
                    <option value="傷や汚れあり">傷や汚れあり</option>
                  </select>
                </label>

                <MediaUploader
                  label="商品状態の写真"
                  hint="傷・汚れ・タグ・付属品など、購入者が状態を確認できる写真を追加してください。必須項目です。"
                  emptyText="商品状態の写真が登録されていません。"
                  selectButtonLabel="写真を追加"
                  selectingButtonLabel="追加中..."
                  accept="image/*"
                  multiple
                  items={conditionMediaItems}
                  currentIndex={conditionMediaCurrentIndex}
                  inputRef={conditionMediaInputRef}
                  carouselRef={conditionMediaCarouselRef}
                  onFilesSelected={handleConditionMediaSelected}
                  onRemoveItem={handleRemoveConditionMedia}
                  onCarouselScroll={handleConditionMediaCarouselScroll}
                  onMoveToSlide={handleMoveToConditionMediaSlide}
                />

                <Textbox
                  label="説明文"
                  value={description}
                  placeholder="購入時期、着用回数、保管状態などを入力してください。"
                  rows={6}
                  helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
                  counterText={`${description.length}/1000`}
                  maxLength={1000}
                  onChange={(event) =>
                    setDescription(event.currentTarget.value)
                  }
                />
              </div>
            </section>

            {errorMessage ? (
              <p className="page-error" role="alert">
                {errorMessage}
              </p>
            ) : null}
          </div>
        )}
      </section>
    </Layout>
  );
}
