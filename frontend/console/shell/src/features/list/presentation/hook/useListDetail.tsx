// frontend/console/shell/src/features/list/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { usePriceCard } from "./usePriceCard";

import type { PriceRow } from "../../../inventory/application/listCreate/listCreateService";

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";

import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";

import { saveListDetailChanges } from "../../application/listDetail/listDetailSave.usecase";

import { updatePriceRowPrice } from "../../application/listDetail/listDetailMapper";

import {
  isValidListStatus,
  type ListStatus,
} from "../../../../shared/types/list";

import {
  computeListDetailPageTitle,
  deriveListDetail,
  loadListDetailDTO,
  resolveListDetailParams,
  type ListDetailDTO,
  type ListDetailRouteParams,
} from "../../application/listDetailService";

export type DraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;

  loading: boolean;
  error: string;

  saving: boolean;
  saveError: string;

  dto: ListDetailDTO | null;
  reload: () => Promise<void>;

  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;
  onSave: (payload?: any) => Promise<void>;

  listingTitle: string;
  description: string;

  draftListingTitle: string;
  setDraftListingTitle: React.Dispatch<React.SetStateAction<string>>;

  draftDescription: string;
  setDraftDescription: React.Dispatch<React.SetStateAction<string>>;

  status: ListStatus | "";
  draftStatus: ListStatus;
  setDraftStatus: React.Dispatch<React.SetStateAction<ListStatus>>;
  onToggleStatus: (next: ListStatus) => void;

  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  imageUrls: string[];
  draftImages: DraftImage[];

  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (idx: number) => void;
  onClearImages: () => void;

  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  priceRows: PriceRow[];
  draftPriceRows: PriceRow[];
  setDraftPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;

  onChangePrice: (
    index: number,
    price: number | undefined,
    row: PriceRow,
  ) => void;

  priceCard: ReturnType<typeof usePriceCard>;

  assigneeId: string;
  assigneeName: string;

  draftAssigneeId: string;
  setDraftAssigneeId: React.Dispatch<React.SetStateAction<string>>;

  onSelectAssignee: (id: string) => void;
  onChangeAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;

  createdByName: string;
  createdAt: string;

  updatedByName: string;
  updatedAt: string;
};

// ==============================
// Local helpers
// ==============================

function clonePriceRows(rows: PriceRow[]): PriceRow[] {
  return Array.isArray(rows)
    ? rows.map((row) => ({ ...row }))
    : [];
}

function cloneDraftImagesFromUrls(urls: string[]): DraftImage[] {
  return (Array.isArray(urls) ? urls : [])
    .map((url) => String(url ?? "").trim())
    .filter(Boolean)
    .map((url) => ({
      url,
      isNew: false,
    }));
}

function revokeDraftBlobUrls(items: DraftImage[]): void {
  for (const item of Array.isArray(items) ? items : []) {
    if (
      item.isNew &&
      typeof item.url === "string" &&
      item.url.startsWith("blob:")
    ) {
      try {
        URL.revokeObjectURL(item.url);
      } catch {
        // Blob URLの解放失敗は無視する。
      }
    }
  }
}

// ==============================
// List image draft hook
// ==============================

function fileKey(file: File): string {
  return `${file.name}__${file.size}__${file.lastModified}`;
}

function isImageFile(file: File): boolean {
  return file.type.startsWith("image/");
}

function useListImages(args: {
  isEdit: boolean;
  saving: boolean;
  initialUrls: string[];
}) {
  const {
    isEdit,
    saving,
    initialUrls,
  } = args;

  const [
    draftImages,
    setDraftImages,
  ] = React.useState<DraftImage[]>(
    cloneDraftImagesFromUrls(initialUrls),
  );

  React.useEffect(() => {
    if (isEdit) {
      return;
    }

    setDraftImages(
      cloneDraftImagesFromUrls(initialUrls),
    );
  }, [
    isEdit,
    initialUrls,
  ]);

  const addFiles = React.useCallback(
    (files: File[]) => {
      if (!isEdit || saving) {
        return;
      }

      const incoming = (
        Array.isArray(files)
          ? files
          : []
      )
        .filter(Boolean)
        .filter(isImageFile);

      if (incoming.length === 0) {
        return;
      }

      setDraftImages((previousImages) => {
        const currentImages =
          Array.isArray(previousImages)
            ? previousImages
            : [];

        const existingFileKeys = new Set(
          currentImages
            .filter(
              (item) =>
                item.isNew &&
                item.file,
            )
            .map(
              (item) =>
                fileKey(item.file as File),
            ),
        );

        const newImages: DraftImage[] = [];

        for (const file of incoming) {
          const key = fileKey(file);

          if (existingFileKeys.has(key)) {
            continue;
          }

          existingFileKeys.add(key);

          newImages.push({
            url: URL.createObjectURL(file),
            file,
            isNew: true,
          });
        }

        return [
          ...currentImages,
          ...newImages,
        ];
      });
    },
    [
      isEdit,
      saving,
    ],
  );

  const onAddImages = React.useCallback(
    (files: FileList | null) => {
      if (!files || files.length === 0) {
        return;
      }

      addFiles(
        Array.from(files).filter(Boolean),
      );
    },
    [addFiles],
  );

  const onRemoveImageAt = React.useCallback(
    (index: number) => {
      if (!isEdit || saving) {
        return;
      }

      setDraftImages((previousImages) => {
        const currentImages =
          Array.isArray(previousImages)
            ? previousImages
            : [];

        if (
          index < 0 ||
          index >= currentImages.length
        ) {
          return currentImages;
        }

        const target =
          currentImages[index];

        if (
          target.isNew &&
          target.url.startsWith("blob:")
        ) {
          try {
            URL.revokeObjectURL(target.url);
          } catch {
            // Blob URLの解放失敗は無視する。
          }
        }

        return currentImages
          .slice(0, index)
          .concat(
            currentImages.slice(index + 1),
          );
      });
    },
    [
      isEdit,
      saving,
    ],
  );

  const onClearImages = React.useCallback(() => {
    if (!isEdit || saving) {
      return;
    }

    setDraftImages((previousImages) => {
      const currentImages =
        Array.isArray(previousImages)
          ? previousImages
          : [];

      revokeDraftBlobUrls(currentImages);

      return [];
    });
  }, [
    isEdit,
    saving,
  ]);

  const imageUrls = React.useMemo(
    () =>
      (
        Array.isArray(draftImages)
          ? draftImages
          : []
      )
        .map(
          (item) =>
            String(item.url ?? "").trim(),
        )
        .filter(Boolean),
    [draftImages],
  );

  return {
    draftImages,
    setDraftImages,
    imageUrls,
    onAddImages,
    onRemoveImageAt,
    onClearImages,
  };
}

// ==============================
// Hook
// ==============================

export function useListDetail(): UseListDetailResult {
  const navigate = useNavigate();

  const params =
    useParams<ListDetailRouteParams>();

  const resolved = React.useMemo(
    () =>
      resolveListDetailParams(params),
    [params],
  );

  const {
    listId,
    inventoryId,
  } = resolved;

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // -----------------------------
  // Load DTO
  // -----------------------------

  const [
    dto,
    setDTO,
  ] = React.useState<ListDetailDTO | null>(
    null,
  );

  const [
    loading,
    setLoading,
  ] = React.useState(false);

  const [
    error,
    setError,
  ] = React.useState("");

  const cancelledRef =
    useCancelledRef();

  const reload = React.useCallback(
    async () => {
      const id =
        String(listId ?? "").trim();

      if (!id) {
        setDTO(null);
        setError(
          "listId がありません（ルートパラメータを確認してください）。",
        );
        return;
      }

      setLoading(true);
      setError("");

      try {
        const data =
          await loadListDetailDTO({
            listId: id,
            inventoryIdHint:
              inventoryId,
          });

        if (cancelledRef.current) {
          return;
        }

        setDTO(data);
      } catch (caughtError) {
        if (cancelledRef.current) {
          return;
        }

        setError(
          String(
            caughtError instanceof Error
              ? caughtError.message
              : caughtError,
          ),
        );

        setDTO(null);
      } finally {
        if (cancelledRef.current) {
          return;
        }

        setLoading(false);
      }
    },
    [
      listId,
      inventoryId,
      cancelledRef,
    ],
  );

  React.useEffect(() => {
    void reload();
  }, [reload]);

  // -----------------------------
  // Derived view fields
  // -----------------------------

  const derived = React.useMemo(
    () =>
      deriveListDetail<PriceRow>(dto),
    [dto],
  );

  const {
    listingTitle,
    description,
    status,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls: viewImageUrls,
    priceRows: viewPriceRows,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,

    updatedByName,
    updatedAt,
  } = derived;

  const statusForEdit =
    React.useMemo<ListStatus>(
      () =>
        status === "listing"
          ? "listing"
          : "suspended",
      [status],
    );

  // -----------------------------
  // Edit state
  // -----------------------------

  const [
    isEdit,
    setIsEdit,
  ] = React.useState(false);

  const [
    draftListingTitle,
    setDraftListingTitle,
  ] = React.useState(listingTitle);

  const [
    draftDescription,
    setDraftDescription,
  ] = React.useState(description);

  const [
    draftPriceRows,
    setDraftPriceRows,
  ] = React.useState<PriceRow[]>(
    clonePriceRows(viewPriceRows),
  );

  const [
    draftStatus,
    setDraftStatus,
  ] = React.useState<ListStatus>(
    statusForEdit,
  );

  const [
    draftAssigneeId,
    setDraftAssigneeId,
  ] = React.useState(assigneeId);

  const [
    saving,
    setSaving,
  ] = React.useState(false);

  const [
    saveError,
    setSaveError,
  ] = React.useState("");

  const img = useListImages({
    isEdit,
    saving,
    initialUrls:
      viewImageUrls,
  });

  React.useEffect(() => {
    if (isEdit) {
      return;
    }

    setDraftListingTitle(
      listingTitle,
    );

    setDraftDescription(
      description,
    );

    setDraftPriceRows(
      clonePriceRows(viewPriceRows),
    );

    setDraftStatus(
      statusForEdit,
    );

    setDraftAssigneeId(
      assigneeId,
    );

    img.setDraftImages(
      cloneDraftImagesFromUrls(
        viewImageUrls,
      ),
    );
  }, [
    isEdit,
    listingTitle,
    description,
    viewPriceRows,
    statusForEdit,
    assigneeId,
    viewImageUrls,
    img.setDraftImages,
  ]);

  const onEdit = React.useCallback(() => {
    setDraftListingTitle(
      listingTitle,
    );

    setDraftDescription(
      description,
    );

    setDraftPriceRows(
      clonePriceRows(viewPriceRows),
    );

    setDraftStatus(
      statusForEdit,
    );

    setDraftAssigneeId(
      assigneeId,
    );

    img.setDraftImages(
      cloneDraftImagesFromUrls(
        viewImageUrls,
      ),
    );

    setSaveError("");
    setIsEdit(true);
  }, [
    listingTitle,
    description,
    viewPriceRows,
    statusForEdit,
    assigneeId,
    viewImageUrls,
    img.setDraftImages,
  ]);

  const onCancel = React.useCallback(() => {
    revokeDraftBlobUrls(
      img.draftImages,
    );

    setDraftListingTitle(
      listingTitle,
    );

    setDraftDescription(
      description,
    );

    setDraftPriceRows(
      clonePriceRows(viewPriceRows),
    );

    setDraftStatus(
      statusForEdit,
    );

    setDraftAssigneeId(
      assigneeId,
    );

    img.setDraftImages(
      cloneDraftImagesFromUrls(
        viewImageUrls,
      ),
    );

    setSaveError("");
    setIsEdit(false);
  }, [
    img.draftImages,
    img.setDraftImages,
    listingTitle,
    description,
    viewPriceRows,
    statusForEdit,
    assigneeId,
    viewImageUrls,
  ]);

  const onToggleStatus =
    React.useCallback(
      (next: ListStatus) => {
        if (!isEdit || saving) {
          return;
        }

        setDraftStatus(next);
      },
      [
        isEdit,
        saving,
      ],
    );

  const onSelectAssignee =
    React.useCallback(
      (id: string) => {
        if (!isEdit || saving) {
          return;
        }

        setDraftAssigneeId(
          String(id ?? "").trim(),
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const onChangeAssignee =
    React.useCallback(
      (id: string) => {
        if (!isEdit || saving) {
          return;
        }

        setDraftAssigneeId(
          String(id ?? "").trim(),
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const onEditAssignee =
    React.useCallback(() => {
      // ListDetail全体の編集モードで制御する。
    }, []);

  const onClickAssignee =
    React.useCallback(() => {
      // 遷移先またはモーダル確定後に処理を追加する。
    }, []);

  const effectiveImageUrls =
    React.useMemo(
      () =>
        isEdit
          ? img.imageUrls
          : viewImageUrls,
      [
        isEdit,
        img.imageUrls,
        viewImageUrls,
      ],
    );

  const [
    mainImageIndex,
    setMainImageIndex,
  ] = React.useState(0);

  useMainImageIndexGuard({
    imageUrls:
      effectiveImageUrls,
    mainImageIndex,
    setMainImageIndex,
  });

  // -----------------------------
  // Price
  // -----------------------------

  const onChangePrice =
    React.useCallback(
      (
        index: number,
        price: number | undefined,
        _row: PriceRow,
      ) => {
        if (!isEdit) {
          return;
        }

        setDraftPriceRows(
          (previousRows) =>
            updatePriceRowPrice(
              previousRows,
              index,
              price,
            ),
        );
      },
      [isEdit],
    );

  // -----------------------------
  // Save
  // -----------------------------

  const onSave = React.useCallback(
    async (payload?: any) => {
      const id =
        String(listId ?? "").trim();

      if (!id) {
        setSaveError(
          "invalid_list_id",
        );
        return;
      }

      const nextTitle =
        String(
          payload?.title ?? "",
        ).trim() ||
        String(
          payload?.listingTitle ?? "",
        ).trim() ||
        String(
          draftListingTitle ?? "",
        ).trim();

      const nextDescription =
        payload &&
        payload.description !== undefined
          ? String(
              payload.description ?? "",
            )
          : String(
              draftDescription ?? "",
            );

      const payloadStatus =
        String(
          payload?.status ?? "",
        ).trim();

      const nextStatus =
        isValidListStatus(payloadStatus)
          ? payloadStatus
          : draftStatus;

      const uid =
        String(
          auth.currentUser?.uid ?? "",
        ).trim() ||
        "system";

      setSaving(true);
      setSaveError("");

      try {
        const result =
          await saveListDetailChanges({
            listId: id,
            inventoryIdHint:
              inventoryId,
            currentDTO:
              dto,

            title:
              nextTitle,
            description:
              nextDescription,
            status:
              nextStatus,

            assigneeId:
              String(
                payload?.assigneeId ?? "",
              ).trim() ||
              String(
                draftAssigneeId ?? "",
              ).trim() ||
              String(
                dto?.assigneeId ?? "",
              ).trim() ||
              undefined,

            updatedBy:
              uid,

            draftPriceRows:
              Array.isArray(
                draftPriceRows,
              )
                ? draftPriceRows
                : [],

            draftImages:
              Array.isArray(
                img.draftImages,
              )
                ? img.draftImages
                : [],

            mainImageIndex,
          });

        if (cancelledRef.current) {
          return;
        }

        revokeDraftBlobUrls(
          img.draftImages,
        );

        setDTO(result.dto);
        setIsEdit(false);
      } catch (caughtError) {
        if (cancelledRef.current) {
          return;
        }

        setSaveError(
          String(
            caughtError instanceof Error
              ? caughtError.message
              : caughtError,
          ),
        );
      } finally {
        if (cancelledRef.current) {
          return;
        }

        setSaving(false);
      }
    },
    [
      listId,
      inventoryId,
      dto,
      draftStatus,
      draftListingTitle,
      draftDescription,
      draftAssigneeId,
      draftPriceRows,
      img.draftImages,
      mainImageIndex,
      cancelledRef,
    ],
  );

  const effectiveForPriceCard =
    isEdit
      ? draftPriceRows
      : viewPriceRows;

  const priceCard = usePriceCard({
    title: "価格",
    rows: effectiveForPriceCard,
    mode:
      isEdit
        ? "edit"
        : "view",
    currencySymbol: "¥",
    onChangePrice:
      isEdit
        ? onChangePrice
        : undefined,
  });

  const pageTitle =
    React.useMemo(
      () =>
        computeListDetailPageTitle({
          listId,
          listingTitle,
        }),
      [
        listId,
        listingTitle,
      ],
    );

  return {
    pageTitle,
    onBack,

    loading,
    error,

    saving,
    saveError,

    dto,
    reload,

    isEdit,
    onEdit,
    onCancel,
    onSave,

    listingTitle,
    description,

    draftListingTitle,
    setDraftListingTitle,

    draftDescription,
    setDraftDescription,

    status,
    draftStatus,
    setDraftStatus,
    onToggleStatus,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls:
      effectiveImageUrls,

    draftImages:
      img.draftImages,

    onAddImages:
      img.onAddImages,

    onRemoveImageAt:
      img.onRemoveImageAt,

    onClearImages:
      img.onClearImages,

    mainImageIndex,
    setMainImageIndex,

    priceRows:
      viewPriceRows,

    draftPriceRows,
    setDraftPriceRows,
    onChangePrice,

    priceCard,

    assigneeId,
    assigneeName,

    draftAssigneeId,
    setDraftAssigneeId,

    onSelectAssignee,
    onChangeAssignee,
    onEditAssignee,
    onClickAssignee,

    createdByName,
    createdAt,

    updatedByName,
    updatedAt,
  };
}