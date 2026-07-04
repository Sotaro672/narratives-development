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
import Dropdown from "../components/ui/Dropdown";
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

type ResaleConditionValue =
  | "新品・未使用"
  | "未使用に近い"
  | "目立った傷や汚れなし"
  | "やや傷や汚れあり"
  | "傷や汚れあり";

const CONDITION_OPTIONS: {
  value: ResaleConditionValue;
  label: string;
}[] = [
  {
    value: "新品・未使用",
    label: "新品・未使用",
  },
  {
    value: "未使用に近い",
    label: "未使用に近い",
  },
  {
    value: "目立った傷や汚れなし",
    label: "目立った傷や汚れなし",
  },
  {
    value: "やや傷や汚れあり",
    label: "やや傷や汚れあり",
  },
  {
    value: "傷や汚れあり",
    label: "傷や汚れあり",
  },
];

type ResaleEditableStatus = "listing" | "suspended";

const RESALE_STATUS_OPTIONS: {
  value: ResaleEditableStatus;
  label: string;
}[] = [
  {
    value: "listing",
    label: "出品中",
  },
  {
    value: "suspended",
    label: "公開停止",
  },
];

type ResaleModelColor = {
  name?: string;
  rgb?: number;
};

type ResaleModelVolume = {
  amount?: number;
  value?: number;
  unit?: string;
};

type ResaleListingWithModel = ResaleListing & {
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: ResaleModelColor | null;
  measurements?: Record<string, number> | null;
  volume?: ResaleModelVolume | null;
};

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

function formatModelColor(color: ResaleModelColor | null | undefined): string {
  if (!color) {
    return "-";
  }

  const name = normalizeText(color.name);
  const rgb = Number(color.rgb);

  if (!name && !Number.isFinite(rgb)) {
    return "-";
  }

  if (!Number.isFinite(rgb)) {
    return name || "-";
  }

  return name ? `${name} / RGB: ${rgb}` : `RGB: ${rgb}`;
}

function formatModelVolume(volume: ResaleModelVolume | null | undefined): string {
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

function normalizeEditableStatus(value: unknown): ResaleEditableStatus {
  return value === "suspended" ? "suspended" : "listing";
}

function normalizeCondition(value: unknown): ResaleConditionValue {
  const text = normalizeText(value);

  if (
    text === "新品・未使用" ||
    text === "未使用に近い" ||
    text === "目立った傷や汚れなし" ||
    text === "やや傷や汚れあり" ||
    text === "傷や汚れあり"
  ) {
    return text;
  }

  return "未使用に近い";
}

function getConditionOptionLabel(value: ResaleConditionValue): string {
  return (
    CONDITION_OPTIONS.find((option) => option.value === value)?.label ??
    "未使用に近い"
  );
}

function getStatusOptionLabel(value: ResaleEditableStatus): string {
  return (
    RESALE_STATUS_OPTIONS.find((option) => option.value === value)?.label ??
    "出品中"
  );
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

function getTokenIconUrl(item: ResaleListingWithModel | null): string {
  if (!item) {
    return "";
  }

  const record = item as ResaleListingWithModel & {
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

  const [item, setItem] = useState<ResaleListingWithModel | null>(null);
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
  const [conditionInput, setConditionInput] =
    useState<ResaleConditionValue>("未使用に近い");
  const [descriptionInput, setDescriptionInput] = useState("");
  const [statusInput, setStatusInput] =
    useState<ResaleEditableStatus>("listing");

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

  const modelId = normalizeText(item?.modelId);
  const modelKind = normalizeText(item?.kind);
  const modelNumber = normalizeText(item?.modelNumber);
  const modelSize = normalizeText(item?.size);
  const modelColorLabel = formatModelColor(item?.color);
  const modelVolumeLabel = formatModelVolume(item?.volume);
  const measurementsLabel = formatMeasurements(item?.measurements);

  const hasModelInfo =
    Boolean(modelId) ||
    Boolean(modelKind) ||
    Boolean(modelNumber) ||
    Boolean(modelSize) ||
    modelColorLabel !== "-" ||
    modelVolumeLabel !== "-" ||
    measurementsLabel !== "-";

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
  const selectedConditionLabel = getConditionOptionLabel(conditionInput);
  const selectedStatusLabel = getStatusOptionLabel(statusInput);

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
    (nextItem: ResaleListingWithModel | null, nextImages: ResaleConditionImage[]) => {
      const nextPrice = Number(nextItem?.price ?? 0);
      const nextStatus = normalizeEditableStatus(nextItem?.status);
      const nextCondition = normalizeCondition(nextItem?.condition);

      setPriceInput(
        Number.isFinite(nextPrice) && nextPrice > 0 ? String(nextPrice) : "",
      );
      setConditionInput(nextCondition);
      setDescriptionInput(normalizeText(nextItem?.description));
      setStatusInput(nextStatus);
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
        (result.items as ResaleListingWithModel[] | undefined)?.find(
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

  const handleSelectCondition = (value: ResaleConditionValue) => {
    setConditionInput(value);
    setSaveMessage("");
    setErrorMessage("");
  };

  const handleSelectStatus = (value: ResaleEditableStatus) => {
    setStatusInput(value);
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
        (result.items as ResaleListingWithModel[] | undefined)?.find(
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
        (refreshedResult.items as ResaleListingWithModel[] | undefined)?.find(
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

              {hasModelInfo ? (
                <dl className="page-definition-list resale-detail-page__readonly-meta">
                  {modelKind ? (
                    <div className="page-definition-list__row">
                      <dt>種別</dt>
                      <dd>{formatModelKind(modelKind)}</dd>
                    </div>
                  ) : null}

                  {modelNumber ? (
                    <div className="page-definition-list__row">
                      <dt>モデル番号</dt>
                      <dd>{modelNumber}</dd>
                    </div>
                  ) : null}

                  {modelSize ? (
                    <div className="page-definition-list__row">
                      <dt>サイズ</dt>
                      <dd>{modelSize}</dd>
                    </div>
                  ) : null}

                  {modelColorLabel !== "-" ? (
                    <div className="page-definition-list__row">
                      <dt>カラー</dt>
                      <dd>{modelColorLabel}</dd>
                    </div>
                  ) : null}

                  {measurementsLabel !== "-" ? (
                    <div className="page-definition-list__row">
                      <dt>採寸</dt>
                      <dd>{measurementsLabel}</dd>
                    </div>
                  ) : null}

                  {modelVolumeLabel !== "-" ? (
                    <div className="page-definition-list__row">
                      <dt>容量</dt>
                      <dd>{modelVolumeLabel}</dd>
                    </div>
                  ) : null}
                </dl>
              ) : null}
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

                  <div className="page-form__field">
                    <span className="page-form__label">商品の状態</span>

                    <Dropdown
                      buttonLabel={selectedConditionLabel}
                      items={CONDITION_OPTIONS}
                      selectedValue={conditionInput}
                      onSelect={handleSelectCondition}
                      renderButton={({ isOpen, toggle }) => (
                        <button
                          type="button"
                          className="page-form__dropdown-button"
                          onClick={toggle}
                          aria-expanded={isOpen}
                        >
                          <span>{selectedConditionLabel}</span>
                          <span aria-hidden="true">{isOpen ? "▲" : "▼"}</span>
                        </button>
                      )}
                    />
                  </div>

                  <div className="page-form__field">
                    <span className="page-form__label">公開状態</span>

                    <Dropdown
                      buttonLabel={selectedStatusLabel}
                      items={RESALE_STATUS_OPTIONS}
                      selectedValue={statusInput}
                      onSelect={handleSelectStatus}
                      renderButton={({ isOpen, toggle }) => (
                        <button
                          type="button"
                          className="page-form__dropdown-button"
                          onClick={toggle}
                          aria-expanded={isOpen}
                        >
                          <span>{selectedStatusLabel}</span>
                          <span aria-hidden="true">{isOpen ? "▲" : "▼"}</span>
                        </button>
                      )}
                    />
                  </div>

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