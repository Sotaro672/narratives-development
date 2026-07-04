// frontend/amol/src/pages/ResaleDetailPage.tsx
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import MediaGallery from "../components/ui/MediaGallery";
import type { MediaGalleryItem } from "../components/ui/MediaGallery";
import MediaIcon from "../components/ui/MediaIcon";
import MediaUploader, {
  type MediaUploaderItem,
} from "../components/ui/MediaUploader";
import SectionHeader from "../components/ui/SectionHeader";
import Textbox from "../components/ui/Textbox";
import { fetchCurrentAvatarId } from "../features/catalog/infrastructure/avatarStateRepository";
import {
  addMyResaleConditionImages,
  deleteMyResaleConditionImage,
  deleteResaleListing,
  listMyResaleConditionImages,
  listMyResaleListings,
  updatePrimaryResaleImage,
  updateResaleListing,
} from "../features/resale/api/resaleApi";
import type {
  ResaleConditionImage,
  ResaleListing,
} from "../features/resale/api/resaleApi";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../styles/resale-detail-page.css";

const CONDITION_OPTIONS = [
  "新品・未使用",
  "未使用に近い",
  "目立った傷や汚れなし",
  "やや傷や汚れあり",
  "傷や汚れあり",
];

const RESALE_STATUS_OPTIONS = [
  {
    value: "listing",
    label: "出品中",
  },
  {
    value: "suspended",
    label: "公開停止",
  },
];

type EditableConditionMediaItem = MediaUploaderItem & {
  source: "existing" | "new";
  file?: File;
  image?: ResaleConditionImage;
};

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

function formatPriceInput(value: string): string {
  const digits = value.replace(/[^\d]/g, "");

  if (!digits) {
    return "";
  }

  return Number(digits).toLocaleString("ja-JP");
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

function formatResaleStatus(value: string): string {
  switch (value) {
    case "listing":
      return "出品中";
    case "suspended":
      return "公開停止";
    case "sold":
      return "売却済み";
    default:
      return value || "-";
  }
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

function createEditableImageItem(
  image: ResaleConditionImage,
): EditableConditionMediaItem {
  return {
    id: image.id,
    type: "image",
    previewUrl: image.url,
    title: image.fileName || "商品状態の写真",
    fileName: image.fileName,
    source: "existing",
    image,
  };
}

function createNewImageItem(file: File): EditableConditionMediaItem {
  return {
    id: `${file.name}-${file.size}-${file.lastModified}-${crypto.randomUUID()}`,
    type: "image",
    previewUrl: URL.createObjectURL(file),
    title: file.name,
    fileName: file.name,
    source: "new",
    file,
  };
}

function createGalleryItem(image: ResaleConditionImage): MediaGalleryItem {
  return {
    id: image.id,
    url: image.url,
    fileName: image.fileName,
    type: image.mimeType,
  };
}

function getTokenIconUrl(item: ResaleListing | null): string {
  if (!item) {
    return "";
  }

  const record = item as ResaleListing & {
    imageUrl?: string;
    tokenIcon?: string;
    tokenIconUrl?: string;
    metadata?: {
      image?: string;
    };
  };

  return (
    normalizeText(record.tokenIconUrl) ||
    normalizeText(record.tokenIcon) ||
    normalizeText(record.imageUrl) ||
    normalizeText(record.metadata?.image)
  );
}

export default function ResaleDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const conditionMediaInputRef = useRef<HTMLInputElement>(null);
  const conditionMediaCarouselRef = useRef<HTMLDivElement>(null);

  const normalizedResaleId = normalizeText(resaleId);

  const [item, setItem] = useState<ResaleListing | null>(null);
  const [images, setImages] = useState<ResaleConditionImage[]>([]);
  const [activeGalleryIndex, setActiveGalleryIndex] = useState(0);

  const [conditionMediaItems, setConditionMediaItems] = useState<
    EditableConditionMediaItem[]
  >([]);
  const [conditionMediaCurrentIndex, setConditionMediaCurrentIndex] =
    useState(0);
  const [deletedImageIds, setDeletedImageIds] = useState<string[]>([]);

  const [currentAvatarId, setCurrentAvatarId] = useState("");
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [isEditing, setIsEditing] = useState<boolean>(false);

  const [errorMessage, setErrorMessage] = useState("");
  const [saveMessage, setSaveMessage] = useState("");

  const [priceInput, setPriceInput] = useState("");
  const [conditionInput, setConditionInput] = useState("未使用に近い");
  const [descriptionInput, setDescriptionInput] = useState("");
  const [statusInput, setStatusInput] = useState("listing");

  const sortedImages = useMemo(() => sortImages(images), [images]);
  const galleryItems = useMemo(
    () => sortedImages.map(createGalleryItem),
    [sortedImages],
  );

  const productName = normalizeText(item?.productName);
  const tokenName = normalizeText(item?.tokenName);
  const brandName = normalizeText(item?.brandName);
  const condition = normalizeText(item?.condition);
  const description = normalizeText(item?.description);
  const status = normalizeText(item?.status);
  const resaleAvatarId = normalizeText(item?.avatarId);
  const tokenIconUrl = getTokenIconUrl(item);

  const isSold = status === "sold";

  const isOwnResale =
    Boolean(currentAvatarId) &&
    Boolean(resaleAvatarId) &&
    currentAvatarId === resaleAvatarId;

  const title = productName || tokenName || "出品詳細";

  const priceLabel = formatPrice(item?.price);
  const editablePriceLabel = formatPriceInput(priceInput);
  const createdAtLabel = formatDateTime(item?.createdAt);
  const updatedAtLabel = formatDateTime(item?.updatedAt);
  const statusLabel = formatResaleStatus(status);

  const priceNumber = Number(priceInput.replace(/[^\d]/g, ""));
  const hasValidPrice = Number.isFinite(priceNumber) && priceNumber > 0;
  const hasValidEditableStatus = RESALE_STATUS_OPTIONS.some(
    (option) => option.value === statusInput,
  );

  const canSave =
    isEditing &&
    isOwnResale &&
    !isSold &&
    !saving &&
    Boolean(normalizedResaleId) &&
    hasValidPrice &&
    Boolean(conditionInput.trim()) &&
    hasValidEditableStatus &&
    conditionMediaItems.length > 0;

  const canEdit =
    !loading && Boolean(item) && isOwnResale && !isEditing && !isSold;

  const resetFormFromItem = useCallback(
    (nextItem: ResaleListing | null, nextImages: ResaleConditionImage[]) => {
      const nextPrice = Number(nextItem?.price ?? 0);
      const nextStatus = normalizeText(nextItem?.status);

      setPriceInput(
        Number.isFinite(nextPrice) && nextPrice > 0 ? String(nextPrice) : "",
      );
      setConditionInput(normalizeText(nextItem?.condition) || "未使用に近い");
      setDescriptionInput(normalizeText(nextItem?.description));
      setStatusInput(
        nextStatus === "suspended" || nextStatus === "listing"
          ? nextStatus
          : "listing",
      );
      setConditionMediaItems(sortImages(nextImages).map(createEditableImageItem));
      setConditionMediaCurrentIndex(0);
      setDeletedImageIds([]);
    },
    [],
  );

  const revokeNewPreviewUrls = useCallback(() => {
    conditionMediaItems.forEach((mediaItem) => {
      if (mediaItem.source === "new" && mediaItem.previewUrl) {
        URL.revokeObjectURL(mediaItem.previewUrl);
      }
    });
  }, [conditionMediaItems]);

  const loadDetail = useCallback(async () => {
    if (!normalizedResaleId) {
      setErrorMessage("出品情報が見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setErrorMessage("");
    setSaveMessage("");

    try {
      const [nextCurrentAvatarId, result] = await Promise.all([
        fetchCurrentAvatarId(),
        listMyResaleListings({
          page: 1,
          perPage: 100,
        }),
      ]);

      const nextItem =
        result.items?.find(
          (listing) => normalizeText(listing.id) === normalizedResaleId,
        ) ?? null;

      if (!nextItem) {
        setCurrentAvatarId(nextCurrentAvatarId);
        setItem(null);
        setImages([]);
        resetFormFromItem(null, []);
        setActiveGalleryIndex(0);
        setErrorMessage("出品情報が見つかりません。");
        return;
      }

      const nextImages = await listMyResaleConditionImages(normalizedResaleId);

      setCurrentAvatarId(nextCurrentAvatarId);
      setItem(nextItem);
      setImages(nextImages);
      resetFormFromItem(nextItem, nextImages);
      setActiveGalleryIndex(0);
      setIsEditing(false);
    } catch (error) {
      setItem(null);
      setImages([]);
      resetFormFromItem(null, []);
      setActiveGalleryIndex(0);
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "出品情報の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [normalizedResaleId, resetFormFromItem]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  useEffect(() => {
    return () => {
      revokeNewPreviewUrls();
    };
  }, [revokeNewPreviewUrls]);

  const handlePrevGalleryItem = () => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveGalleryIndex((current) =>
      current <= 0 ? galleryItems.length - 1 : current - 1,
    );
  };

  const handleNextGalleryItem = () => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveGalleryIndex((current) =>
      current >= galleryItems.length - 1 ? 0 : current + 1,
    );
  };

  const handleSelectGalleryItem = (index: number) => {
    if (index < 0 || index >= galleryItems.length) {
      return;
    }

    setActiveGalleryIndex(index);
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

  const handleChangePrice = (event: ChangeEvent<HTMLInputElement>) => {
    setPriceInput(event.currentTarget.value.replace(/[^\d]/g, ""));
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleChangeCondition = (event: ChangeEvent<HTMLSelectElement>) => {
    setConditionInput(event.currentTarget.value);
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleChangeStatus = (event: ChangeEvent<HTMLSelectElement>) => {
    setStatusInput(event.currentTarget.value);
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleChangeDescription = (
    event: ChangeEvent<HTMLTextAreaElement>,
  ) => {
    setDescriptionInput(event.currentTarget.value);
    setSaveMessage("");
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

    const nextItems = files.map(createNewImageItem);

    setConditionMediaItems((current) => [...current, ...nextItems]);
    setSaveMessage("");
    setErrorMessage("");
    event.currentTarget.value = "";
  };

  const handleRemoveConditionMedia = (id: string) => {
    setConditionMediaItems((current) => {
      const removingItem = current.find((mediaItem) => mediaItem.id === id);

      if (removingItem?.source === "new" && removingItem.previewUrl) {
        URL.revokeObjectURL(removingItem.previewUrl);
      }

      if (removingItem?.source === "existing") {
        setDeletedImageIds((currentIds) =>
          currentIds.includes(id) ? currentIds : [...currentIds, id],
        );
      }

      const nextItems = current.filter((mediaItem) => mediaItem.id !== id);

      setConditionMediaCurrentIndex((currentIndex) => {
        if (nextItems.length === 0) {
          return 0;
        }

        return Math.min(currentIndex, nextItems.length - 1);
      });

      return nextItems;
    });

    setSaveMessage("");
    setErrorMessage("");
  };

  const handleStartEdit = () => {
    if (!isOwnResale) {
      setErrorMessage("この出品はログイン中のアバターの出品ではありません。");
      return;
    }

    if (isSold) {
      setErrorMessage("売却済みの出品は編集できません。");
      return;
    }

    resetFormFromItem(item, images);
    setIsEditing(true);
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleCancelEdit = () => {
    revokeNewPreviewUrls();
    resetFormFromItem(item, images);
    setIsEditing(false);
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleSave = async () => {
    if (!isOwnResale) {
      setErrorMessage("この出品はログイン中のアバターの出品ではありません。");
      return;
    }

    if (isSold) {
      setErrorMessage("売却済みの出品は編集できません。");
      return;
    }

    if (!canSave) {
      setErrorMessage(
        "販売価格、商品の状態、公開状態、商品状態の写真を入力してください。",
      );
      return;
    }

    setSaving(true);
    setErrorMessage("");
    setSaveMessage("");

    try {
      const newFiles = conditionMediaItems
        .filter((mediaItem) => mediaItem.source === "new" && mediaItem.file)
        .map((mediaItem) => mediaItem.file as File);

      await updateResaleListing({
        resaleId: normalizedResaleId,
        price: priceNumber,
        condition: conditionInput,
        description: descriptionInput,
        status: statusInput,
      });

      await Promise.all(
        deletedImageIds.map((imageId) =>
          deleteMyResaleConditionImage({
            resaleId: normalizedResaleId,
            imageId,
          }),
        ),
      );

      if (newFiles.length > 0) {
        await addMyResaleConditionImages({
          resaleId: normalizedResaleId,
          files: newFiles,
          startDisplayOrder: conditionMediaItems.length,
        });
      }

      const [result, nextImages] = await Promise.all([
        listMyResaleListings({
          page: 1,
          perPage: 100,
        }),
        listMyResaleConditionImages(normalizedResaleId),
      ]);

      const nextItem =
        result.items?.find(
          (listing) => normalizeText(listing.id) === normalizedResaleId,
        ) ?? null;

      const sortedNextImages = sortImages(nextImages);

      if (sortedNextImages.length > 0) {
        const primaryImageId = sortedNextImages[0].id;

        await updatePrimaryResaleImage({
          resaleId: normalizedResaleId,
          imageId: primaryImageId,
        });
      }

      const [refreshedResult, refreshedImages] = await Promise.all([
        listMyResaleListings({
          page: 1,
          perPage: 100,
        }),
        listMyResaleConditionImages(normalizedResaleId),
      ]);

      const refreshedItem =
        refreshedResult.items?.find(
          (listing) => normalizeText(listing.id) === normalizedResaleId,
        ) ?? nextItem;

      setItem(refreshedItem);
      setImages(refreshedImages);
      resetFormFromItem(refreshedItem, refreshedImages);
      setIsEditing(false);
      setActiveGalleryIndex(0);
      setSaveMessage("出品情報を更新しました。");
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "出品情報の更新に失敗しました。",
      );
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!isOwnResale) {
      setErrorMessage("この出品はログイン中のアバターの出品ではありません。");
      return;
    }

    if (!normalizedResaleId || saving) {
      return;
    }

    const ok = window.confirm("この出品を削除します。よろしいですか？");

    if (!ok) {
      return;
    }

    setSaving(true);
    setErrorMessage("");
    setSaveMessage("");

    try {
      await deleteResaleListing(normalizedResaleId);

      navigate("/wallet", {
        replace: true,
        state: {
          resaleDeleted: true,
          resaleId: normalizedResaleId,
        },
      });
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "出品情報の削除に失敗しました。",
      );
    } finally {
      setSaving(false);
    }
  };

  const footerProps =
    isEditing && isOwnResale && !isSold
      ? {
          variant: "tripleAction" as const,
          leftButtonLabel: "キャンセル",
          centerButtonLabel: saving ? "保存中..." : "保存する",
          rightButtonLabel: "削除",
          leftButtonDisabled: saving,
          centerButtonDisabled: !canSave,
          rightButtonDisabled: saving,
          onLeftButtonClick: handleCancelEdit,
          onCenterButtonClick: handleSave,
          onRightButtonClick: handleDelete,
        }
      : canEdit
        ? {
            variant: "action" as const,
            buttonLabel: "編集する",
            disabled: false,
            onButtonClick: handleStartEdit,
          }
        : undefined;

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() => navigate(-1)}
      mode="mypage"
      hideAnnouncementButton
      hideSettingsButton
      showFooter={Boolean(footerProps)}
      footerProps={footerProps}
    >
      <section className="page-section resale-detail-page">
        {loading ? (
          <div className="page-card">
            <p className="page-card__text">読み込み中です...</p>
          </div>
        ) : null}

        {!loading && errorMessage && !item ? (
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

        {!loading && item ? (
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
            </section>

            {!isEditing ? (
              <section className="page-card">
                <SectionHeader title="商品状態の写真" titleAs="h2" />

                <MediaGallery
                  items={galleryItems}
                  activeIndex={activeGalleryIndex}
                  altFallback="商品状態の写真"
                  placeholderText="商品状態の写真はありません。"
                  className="resale-detail-page__gallery"
                  onPrev={handlePrevGalleryItem}
                  onNext={handleNextGalleryItem}
                  onSelect={handleSelectGalleryItem}
                />
              </section>
            ) : null}

            <section className="page-card">
              <SectionHeader title="販売情報" titleAs="h2" />

              {isEditing ? (
                <div className="page-form">
                  <Input
                    label="販売価格"
                    type="text"
                    inputMode="numeric"
                    value={editablePriceLabel}
                    required
                    helperText="半角数字で入力してください。"
                    onChange={handleChangePrice}
                  />

                  <label className="page-form__field">
                    <span className="page-form__label">商品の状態</span>
                    <select
                      value={conditionInput}
                      onChange={handleChangeCondition}
                    >
                      {CONDITION_OPTIONS.map((option) => (
                        <option key={option} value={option}>
                          {option}
                        </option>
                      ))}
                    </select>
                  </label>

                  <label className="page-form__field">
                    <span className="page-form__label">公開状態</span>
                    <select value={statusInput} onChange={handleChangeStatus}>
                      {RESALE_STATUS_OPTIONS.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
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
                    disabled={saving}
                    selecting={saving}
                    inputRef={conditionMediaInputRef}
                    carouselRef={conditionMediaCarouselRef}
                    onFilesSelected={handleConditionMediaSelected}
                    onRemoveItem={handleRemoveConditionMedia}
                    onCarouselScroll={handleConditionMediaCarouselScroll}
                    onMoveToSlide={handleMoveToConditionMediaSlide}
                  />

                  <Textbox
                    label="説明文"
                    value={descriptionInput}
                    rows={6}
                    maxLength={1000}
                    helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
                    counterText={`${descriptionInput.length}/1000`}
                    onChange={handleChangeDescription}
                  />
                </div>
              ) : (
                <div className="page-stack">
                  <dl className="page-definition-list">
                    <div className="page-definition-list__row">
                      <dt>販売価格</dt>
                      <dd>{priceLabel}</dd>
                    </div>

                    <div className="page-definition-list__row">
                      <dt>商品の状態</dt>
                      <dd>{condition || "-"}</dd>
                    </div>

                    <div className="page-definition-list__row">
                      <dt>出品ステータス</dt>
                      <dd>{statusLabel}</dd>
                    </div>

                    <div className="page-definition-list__row">
                      <dt>出品日時</dt>
                      <dd>{createdAtLabel}</dd>
                    </div>

                    <div className="page-definition-list__row">
                      <dt>更新日時</dt>
                      <dd>{updatedAtLabel}</dd>
                    </div>
                  </dl>

                  <div>
                    <h3 className="page-card__subtitle">説明文</h3>
                    <p className="page-card__text resale-detail-page__description">
                      {description || "説明文はありません。"}
                    </p>
                  </div>
                </div>
              )}

              {isEditing ? (
                <dl className="page-definition-list resale-detail-page__readonly-meta">
                  <div className="page-definition-list__row">
                    <dt>出品日時</dt>
                    <dd>{createdAtLabel}</dd>
                  </div>

                  <div className="page-definition-list__row">
                    <dt>更新日時</dt>
                    <dd>{updatedAtLabel}</dd>
                  </div>
                </dl>
              ) : null}

              {errorMessage ? (
                <p className="page-error" role="alert">
                  {errorMessage}
                </p>
              ) : null}

              {saveMessage ? (
                <p className="page-card__text" role="status">
                  {saveMessage}
                </p>
              ) : null}

              {!isEditing && !isOwnResale ? (
                <p className="page-card__text">
                  この出品はログイン中のアバターの出品ではないため編集できません。
                </p>
              ) : null}

              {!isEditing && isOwnResale && isSold ? (
                <p className="page-card__text">
                  売却済みの出品は編集できません。
                </p>
              ) : null}
            </section>

            <div className="resale-detail-page__footer-spacer" />
          </div>
        ) : null}
      </section>
    </Layout>
  );
}