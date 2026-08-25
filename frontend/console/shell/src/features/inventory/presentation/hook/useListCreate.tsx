// frontend/console/shell/src/features/inventory/presentation/hook/useListCreate.tsx

import * as React from "react";
import { useNavigate, useParams, type NavigateFunction } from "react-router-dom";

import { usePriceCard } from "../../../list/presentation/hook/usePriceCard";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import type { AssigneeCandidate } from "../../../admin/application/AdminService";
import type {
  ListStatus,
} from "../../../../shared/types/list";
import type { ListCreateDTO } from "../../../../shared/types/inventory";

import {
  createCompletedListProgress,
  createFailedListProgress,
  createInitialListProgress,
  createPreparingListProgress,
  createSavingListProgress,
  createUploadingListProgress,
  isListProgressVisible,
  type ListProgress,
} from "../../../list/presentation/modal/listProgress";

import {
  buildBackPath,
  canFetchListCreate,
  createListWithImages,
  extractDisplayStrings,
  loadListCreateDTOFromParams,
  resolveListCreateParams,
  type ListCreateRouteParams,
  type PriceRow,
  type ResolvedListCreateParams,
} from "../../application/listCreateService";

type ImageInputRef = React.RefObject<HTMLInputElement | null>;

export type UseListCreateResult = {
  title: string;
  onBack: () => void;
  onCreate: () => void;

  saving: boolean;
  isUploading: boolean;

  progress: ListProgress;
  progressOpen: boolean;
  onCloseProgress: () => void;

  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;

  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;

  priceRows: PriceRow[];
  onChangePrice: (index: number, price: number | undefined) => void;
  priceCard: ReturnType<typeof usePriceCard>;

  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;

  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef;
  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (index: number) => void;
  onClearImages: () => void;

  assigneeName: string;
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;
  handleSelectAssignee: (id: string) => void;

  status: ListStatus;
  setStatus: React.Dispatch<React.SetStateAction<ListStatus>>;
};

type UsePriceRowsResult = {
  priceRows: PriceRow[];
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  onChangePrice: (index: number, price: number | undefined) => void;
  priceCard: ReturnType<typeof usePriceCard>;
};

function errorMessageFromUnknown(
  error: unknown,
): string {
  return String(
    error instanceof Error
      ? error.message
      : error,
  );
}

function dedupeFiles(previousFiles: File[], addedFiles: File[]): File[] {
  const existingKeys = new Set(
    previousFiles.map(
      (file) => `${file.name}__${file.size}__${file.lastModified}`,
    ),
  );

  const filteredFiles = addedFiles.filter(
    (file) =>
      !existingKeys.has(
        `${file.name}__${file.size}__${file.lastModified}`,
      ),
  );

  return [...previousFiles, ...filteredFiles];
}

function filterImageFiles(
  files: FileList | File[] | null | undefined,
): File[] {
  return Array.from(files ?? [])
    .filter(Boolean)
    .filter((file) => String(file.type || "").startsWith("image/")) as File[];
}

function useListCreateParamsAndTitle(): {
  resolvedParams: ResolvedListCreateParams;
  title: string;
} {
  const params = useParams<ListCreateRouteParams>();

  const resolvedParams = React.useMemo(
    () => resolveListCreateParams(params),
    [params],
  );

  return {
    resolvedParams,
    title: "出品作成",
  };
}

function useListingStatus(): {
  status: ListStatus;
  setStatus: React.Dispatch<React.SetStateAction<ListStatus>>;
} {
  const [status, setStatus] = React.useState<ListStatus>("listing");

  return {
    status,
    setStatus,
  };
}

function useListingFields(): {
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;
} {
  const [listingTitle, setListingTitle] = React.useState("");
  const [description, setDescription] = React.useState("");

  return {
    listingTitle,
    setListingTitle,
    description,
    setDescription,
  };
}

function useListingImages(): {
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef;
  onSelectImages: (files: FileList | null) => void;
  removeImageAt: (index: number) => void;
  clearImages: () => void;
} {
  const [images, setImages] = React.useState<File[]>([]);
  const [mainImageIndex, setMainImageIndex] = React.useState(0);
  const [imagePreviewUrls, setImagePreviewUrls] = React.useState<string[]>([]);

  const imageInputRef = React.useRef<HTMLInputElement | null>(null);

  const appendImages = React.useCallback(
    (filesLike: FileList | File[] | null) => {
      const files = filterImageFiles(filesLike);

      if (files.length === 0) {
        return;
      }

      setImages((previousFiles) =>
        dedupeFiles(previousFiles, files),
      );
    },
    [],
  );

  const onSelectImages = React.useCallback(
    (files: FileList | null) => {
      appendImages(files);
    },
    [appendImages],
  );

  const removeImageAt = React.useCallback((index: number) => {
    setImages((previousFiles) =>
      previousFiles.filter((_, previousIndex) => previousIndex !== index),
    );

    setMainImageIndex((previousMainIndex) => {
      if (index === previousMainIndex) {
        return 0;
      }

      if (index < previousMainIndex) {
        return Math.max(0, previousMainIndex - 1);
      }

      return previousMainIndex;
    });
  }, []);

  const clearImages = React.useCallback(() => {
    setImages([]);
    setMainImageIndex(0);
  }, []);

  React.useEffect(() => {
    if (images.length === 0) {
      setImagePreviewUrls([]);
      return;
    }

    const urls = images.map((file) => URL.createObjectURL(file));
    setImagePreviewUrls(urls);

    return () => {
      urls.forEach((url) => {
        URL.revokeObjectURL(url);
      });
    };
  }, [images]);

  React.useEffect(() => {
    if (images.length === 0) {
      if (mainImageIndex !== 0) {
        setMainImageIndex(0);
      }
      return;
    }

    if (
      mainImageIndex < 0 ||
      mainImageIndex >= images.length
    ) {
      setMainImageIndex(0);
    }
  }, [images.length, mainImageIndex]);

  return {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    removeImageAt,
    clearImages,
  };
}

function usePriceRows(): UsePriceRowsResult {
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);
  const initializedPriceRowsRef = React.useRef(false);

  const onChangePrice = React.useCallback(
    (index: number, price: number | undefined) => {
      setPriceRows((previousRows) => {
        const currentRow = previousRows[index];

        if (!currentRow) {
          return previousRows;
        }

        const nextRows = [...previousRows];

        nextRows[index] = {
          ...currentRow,
          price,
        };

        return nextRows;
      });
    },
    [],
  );

  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows,
    mode: "edit",
    currencySymbol: "¥",
    onChangePrice,
  });

  return {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  };
}

function useListCreateNavigation(
  resolvedParams: ResolvedListCreateParams,
  isUploading: boolean,
): {
  navigate: NavigateFunction;
  onBack: () => void;
} {
  const navigate = useNavigate();

  const onBack = React.useCallback(() => {
    if (isUploading) {
      return;
    }

    navigate(buildBackPath(resolvedParams));
  }, [
    navigate,
    resolvedParams,
    isUploading,
  ]);

  return {
    navigate,
    onBack,
  };
}

function useListCreateDTO(
  args: {
    resolvedParams: ResolvedListCreateParams;
    initializedPriceRowsRef: React.MutableRefObject<boolean>;
    setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  },
): {
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  const {
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  } = args;

  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState("");

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      if (!canFetchListCreate(resolvedParams)) {
        return;
      }

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await loadListCreateDTOFromParams(resolvedParams);

        if (cancelled) {
          return;
        }

        setDTO(data);

        if (!initializedPriceRowsRef.current) {
          setPriceRows(data.priceRows);
          initializedPriceRowsRef.current = true;
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        setDTOError(
          String(
            error instanceof Error
              ? error.message
              : error,
          ),
        );
      } finally {
        if (!cancelled) {
          setLoadingDTO(false);
        }
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  ]);

  const {
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  } = React.useMemo(
    () => extractDisplayStrings(dto),
    [dto],
  );

  return {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  };
}

function useCreateList(
  args: {
    resolvedParams: ResolvedListCreateParams;
    status: ListStatus;
    listingTitle: string;
    description: string;
    priceRows: PriceRow[];
    assigneeId: string | undefined;
    images: File[];
    mainImageIndex: number;
    saving: boolean;
    setSaving: React.Dispatch<React.SetStateAction<boolean>>;
    setProgress: React.Dispatch<React.SetStateAction<ListProgress>>;
    setCreatedListId: React.Dispatch<React.SetStateAction<string>>;
  },
): {
  onCreate: () => Promise<void>;
} {
  const {
    resolvedParams,
    status,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
    saving,
    setSaving,
    setProgress,
    setCreatedListId,
  } = args;

  const onCreate = React.useCallback(async () => {
    if (saving) {
      return;
    }

    let imageUploadFailedMessage = "";

    let transferredBytes = 0;
    let totalBytes = 0;
    let completedUploadCount = 0;
    let expectedUploadCount = 0;

    setSaving(true);
    setCreatedListId("");

    try {
      const inventoryId = String(
        resolvedParams.inventoryId ?? "",
      );

      const created = await createListWithImages({
        params: {
          ...resolvedParams,
          inventoryId,
        },
        listingTitle,
        description,
        priceRows,
        status,
        assigneeId,
        images,
        mainImageIndex,

        progressHandlers: {
          onPreparing: () => {
            setProgress(
              createPreparingListProgress({
                title: "作成準備中",
                message: "出品情報の作成準備をしています。",
              }),
            );
          },

          onImageProgress: (imageProgress) => {
            transferredBytes =
              imageProgress.transferredBytes;

            totalBytes =
              imageProgress.totalBytes;

            completedUploadCount =
              imageProgress.completedUploadCount;

            expectedUploadCount =
              imageProgress.expectedUploadCount;

            setProgress(
              createUploadingListProgress({
                fileName:
                  imageProgress.fileName,

                transferredBytes:
                  imageProgress.transferredBytes,

                totalBytes:
                  imageProgress.totalBytes,

                completedUploadCount:
                  imageProgress.completedUploadCount,

                expectedUploadCount:
                  imageProgress.expectedUploadCount,

                title:
                  "画像を転送中",

                message:
                  "画像転送が完了するまで、この画面を閉じたり移動したりしないでください。",
              }),
            );
          },

          onSaving: (savingProgress) => {
            transferredBytes =
              savingProgress.transferredBytes;

            totalBytes =
              savingProgress.totalBytes;

            completedUploadCount =
              savingProgress.completedUploadCount;

            expectedUploadCount =
              savingProgress.expectedUploadCount;

            setProgress(
              createSavingListProgress({
                transferredBytes:
                  savingProgress.transferredBytes,

                totalBytes:
                  savingProgress.totalBytes,

                completedUploadCount:
                  savingProgress.completedUploadCount,

                expectedUploadCount:
                  savingProgress.expectedUploadCount,

                title:
                  "出品を保存中",

                message:
                  "画像転送が完了しました。出品情報を保存しています。",
              }),
            );
          },
        },

        onImageUploadFailed: (message) => {
          imageUploadFailedMessage = message;
        },
      });

      if (!created.id) {
        throw new Error("created_list_missing_id");
      }

      setCreatedListId(
        created.id,
      );

      if (imageUploadFailedMessage) {
        setProgress(
          createCompletedListProgress({
            title:
              "出品を作成しました",

            message:
              imageUploadFailedMessage,
          }),
        );

        return;
      }

      setProgress(
        createCompletedListProgress({
          transferredBytes,

          totalBytes,

          completedUploadCount,

          expectedUploadCount,

          title:
            "作成が完了しました",

          message:
            "出品の作成が完了しました。",
        }),
      );
    } catch (error) {
      const message =
        errorMessageFromUnknown(
          error,
        );

      setProgress(
        createFailedListProgress(
          message,
          {
            title:
              "作成に失敗しました",

            message:
              "出品の作成中にエラーが発生しました。",
          },
        ),
      );
    } finally {
      setSaving(false);
    }
  }, [
    resolvedParams,
    status,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
    saving,
    setSaving,
    setProgress,
    setCreatedListId,
  ]);

  return {
    onCreate,
  };
}

export function useListCreate(): UseListCreateResult {
  const {
    resolvedParams,
    title,
  } = useListCreateParamsAndTitle();

  const {
    status,
    setStatus,
  } = useListingStatus();

  const {
    listingTitle,
    setListingTitle,
    description,
    setDescription,
  } = useListingFields();

  const {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    removeImageAt,
    clearImages,
  } = useListingImages();

  const {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  } = usePriceRows();

  const [saving, setSaving] = React.useState(false);

  const [
    progress,
    setProgress,
  ] = React.useState<ListProgress>(
    createInitialListProgress,
  );

  const [
    createdListId,
    setCreatedListId,
  ] = React.useState("");

  const isUploading =
    progress.phase === "uploading" &&
    saving;

  const progressOpen =
    isListProgressVisible(
      progress,
    );

  const {
    navigate,
    onBack,
  } = useListCreateNavigation(
    resolvedParams,
    isUploading,
  );

  React.useEffect(() => {
    if (!isUploading) {
      return;
    }

    const handleBeforeUnload = (
      event: BeforeUnloadEvent,
    ) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener(
      "beforeunload",
      handleBeforeUnload,
    );

    return () => {
      window.removeEventListener(
        "beforeunload",
        handleBeforeUnload,
      );
    };
  }, [
    isUploading,
  ]);

  const onCloseProgress = React.useCallback(() => {
    if (isUploading) {
      return;
    }

    if (
      progress.phase === "completed" &&
      createdListId
    ) {
      const listId = createdListId;

      setProgress(
        createInitialListProgress(),
      );

      setCreatedListId("");

      navigate(
        `/list/${encodeURIComponent(listId)}`,
      );

      return;
    }

    setProgress(
      createInitialListProgress(),
    );
  }, [
    isUploading,
    progress.phase,
    createdListId,
    navigate,
  ]);

  const {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({
    defaultToCurrentMember: true,
  });

  const {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  } = useListCreateDTO({
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  });

  const {
    onCreate,
  } = useCreateList({
    resolvedParams,
    status,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
    saving,
    setSaving,
    setProgress,
    setCreatedListId,
  });

  return {
    title,
    onBack,
    onCreate,

    saving,
    isUploading,

    progress,
    progressOpen,
    onCloseProgress,

    dto,
    loadingDTO,
    dtoError,

    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    priceRows,
    onChangePrice,
    priceCard,

    listingTitle,
    setListingTitle,
    description,
    setDescription,

    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,

    onAddImages: onSelectImages,
    onRemoveImageAt: removeImageAt,
    onClearImages: clearImages,

    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,

    status,
    setStatus,
  };
}