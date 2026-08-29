// frontend/amol/src/features/resale/presentation/hooks/useResaleDetailPage.ts

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { formatDateTime } from "../../../../components/utils/date";
import { formatYen } from "../../../../components/utils/price";

import {
  addMyResaleConditionImages,
  createMyResaleComment,
  deleteMyResaleComment,
  deleteMyResaleConditionImage,
  deleteResaleListing,
  fetchMyResaleComments,
  getMyResaleListing,
  listMyResaleConditionImages,
  updatePrimaryResaleImage,
  updateResaleListing,
} from "../../api/resaleApi";

import {
  createResaleConditionGalleryItems,
  sortResaleConditionImages,
} from "../../../shared/presentation/utils/resaleConditionMedia";
import { createProductModelDisplay } from "../../../shared/presentation/utils/productModelDisplay";

import {
  DEFAULT_RESALE_CONDITION,
  DEFAULT_RESALE_EDITABLE_STATUS,
  type ResaleCondition,
  type ResaleConditionImage,
  type ResaleEditableStatus,
  type ResaleListing,
} from "../../../shared/types/resale";
import type { ResaleReviewComment } from "../../../shared/types/resaleReview";

import type {
  ResaleDetailEditFormProps,
  ResaleDetailFooterProps,
  ResaleDetailReadonlyInfoProps,
  ResaleListingTargetSummary,
} from "../types/resaleDetailPageTypes";

import { formatResaleStatus } from "../utils/resaleDetailFormatters";

import {
  formatResalePriceInput,
  normalizeResalePriceInput,
  parseResalePriceInput,
} from "../utils/resalePriceInput";

import { useResaleDetailConditionMedia } from "./useResaleDetailConditionMedia";

const DEFAULT_COMMENT_PAGE = 1;
const DEFAULT_COMMENT_PER_PAGE = 20;

function getErrorMessage(error: unknown, fallbackMessage: string): string {
  if (error instanceof Error && error.message !== "") {
    return error.message;
  }

  return fallbackMessage;
}

export function useResaleDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const normalizedResaleId = resaleId?.trim() ?? "";
  const loadRequestIdRef = useRef(0);

  const [item, setItem] = useState<ResaleListing | null>(null);
  const [images, setImages] = useState<ResaleConditionImage[]>([]);
  const [comments, setComments] = useState<ResaleReviewComment[]>([]);
  const [activeGalleryIndex, setActiveGalleryIndex] = useState(0);

  const [loading, setLoading] = useState(true);
  const [loadingComments, setLoadingComments] = useState(false);
  const [saving, setSaving] = useState(false);
  const [submittingComment, setSubmittingComment] = useState(false);
  const [deletingCommentId, setDeletingCommentId] = useState("");

  const [isEditing, setIsEditing] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [saveMessage, setSaveMessage] = useState("");
  const [commentError, setCommentError] = useState("");

  const [priceInput, setPriceInput] = useState("");
  const [conditionInput, setConditionInput] = useState<ResaleCondition>(
    DEFAULT_RESALE_CONDITION,
  );
  const [descriptionInput, setDescriptionInput] = useState("");
  const [statusInput, setStatusInput] = useState<ResaleEditableStatus>(
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

  const clearMessages = useCallback(() => {
    setErrorMessage("");
    setSaveMessage("");
  }, []);

  const resetFormFromItem = useCallback(
    (nextItem: ResaleListing | null, nextImages: ResaleConditionImage[]) => {
      if (!nextItem) {
        setPriceInput("");
        setConditionInput(DEFAULT_RESALE_CONDITION);
        setDescriptionInput("");
        setStatusInput(DEFAULT_RESALE_EDITABLE_STATUS);
        resetConditionMedia(nextImages);
        return;
      }

      setPriceInput(String(nextItem.price));
      setConditionInput(nextItem.condition);
      setDescriptionInput(nextItem.description);
      setStatusInput(
        nextItem.status === "sold"
          ? DEFAULT_RESALE_EDITABLE_STATUS
          : nextItem.status,
      );
      resetConditionMedia(nextImages);
    },
    [resetConditionMedia],
  );

  const loadDetail = useCallback(async (): Promise<void> => {
    const requestId = ++loadRequestIdRef.current;

    if (!normalizedResaleId) {
      setItem(null);
      setImages([]);
      setComments([]);
      resetFormFromItem(null, []);
      setActiveGalleryIndex(0);
      setIsEditing(false);
      setErrorMessage("出品情報が見つかりません。");
      setSaveMessage("");
      setCommentError("");
      setLoading(false);
      setLoadingComments(false);
      return;
    }

    setLoading(true);
    setLoadingComments(true);
    setErrorMessage("");
    setSaveMessage("");
    setCommentError("");

    const commentsPromise = fetchMyResaleComments({
      resaleId: normalizedResaleId,
      page: DEFAULT_COMMENT_PAGE,
      perPage: DEFAULT_COMMENT_PER_PAGE,
    })
      .then((result) => ({
        result,
        error: null as unknown,
      }))
      .catch((error: unknown) => ({
        result: null,
        error,
      }));

    try {
      const [nextItem, nextImages, commentsResult] = await Promise.all([
        getMyResaleListing(normalizedResaleId),
        listMyResaleConditionImages(normalizedResaleId),
        commentsPromise,
      ]);

      if (requestId !== loadRequestIdRef.current) {
        return;
      }

      const sortedNextImages = sortResaleConditionImages(nextImages);

      setItem(nextItem);
      setImages(sortedNextImages);
      resetFormFromItem(nextItem, sortedNextImages);
      setActiveGalleryIndex(0);
      setIsEditing(false);

      if (commentsResult.result) {
        setComments(commentsResult.result.items);
        setCommentError("");
      } else {
        setComments([]);
        setCommentError(
          getErrorMessage(
            commentsResult.error,
            "コメントの取得に失敗しました。",
          ),
        );
      }
    } catch (error) {
      if (requestId !== loadRequestIdRef.current) {
        return;
      }

      setItem(null);
      setImages([]);
      setComments([]);
      resetFormFromItem(null, []);
      setActiveGalleryIndex(0);
      setIsEditing(false);
      setCommentError("");

      setErrorMessage(
        getErrorMessage(error, "出品情報の取得に失敗しました。"),
      );
    } finally {
      if (requestId === loadRequestIdRef.current) {
        setLoading(false);
        setLoadingComments(false);
      }
    }
  }, [normalizedResaleId, resetFormFromItem]);

  useEffect(() => {
    void loadDetail();

    return () => {
      loadRequestIdRef.current += 1;
    };
  }, [loadDetail]);

  const galleryItems = useMemo(
    () => createResaleConditionGalleryItems(images),
    [images],
  );

  useEffect(() => {
    setActiveGalleryIndex((currentIndex) => {
      if (galleryItems.length === 0) {
        return 0;
      }

      return Math.min(currentIndex, galleryItems.length - 1);
    });
  }, [galleryItems.length]);

  const productName = item?.productName ?? "";
  const tokenName = item?.tokenName ?? "";
  const tokenDescription = item?.tokenDescription ?? "";
  const brandName = item?.brandName ?? "";
  const tokenIconUrl = item?.tokenIcon ?? "";
  const description = item?.description ?? "";

  const model = useMemo(
    () => createProductModelDisplay(item),
    [item],
  );

  const isSold = item?.status === "sold";
  const isListing = item?.status === "listing";
  const title = productName || tokenName || "出品詳細";

  const priceLabel = item
    ? formatYen(item.price, "-")
    : "-";

  const editablePriceLabel = formatResalePriceInput(priceInput);
  const createdAtLabel = formatDateTime(item?.createdAt);
  const updatedAtLabel = formatDateTime(item?.updatedAt);

  const statusLabel = item
    ? formatResaleStatus(item.status)
    : "-";

  const conditionLabel = item
    ? item.condition
    : "-";

  const priceNumber = parseResalePriceInput(priceInput);

  const hasValidPrice =
    priceNumber !== null &&
    priceNumber > 0;

  const canSave =
    isEditing &&
    !isSold &&
    !saving &&
    Boolean(normalizedResaleId) &&
    hasValidPrice &&
    conditionMediaItems.length > 0;

  const canEdit =
    !loading &&
    item !== null &&
    !isEditing &&
    !isSold;

  const canComment =
    !loading &&
    !loadingComments &&
    isListing &&
    !submittingComment;

  const canDeleteComments =
    !loading &&
    !loadingComments &&
    isListing &&
    deletingCommentId === "";

  const handlePrevGalleryItem = useCallback(() => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveGalleryIndex((currentIndex) =>
      currentIndex <= 0
        ? galleryItems.length - 1
        : currentIndex - 1,
    );
  }, [galleryItems.length]);

  const handleNextGalleryItem = useCallback(() => {
    if (galleryItems.length <= 1) {
      return;
    }

    setActiveGalleryIndex((currentIndex) =>
      currentIndex >= galleryItems.length - 1
        ? 0
        : currentIndex + 1,
    );
  }, [galleryItems.length]);

  const handleSelectGalleryItem = useCallback(
    (index: number) => {
      if (index < 0 || index >= galleryItems.length) {
        return;
      }

      setActiveGalleryIndex(index);
    },
    [galleryItems.length],
  );

  const handlePriceChange = useCallback(
    (value: string) => {
      setPriceInput(normalizeResalePriceInput(value));
      clearMessages();
    },
    [clearMessages],
  );

  const handleConditionChange = useCallback(
    (value: ResaleCondition) => {
      setConditionInput(value);
      clearMessages();
    },
    [clearMessages],
  );

  const handleStatusChange = useCallback(
    (value: ResaleEditableStatus) => {
      setStatusInput(value);
      clearMessages();
    },
    [clearMessages],
  );

  const handleDescriptionChange = useCallback(
    (value: string) => {
      setDescriptionInput(value);
      clearMessages();
    },
    [clearMessages],
  );

  const handleCreateComment = useCallback(
    async (body: string): Promise<boolean> => {
      if (!normalizedResaleId) {
        setCommentError("出品情報が見つかりません。");
        return false;
      }

      if (!isListing) {
        setCommentError("現在の出品状態ではコメントできません。");
        return false;
      }

      if (!body || !/\S/u.test(body)) {
        setCommentError("コメントを入力してください。");
        return false;
      }

      if (submittingComment) {
        return false;
      }

      setSubmittingComment(true);
      setCommentError("");

      try {
        const result = await createMyResaleComment({
          resaleId: normalizedResaleId,
          body,
        });

        setComments((current) => [
          result.comment,
          ...current.filter(
            (comment) => comment.commentId !== result.comment.commentId,
          ),
        ]);

        return true;
      } catch (error) {
        setCommentError(
          getErrorMessage(error, "コメントの投稿に失敗しました。"),
        );
        return false;
      } finally {
        setSubmittingComment(false);
      }
    },
    [
      isListing,
      normalizedResaleId,
      submittingComment,
    ],
  );

  const handleDeleteComment = useCallback(
    async (commentId: string): Promise<boolean> => {
      if (!normalizedResaleId) {
        setCommentError("出品情報が見つかりません。");
        return false;
      }

      if (!isListing) {
        setCommentError("現在の出品状態ではコメントを削除できません。");
        return false;
      }

      if (!commentId) {
        setCommentError("コメント情報が見つかりません。");
        return false;
      }

      if (deletingCommentId !== "") {
        return false;
      }

      const confirmed = window.confirm(
        "このコメントを削除します。よろしいですか？",
      );

      if (!confirmed) {
        return false;
      }

      setDeletingCommentId(commentId);
      setCommentError("");

      try {
        await deleteMyResaleComment({
          resaleId: normalizedResaleId,
          commentId,
        });

        setComments((current) =>
          current.filter(
            (comment) => comment.commentId !== commentId,
          ),
        );

        return true;
      } catch (error) {
        setCommentError(
          getErrorMessage(error, "コメントの削除に失敗しました。"),
        );
        return false;
      } finally {
        setDeletingCommentId("");
      }
    },
    [
      deletingCommentId,
      isListing,
      normalizedResaleId,
    ],
  );

  const handleStartEdit = useCallback(() => {
    if (isSold) {
      setErrorMessage("売却済みの出品は編集できません。");
      return;
    }

    resetFormFromItem(item, images);
    setIsEditing(true);
    clearMessages();
  }, [
    clearMessages,
    images,
    isSold,
    item,
    resetFormFromItem,
  ]);

  const handleCancelEdit = useCallback(() => {
    resetFormFromItem(item, images);
    setIsEditing(false);
    clearMessages();
  }, [
    clearMessages,
    images,
    item,
    resetFormFromItem,
  ]);

  const handleSave = useCallback(
    async (): Promise<void> => {
      if (isSold) {
        setErrorMessage("売却済みの出品は編集できません。");
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
        const newFiles = getNewConditionFiles();

        const nextDisplayOrder =
          conditionMediaItems.reduce(
            (currentMax, mediaItem) => {
              if (
                mediaItem.source !== "existing" ||
                !mediaItem.image
              ) {
                return currentMax;
              }

              return Math.max(
                currentMax,
                mediaItem.image.displayOrder,
              );
            },
            -1,
          ) + 1;

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
            startDisplayOrder: nextDisplayOrder,
          });
        }

        const nextImages = sortResaleConditionImages(
          await listMyResaleConditionImages(normalizedResaleId),
        );

        if (nextImages.length > 0) {
          await updatePrimaryResaleImage({
            resaleId: normalizedResaleId,
            imageId: nextImages[0].id,
          });
        }

        const refreshedItem = await getMyResaleListing(
          normalizedResaleId,
        );

        setItem(refreshedItem);
        setImages(nextImages);
        resetFormFromItem(refreshedItem, nextImages);
        setActiveGalleryIndex(0);
        setIsEditing(false);
        setSaveMessage("出品情報を更新しました。");
      } catch (error) {
        setErrorMessage(
          getErrorMessage(error, "出品情報の更新に失敗しました。"),
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

  const handleDelete = useCallback(
    async (): Promise<void> => {
      if (isSold) {
        setErrorMessage("売却済みの出品は削除できません。");
        return;
      }

      if (!normalizedResaleId || saving) {
        return;
      }

      const confirmed = window.confirm(
        "この出品を削除します。よろしいですか？",
      );

      if (!confirmed) {
        return;
      }

      setSaving(true);
      clearMessages();

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
          getErrorMessage(error, "出品情報の削除に失敗しました。"),
        );
      } finally {
        setSaving(false);
      }
    },
    [
      clearMessages,
      isSold,
      navigate,
      normalizedResaleId,
      saving,
    ],
  );

  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleBackToWallet = useCallback(() => {
    navigate("/wallet");
  }, [navigate]);

  const handleReload = useCallback(async (): Promise<void> => {
    await loadDetail();
  }, [loadDetail]);

  const listingTarget = useMemo<ResaleListingTargetSummary>(
    () => ({
      tokenIconUrl,
      tokenName,
      tokenDescription,
      brandName,
      productName,
    }),
    [
      brandName,
      productName,
      tokenDescription,
      tokenIconUrl,
      tokenName,
    ],
  );

  const readonlyInfoProps = useMemo<ResaleDetailReadonlyInfoProps>(
    () => ({
      galleryItems,
      activeGalleryIndex,
      priceLabel,
      conditionLabel,
      statusLabel,
      createdAtLabel,
      updatedAtLabel,
      description,
      onPrevGalleryItem: handlePrevGalleryItem,
      onNextGalleryItem: handleNextGalleryItem,
      onSelectGalleryItem: handleSelectGalleryItem,
    }),
    [
      activeGalleryIndex,
      conditionLabel,
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

  const editFormProps = useMemo<ResaleDetailEditFormProps>(
    () => ({
      priceValue: editablePriceLabel,
      condition: conditionInput,
      status: statusInput,
      description: descriptionInput,
      saving,
      createdAtLabel,
      updatedAtLabel,
      conditionMediaItems,
      conditionMediaCurrentIndex,
      conditionMediaInputRef,
      conditionMediaCarouselRef,
      onPriceChange: handlePriceChange,
      onConditionChange: handleConditionChange,
      onStatusChange: handleStatusChange,
      onDescriptionChange: handleDescriptionChange,
      onConditionMediaSelected: handleConditionMediaSelected,
      onRemoveConditionMedia: handleRemoveConditionMedia,
      onConditionMediaCarouselScroll: handleConditionMediaCarouselScroll,
      onMoveToConditionMediaSlide: handleMoveToConditionMediaSlide,
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

  const footerProps = useMemo<ResaleDetailFooterProps | undefined>(
    () => {
      if (isEditing && !isSold) {
        return {
          variant: "tripleAction",
          leftButtonLabel: "キャンセル",
          centerButtonLabel: saving
            ? "保存中..."
            : "保存する",
          rightButtonLabel: "削除",
          leftButtonDisabled: saving,
          centerButtonDisabled: !canSave,
          rightButtonDisabled: saving,
          onLeftButtonClick: handleCancelEdit,
          onCenterButtonClick: handleSave,
          onRightButtonClick: handleDelete,
        };
      }

      if (canEdit) {
        return {
          variant: "action",
          buttonLabel: "編集する",
          disabled: false,
          onButtonClick: handleStartEdit,
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
    loadingComments,
    item,
    isEditing,
    isSold,
    comments,
    submittingComment,
    deletingCommentId,
    canComment,
    canDeleteComments,
    commentError,
    errorMessage,
    saveMessage,
    listingTarget,
    model,
    readonlyInfoProps,
    editFormProps,
    handleCreateComment,
    handleDeleteComment,
    handleBack,
    handleReload,
    handleBackToWallet,
  };
}