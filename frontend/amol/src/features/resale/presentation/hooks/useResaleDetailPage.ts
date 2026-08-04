// frontend/amol/src/features/resale/presentation/hooks/useResaleDetailPage.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  formatDateTime,
} from "../../../../components/utils/date";
import {
  formatYen,
} from "../../../../components/utils/price";
import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import {
  addMyResaleConditionImages,
  deleteMyResaleConditionImage,
  deleteResaleListing,
  getMyResaleListing,
  listMyResaleConditionImages,
  updatePrimaryResaleImage,
  updateResaleListing,
} from "../../api/resaleApi";

import type {
  ResaleConditionImage,
} from "../../api/resaleApi";

import {
  DEFAULT_RESALE_CONDITION,
  normalizeResaleCondition,
} from "../../constants/resaleConditions";

import {
  DEFAULT_RESALE_EDITABLE_STATUS,
  isResaleEditableStatus,
  normalizeResaleEditableStatus,
} from "../../constants/resaleStatusOptions";

import type {
  ResaleCondition,
  ResaleEditableStatus,
} from "../../../shared/types/resale";

import type {
  ResaleDetailEditFormProps,
  ResaleDetailFooterProps,
  ResaleDetailModelInfoProps,
  ResaleDetailReadonlyInfoProps,
  ResaleListingTargetSummary,
  ResaleListingWithModel,
} from "../types/resaleDetailPageTypes";

import {
  formatResaleMeasurements,
  formatResaleModelColor,
  formatResaleModelKind,
  formatResaleModelVolume,
  formatResaleStatus,
  resolveResaleTokenIconUrl,
} from "../utils/resaleDetailFormatters";

import {
  createResaleGalleryItems,
  sortResaleConditionImages,
} from "../utils/resaleDetailImages";

import {
  formatResalePriceInput,
  normalizeResalePriceInput,
  parseResalePriceInput,
} from "../utils/resalePriceInput";

import {
  useResaleDetailConditionMedia,
} from "./useResaleDetailConditionMedia";

export function useResaleDetailPage() {
  const navigate = useNavigate();

  const {
    resaleId,
  } = useParams<{
    resaleId: string;
  }>();

  const normalizedResaleId =
    textOrEmpty(resaleId);

  const loadRequestIdRef =
    useRef(0);

  const [
    item,
    setItem,
  ] = useState<
    ResaleListingWithModel | null
  >(null);

  const [
    images,
    setImages,
  ] = useState<
    ResaleConditionImage[]
  >([]);

  const [
    activeGalleryIndex,
    setActiveGalleryIndex,
  ] = useState(0);

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    saving,
    setSaving,
  ] = useState(false);

  const [
    isEditing,
    setIsEditing,
  ] = useState(false);

  const [
    errorMessage,
    setErrorMessage,
  ] = useState("");

  const [
    saveMessage,
    setSaveMessage,
  ] = useState("");

  const [
    priceInput,
    setPriceInput,
  ] = useState("");

  const [
    conditionInput,
    setConditionInput,
  ] = useState<ResaleCondition>(
    DEFAULT_RESALE_CONDITION,
  );

  const [
    descriptionInput,
    setDescriptionInput,
  ] = useState("");

  const [
    statusInput,
    setStatusInput,
  ] = useState<ResaleEditableStatus>(
    DEFAULT_RESALE_EDITABLE_STATUS,
  );

  const {
    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,
    deletedImageIds,

    resetConditionMedia,
    getNewConditionFiles,

    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
  } = useResaleDetailConditionMedia();

  const clearMessages =
    useCallback(() => {
      setErrorMessage("");
      setSaveMessage("");
    }, []);

  const resetFormFromItem =
    useCallback(
      (
        nextItem:
          ResaleListingWithModel | null,
        nextImages:
          ResaleConditionImage[],
      ) => {
        const nextPrice =
          Number(
            nextItem?.price ?? 0,
          );

        setPriceInput(
          Number.isFinite(nextPrice) &&
            nextPrice > 0
            ? String(nextPrice)
            : "",
        );

        setConditionInput(
          normalizeResaleCondition(
            nextItem?.condition,
          ),
        );

        setDescriptionInput(
          textOrEmpty(
            nextItem?.description,
          ),
        );

        setStatusInput(
          normalizeResaleEditableStatus(
            nextItem?.status,
          ),
        );

        resetConditionMedia(
          nextImages,
        );
      },
      [
        resetConditionMedia,
      ],
    );

  const loadDetail =
    useCallback(
      async (): Promise<void> => {
        const requestId =
          ++loadRequestIdRef.current;

        if (!normalizedResaleId) {
          setItem(null);
          setImages([]);
          resetFormFromItem(
            null,
            [],
          );
          setActiveGalleryIndex(0);
          setIsEditing(false);
          setErrorMessage(
            "出品情報が見つかりません。",
          );
          setSaveMessage("");
          setLoading(false);
          return;
        }

        setLoading(true);
        setErrorMessage("");
        setSaveMessage("");

        try {
          const [
            nextItem,
            nextImages,
          ] = await Promise.all([
            getMyResaleListing(
              normalizedResaleId,
            ),
            listMyResaleConditionImages(
              normalizedResaleId,
            ),
          ]);

          if (
            requestId !==
            loadRequestIdRef.current
          ) {
            return;
          }

          if (!nextItem) {
            setItem(null);
            setImages([]);
            resetFormFromItem(
              null,
              [],
            );
            setActiveGalleryIndex(0);
            setIsEditing(false);
            setErrorMessage(
              "出品情報が見つかりません。",
            );
            return;
          }

          const sortedNextImages =
            sortResaleConditionImages(
              nextImages,
            );

          setItem(nextItem);
          setImages(
            sortedNextImages,
          );
          resetFormFromItem(
            nextItem,
            sortedNextImages,
          );
          setActiveGalleryIndex(0);
          setIsEditing(false);
        } catch (error) {
          if (
            requestId !==
            loadRequestIdRef.current
          ) {
            return;
          }

          setItem(null);
          setImages([]);
          resetFormFromItem(
            null,
            [],
          );
          setActiveGalleryIndex(0);
          setIsEditing(false);
          setErrorMessage(
            error instanceof Error
              ? error.message
              : "出品情報の取得に失敗しました。",
          );
        } finally {
          if (
            requestId ===
            loadRequestIdRef.current
          ) {
            setLoading(false);
          }
        }
      },
      [
        normalizedResaleId,
        resetFormFromItem,
      ],
    );

  useEffect(() => {
    void loadDetail();

    return () => {
      loadRequestIdRef.current += 1;
    };
  }, [
    loadDetail,
  ]);

  const sortedImages =
    useMemo(
      () =>
        sortResaleConditionImages(
          images,
        ),
      [
        images,
      ],
    );

  const galleryItems =
    useMemo(
      () =>
        createResaleGalleryItems(
          sortedImages,
        ),
      [
        sortedImages,
      ],
    );

  useEffect(() => {
    setActiveGalleryIndex(
      (currentIndex) => {
        if (
          galleryItems.length === 0
        ) {
          return 0;
        }

        return Math.min(
          currentIndex,
          galleryItems.length - 1,
        );
      },
    );
  }, [
    galleryItems.length,
  ]);

  const productName =
    textOrEmpty(
      item?.productName,
    );

  const tokenName =
    textOrEmpty(
      item?.tokenName,
    );

  const brandName =
    textOrEmpty(
      item?.brandName,
    );

  const condition =
    textOrEmpty(
      item?.condition,
    );

  const description =
    textOrEmpty(
      item?.description,
    );

  const status =
    textOrEmpty(
      item?.status,
    );

  const modelId =
    textOrEmpty(
      item?.modelId,
    );

  const modelKind =
    textOrEmpty(
      item?.kind,
    );

  const modelNumber =
    textOrEmpty(
      item?.modelNumber,
    );

  const modelSize =
    textOrEmpty(
      item?.size,
    );

  const tokenIconUrl =
    resolveResaleTokenIconUrl(
      item,
    );

  const modelKindLabel =
    modelKind
      ? formatResaleModelKind(
          modelKind,
        )
      : "";

  const modelColorLabel =
    formatResaleModelColor(
      item?.color,
    );

  const modelVolumeLabel =
    formatResaleModelVolume(
      item?.volume,
    );

  const measurementsLabel =
    formatResaleMeasurements(
      item?.measurements,
    );

  const hasModelInfo =
    Boolean(modelId) ||
    Boolean(modelKindLabel) ||
    Boolean(modelNumber) ||
    Boolean(modelSize) ||
    modelColorLabel !== "-" ||
    modelVolumeLabel !== "-" ||
    measurementsLabel !== "-";

  const isSold =
    status === "sold";

  const title =
    productName ||
    tokenName ||
    "出品詳細";

  const priceLabel =
    typeof item?.price ===
      "number" &&
    Number.isFinite(
      item.price,
    ) &&
    item.price > 0
      ? formatYen(
          item.price,
          "-",
        )
      : "-";

  const editablePriceLabel =
    formatResalePriceInput(
      priceInput,
    );

  const createdAtLabel =
    formatDateTime(
      item?.createdAt,
    );

  const updatedAtLabel =
    formatDateTime(
      item?.updatedAt,
    );

  const statusLabel =
    formatResaleStatus(
      status,
    );

  const priceNumber =
    parseResalePriceInput(
      priceInput,
    );

  const hasValidPrice =
    priceNumber !== null &&
    priceNumber > 0;

  const canSave =
    isEditing &&
    !isSold &&
    !saving &&
    Boolean(
      normalizedResaleId,
    ) &&
    hasValidPrice &&
    isResaleEditableStatus(
      statusInput,
    ) &&
    conditionMediaItems.length > 0;

  const canEdit =
    !loading &&
    Boolean(item) &&
    !isEditing &&
    !isSold;

  const handlePrevGalleryItem =
    useCallback(() => {
      if (
        galleryItems.length <= 1
      ) {
        return;
      }

      setActiveGalleryIndex(
        (currentIndex) =>
          currentIndex <= 0
            ? galleryItems.length - 1
            : currentIndex - 1,
      );
    }, [
      galleryItems.length,
    ]);

  const handleNextGalleryItem =
    useCallback(() => {
      if (
        galleryItems.length <= 1
      ) {
        return;
      }

      setActiveGalleryIndex(
        (currentIndex) =>
          currentIndex >=
          galleryItems.length - 1
            ? 0
            : currentIndex + 1,
      );
    }, [
      galleryItems.length,
    ]);

  const handleSelectGalleryItem =
    useCallback(
      (
        index: number,
      ) => {
        if (
          index < 0 ||
          index >=
            galleryItems.length
        ) {
          return;
        }

        setActiveGalleryIndex(
          index,
        );
      },
      [
        galleryItems.length,
      ],
    );

  const handlePriceChange =
    useCallback(
      (
        value: string,
      ) => {
        setPriceInput(
          normalizeResalePriceInput(
            value,
          ),
        );
        clearMessages();
      },
      [
        clearMessages,
      ],
    );

  const handleConditionChange =
    useCallback(
      (
        value:
          ResaleCondition,
      ) => {
        setConditionInput(
          value,
        );
        clearMessages();
      },
      [
        clearMessages,
      ],
    );

  const handleStatusChange =
    useCallback(
      (
        value:
          ResaleEditableStatus,
      ) => {
        setStatusInput(
          value,
        );
        clearMessages();
      },
      [
        clearMessages,
      ],
    );

  const handleDescriptionChange =
    useCallback(
      (
        value: string,
      ) => {
        setDescriptionInput(
          value,
        );
        clearMessages();
      },
      [
        clearMessages,
      ],
    );

  const handleStartEdit =
    useCallback(() => {
      if (isSold) {
        setErrorMessage(
          "売却済みの出品は編集できません。",
        );
        return;
      }

      resetFormFromItem(
        item,
        images,
      );
      setIsEditing(true);
      clearMessages();
    }, [
      clearMessages,
      images,
      isSold,
      item,
      resetFormFromItem,
    ]);

  const handleCancelEdit =
    useCallback(() => {
      resetFormFromItem(
        item,
        images,
      );
      setIsEditing(false);
      clearMessages();
    }, [
      clearMessages,
      images,
      item,
      resetFormFromItem,
    ]);

  const handleSave =
    useCallback(
      async (): Promise<void> => {
        if (isSold) {
          setErrorMessage(
            "売却済みの出品は編集できません。",
          );
          return;
        }

        if (
          !canSave ||
          priceNumber === null ||
          priceNumber <= 0
        ) {
          setErrorMessage(
            "販売価格、商品の状態、公開状態、商品状態の写真を入力してください。",
          );
          return;
        }

        setSaving(true);
        clearMessages();

        try {
          const newFiles =
            getNewConditionFiles();

          const nextDisplayOrder =
            conditionMediaItems.reduce(
              (
                currentMax,
                mediaItem,
              ) => {
                if (
                  mediaItem.source !==
                  "existing"
                ) {
                  return currentMax;
                }

                const displayOrder =
                  Number(
                    mediaItem.image
                      ?.displayOrder ??
                      -1,
                  );

                return Number.isFinite(
                  displayOrder,
                )
                  ? Math.max(
                      currentMax,
                      displayOrder,
                    )
                  : currentMax;
              },
              -1,
            ) + 1;

          await updateResaleListing({
            resaleId:
              normalizedResaleId,
            price:
              priceNumber,
            condition:
              conditionInput,
            description:
              descriptionInput,
            status:
              statusInput,
          });

          await Promise.all(
            deletedImageIds.map(
              (
                imageId,
              ) =>
                deleteMyResaleConditionImage({
                  resaleId:
                    normalizedResaleId,
                  imageId,
                }),
            ),
          );

          if (
            newFiles.length > 0
          ) {
            await addMyResaleConditionImages({
              resaleId:
                normalizedResaleId,
              files:
                newFiles,
              startDisplayOrder:
                nextDisplayOrder,
            });
          }

          const nextImages =
            sortResaleConditionImages(
              await listMyResaleConditionImages(
                normalizedResaleId,
              ),
            );

          if (
            nextImages.length > 0
          ) {
            await updatePrimaryResaleImage({
              resaleId:
                normalizedResaleId,
              imageId:
                nextImages[0].id,
            });
          }

          const refreshedItem =
            await getMyResaleListing(
              normalizedResaleId,
            );

          if (!refreshedItem) {
            throw new Error(
              "更新後の出品情報を取得できませんでした。",
            );
          }

          setItem(
            refreshedItem,
          );
          setImages(
            nextImages,
          );
          resetFormFromItem(
            refreshedItem,
            nextImages,
          );
          setActiveGalleryIndex(0);
          setIsEditing(false);
          setSaveMessage(
            "出品情報を更新しました。",
          );
        } catch (error) {
          setErrorMessage(
            error instanceof Error
              ? error.message
              : "出品情報の更新に失敗しました。",
          );
        } finally {
          setSaving(false);
        }
      },
      [
        canSave,
        clearMessages,
        conditionInput,
        conditionMediaItems,
        deletedImageIds,
        descriptionInput,
        getNewConditionFiles,
        isSold,
        normalizedResaleId,
        priceNumber,
        resetFormFromItem,
        statusInput,
      ],
    );

  const handleDelete =
    useCallback(
      async (): Promise<void> => {
        if (
          !normalizedResaleId ||
          saving
        ) {
          return;
        }

        const confirmed =
          window.confirm(
            "この出品を削除します。よろしいですか？",
          );

        if (!confirmed) {
          return;
        }

        setSaving(true);
        clearMessages();

        try {
          await deleteResaleListing(
            normalizedResaleId,
          );

          navigate(
            "/wallet",
            {
              replace: true,
              state: {
                resaleDeleted:
                  true,
                resaleId:
                  normalizedResaleId,
              },
            },
          );
        } catch (error) {
          setErrorMessage(
            error instanceof Error
              ? error.message
              : "出品情報の削除に失敗しました。",
          );
        } finally {
          setSaving(false);
        }
      },
      [
        clearMessages,
        navigate,
        normalizedResaleId,
        saving,
      ],
    );

  const handleBack =
    useCallback(() => {
      navigate(-1);
    }, [
      navigate,
    ]);

  const handleBackToWallet =
    useCallback(() => {
      navigate(
        "/wallet",
      );
    }, [
      navigate,
    ]);

  const handleReload =
    useCallback(
      async (): Promise<void> => {
        await loadDetail();
      },
      [
        loadDetail,
      ],
    );

  const listingTarget =
    useMemo<
      ResaleListingTargetSummary
    >(
      () => ({
        tokenIconUrl,
        tokenName,
        brandName,
        productName,
      }),
      [
        brandName,
        productName,
        tokenIconUrl,
        tokenName,
      ],
    );

  const modelInfoProps =
    useMemo<
      ResaleDetailModelInfoProps
    >(
      () => ({
        hasModelInfo,
        kindLabel:
          modelKindLabel,
        modelNumber,
        size:
          modelSize,
        colorLabel:
          modelColorLabel,
        measurementsLabel,
        volumeLabel:
          modelVolumeLabel,
      }),
      [
        hasModelInfo,
        measurementsLabel,
        modelColorLabel,
        modelKindLabel,
        modelNumber,
        modelSize,
        modelVolumeLabel,
      ],
    );

  const readonlyInfoProps =
    useMemo<
      ResaleDetailReadonlyInfoProps
    >(
      () => ({
        galleryItems,
        activeGalleryIndex,

        priceLabel,
        conditionLabel:
          condition || "-",
        statusLabel,
        createdAtLabel,
        updatedAtLabel,
        description,

        onPrevGalleryItem:
          handlePrevGalleryItem,
        onNextGalleryItem:
          handleNextGalleryItem,
        onSelectGalleryItem:
          handleSelectGalleryItem,
      }),
      [
        activeGalleryIndex,
        condition,
        createdAtLabel,
        description,
        galleryItems,
        handleNextGalleryItem,
        handlePrevGalleryItem,
        handleSelectGalleryItem,
        priceLabel,
        statusLabel,
        updatedAtLabel,
      ],
    );

  const editFormProps =
    useMemo<
      ResaleDetailEditFormProps
    >(
      () => ({
        priceValue:
          editablePriceLabel,
        condition:
          conditionInput,
        status:
          statusInput,
        description:
          descriptionInput,
        saving,

        createdAtLabel,
        updatedAtLabel,

        conditionMediaItems,
        conditionMediaCurrentIndex,
        conditionMediaInputRef,
        conditionMediaCarouselRef,

        onPriceChange:
          handlePriceChange,
        onConditionChange:
          handleConditionChange,
        onStatusChange:
          handleStatusChange,
        onDescriptionChange:
          handleDescriptionChange,

        onConditionMediaSelected:
          handleConditionMediaSelected,
        onRemoveConditionMedia:
          handleRemoveConditionMedia,
        onConditionMediaCarouselScroll:
          handleConditionMediaCarouselScroll,
        onMoveToConditionMediaSlide:
          handleMoveToConditionMediaSlide,
      }),
      [
        conditionInput,
        conditionMediaCarouselRef,
        conditionMediaCurrentIndex,
        conditionMediaInputRef,
        conditionMediaItems,
        createdAtLabel,
        descriptionInput,
        editablePriceLabel,
        handleConditionChange,
        handleConditionMediaCarouselScroll,
        handleConditionMediaSelected,
        handleDescriptionChange,
        handleMoveToConditionMediaSlide,
        handlePriceChange,
        handleRemoveConditionMedia,
        handleStatusChange,
        saving,
        statusInput,
        updatedAtLabel,
      ],
    );

  const footerProps =
    useMemo<
      ResaleDetailFooterProps | undefined
    >(
      () => {
        if (
          isEditing &&
          !isSold
        ) {
          return {
            variant:
              "tripleAction",
            leftButtonLabel:
              "キャンセル",
            centerButtonLabel:
              saving
                ? "保存中..."
                : "保存する",
            rightButtonLabel:
              "削除",
            leftButtonDisabled:
              saving,
            centerButtonDisabled:
              !canSave,
            rightButtonDisabled:
              saving,
            onLeftButtonClick:
              handleCancelEdit,
            onCenterButtonClick:
              handleSave,
            onRightButtonClick:
              handleDelete,
          };
        }

        if (canEdit) {
          return {
            variant:
              "action",
            buttonLabel:
              "編集する",
            disabled:
              false,
            onButtonClick:
              handleStartEdit,
          };
        }

        return undefined;
      },
      [
        canEdit,
        canSave,
        handleCancelEdit,
        handleDelete,
        handleSave,
        handleStartEdit,
        isEditing,
        isSold,
        saving,
      ],
    );

  return {
    title,
    footerProps,

    loading,
    item,
    isEditing,
    isSold,

    errorMessage,
    saveMessage,

    listingTarget,
    modelInfoProps,
    readonlyInfoProps,
    editFormProps,

    handleBack,
    handleReload,
    handleBackToWallet,
  };
}