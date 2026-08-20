// frontend/console/shell/src/features/list/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { PriceRow } from "../../../inventory/application/listCreateService";
import { useAuthContext } from "../../../../auth/application/AuthContext";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";
import { useListImages } from "./useListImages";
import { saveListDetailChanges } from "../../application/listDetail/listDetailSave.usecase";
import { updatePriceRowPrice } from "../../application/listDetail/listDetailMapper";
import { buildListDetailSaveInput } from "../../application/listDetail/buildListDetailSaveInput";
import type { ListDetailSavePayload } from "../../application/listDetail/listDetailSavePayload";
import type { ListStatus } from "../../../../shared/types/list";
import {
  deleteListDetail,
  deriveListDetail,
  loadListDetailDTO,
  resolveListDetailParams,
  type ListDetailDTO,
  type ListDetailRouteParams,
} from "../../application/listDetailService";

export type { DraftImage } from "./useListImages";

export type UseListDetailResult = {
  loading: boolean;
  error: string;
  saving: boolean;
  saveError: string;
  deleting: boolean;
  deleteError: string;
  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;
  onDelete: () => Promise<void>;
  onSave: (payload?: ListDetailSavePayload) => Promise<void>;
  listingTitle: string;
  description: string;
  draftListingTitle: string;
  setDraftListingTitle: React.Dispatch<React.SetStateAction<string>>;
  draftDescription: string;
  setDraftDescription: React.Dispatch<React.SetStateAction<string>>;
  status: ListStatus | "";
  draftStatus: ListStatus;
  onToggleStatus: (next: ListStatus) => void;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
  imageUrls: string[];
  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (index: number) => void;
  onClearImages: () => void;
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  priceRows: PriceRow[];
  draftPriceRows: PriceRow[];
  onChangePrice: (
    index: number,
    price: number | undefined,
    row: PriceRow,
  ) => void;
  assigneeId: string;
  assigneeName: string;
  draftAssigneeId: string;
  draftAssigneeName: string;
  assigneeCandidates: {
    id: string;
    name: string;
  }[];
  loadingMembers: boolean;
  onSelectAssignee: (id: string) => void;
  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;
};

function clonePriceRows(rows: readonly PriceRow[]): PriceRow[] {
  return rows.map((row) => ({ ...row }));
}

export function useListDetail(): UseListDetailResult {
  const params = useParams<ListDetailRouteParams>();
  const navigate = useNavigate();
  const { user } = useAuthContext();

  const resolved = React.useMemo(
    () => resolveListDetailParams(params),
    [params],
  );

  const { listId } = resolved;

  const [dto, setDTO] = React.useState<ListDetailDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");
  const cancelledRef = useCancelledRef();

  const reload = React.useCallback(async () => {
    const id = String(listId ?? "").trim();

    if (!id) {
      setDTO(null);
      setError("listId がありません（ルートパラメータを確認してください）。");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const data = await loadListDetailDTO({ listId: id });

      if (cancelledRef.current) return;
      setDTO(data);
    } catch (caughtError) {
      if (cancelledRef.current) return;

      setError(
        String(
          caughtError instanceof Error
            ? caughtError.message
            : caughtError,
        ),
      );
      setDTO(null);
    } finally {
      if (cancelledRef.current) return;
      setLoading(false);
    }
  }, [listId, cancelledRef]);

  React.useEffect(() => {
    void reload();
  }, [reload]);

  const derived = React.useMemo(
    () => (dto ? deriveListDetail(dto) : null),
    [dto],
  );

  const listingTitle = derived?.title ?? "";
  const description = derived?.description ?? "";
  const status = derived?.status ?? "";
  const productBrandName = derived?.productBrandName ?? "";
  const productName = derived?.productName ?? "";
  const tokenBrandName = derived?.tokenBrandName ?? "";
  const tokenName = derived?.tokenName ?? "";
  const assigneeId = derived?.assigneeId ?? "";
  const assigneeName = derived?.assigneeName ?? "";
  const createdByName = derived?.createdByName ?? "";
  const createdAt = derived?.createdAtLabel ?? "";
  const updatedByName = derived?.updatedByName ?? "";
  const updatedAt = derived?.updatedAtLabel ?? "";
  const primaryImageId = derived?.primaryImageId;

  const {
    assigneeId: draftAssigneeId,
    assigneeName: draftAssigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
    resetAssignee,
  } = useAssigneeSelection({
    initialAssigneeId: assigneeId || null,
    initialAssigneeName: assigneeName || null,
    defaultToCurrentMember: false,
  });

  const viewImages = React.useMemo(
    () => derived?.images ?? [],
    [derived],
  );

  const viewImageUrls = React.useMemo(
    () => viewImages.map((image) => image.url),
    [viewImages],
  );

  const viewPriceRows = React.useMemo<PriceRow[]>(
    () => derived?.priceRows ?? [],
    [derived],
  );

  const viewPrimaryImageIndex = React.useMemo(() => {
    if (!primaryImageId) return 0;

    const index = viewImages.findIndex(
      (image) => image.id === primaryImageId,
    );

    return index >= 0 ? index : 0;
  }, [primaryImageId, viewImages]);

  const [isEdit, setIsEdit] = React.useState(false);
  const [draftListingTitle, setDraftListingTitle] = React.useState(listingTitle);
  const [draftDescription, setDraftDescription] = React.useState(description);
  const [draftPriceRows, setDraftPriceRows] = React.useState<PriceRow[]>(
    clonePriceRows(viewPriceRows),
  );
  const [draftStatus, setDraftStatus] = React.useState<ListStatus>("suspended");
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState("");
  const [deleting, setDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState("");
  const [mainImageIndex, setMainImageIndex] = React.useState(viewPrimaryImageIndex);

  const images = useListImages({
    isEdit,
    saving,
    initialImages: viewImages,
  });

  const resetDraftFromView = React.useCallback(() => {
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));

    if (status) {
      setDraftStatus(status);
    }

    resetAssignee();
  }, [
    listingTitle,
    description,
    viewPriceRows,
    status,
    resetAssignee,
  ]);

  React.useEffect(() => {
    if (isEdit) return;
    resetDraftFromView();
  }, [isEdit, resetDraftFromView]);

  React.useEffect(() => {
    if (isEdit) return;
    setMainImageIndex(viewPrimaryImageIndex);
  }, [isEdit, viewPrimaryImageIndex]);

  const onEdit = React.useCallback(() => {
    if (deleting) return;

    resetDraftFromView();
    setMainImageIndex(viewPrimaryImageIndex);
    setSaveError("");
    setDeleteError("");
    setIsEdit(true);
  }, [
    deleting,
    resetDraftFromView,
    viewPrimaryImageIndex,
  ]);

  const onCancel = React.useCallback(() => {
    if (saving || deleting) return;

    images.releaseDraftBlobUrls();
    resetDraftFromView();
    setMainImageIndex(viewPrimaryImageIndex);
    setSaveError("");
    setDeleteError("");
    setIsEdit(false);
  }, [
    saving,
    deleting,
    images.releaseDraftBlobUrls,
    resetDraftFromView,
    viewPrimaryImageIndex,
  ]);

  const onDelete = React.useCallback(async () => {
    const id = String(listId ?? "").trim();

    if (!id) {
      setDeleteError("invalid_list_id");
      return;
    }

    if (saving || deleting) return;

    const confirmed = window.confirm(
      "この出品を削除しますか？削除後は元に戻せません。",
    );

    if (!confirmed) return;

    setDeleting(true);
    setDeleteError("");

    try {
      await deleteListDetail(id);

      if (cancelledRef.current) return;

      images.releaseDraftBlobUrls();
      navigate("/list", { replace: true });
    } catch (caughtError) {
      if (cancelledRef.current) return;

      setDeleteError(
        String(
          caughtError instanceof Error
            ? caughtError.message
            : caughtError,
        ),
      );
    } finally {
      if (cancelledRef.current) return;
      setDeleting(false);
    }
  }, [
    listId,
    saving,
    deleting,
    cancelledRef,
    images.releaseDraftBlobUrls,
    navigate,
  ]);

  const onToggleStatus = React.useCallback(
    (next: ListStatus) => {
      if (!isEdit || saving || deleting) return;
      setDraftStatus(next);
    },
    [isEdit, saving, deleting],
  );

  const onSelectAssignee = React.useCallback(
    (id: string) => {
      if (!isEdit || saving || deleting) return;
      handleSelectAssignee(id);
    },
    [isEdit, saving, deleting, handleSelectAssignee],
  );

  const effectiveImageUrls = React.useMemo(
    () => (isEdit ? images.imageUrls : viewImageUrls),
    [isEdit, images.imageUrls, viewImageUrls],
  );

  useMainImageIndexGuard({
    imageUrls: effectiveImageUrls,
    mainImageIndex,
    setMainImageIndex,
  });

  const onChangePrice = React.useCallback(
    (
      index: number,
      price: number | undefined,
      _row: PriceRow,
    ) => {
      if (!isEdit || saving || deleting) return;

      setDraftPriceRows((previousRows) =>
        updatePriceRowPrice(previousRows, index, price),
      );
    },
    [isEdit, saving, deleting],
  );

  const onSave = React.useCallback(
    async (payload?: ListDetailSavePayload) => {
      const id = String(listId ?? "").trim();

      if (!id) {
        setSaveError("invalid_list_id");
        return;
      }

      if (!dto) {
        setSaveError("list_detail_not_loaded");
        return;
      }

      if (deleting) return;

      if (!draftAssigneeId) {
        setSaveError("assignee_required");
        return;
      }

      const saveInput = buildListDetailSaveInput({
        payload,
        draftListingTitle,
        draftDescription,
        draftStatus,
        draftAssigneeId,
        currentUserUid: user?.uid,
      });

      setSaving(true);
      setSaveError("");
      setDeleteError("");

      try {
        const result = await saveListDetailChanges({
          listId: id,
          currentDTO: dto,
          title: saveInput.title,
          description: saveInput.description,
          status: saveInput.status,
          assigneeId: saveInput.assigneeId,
          updatedBy: saveInput.updatedBy,
          draftPriceRows,
          draftImages: images.draftImages,
          mainImageIndex,
        });

        if (cancelledRef.current) return;

        images.releaseDraftBlobUrls();
        setDTO(result.dto);
        setIsEdit(false);
      } catch (caughtError) {
        if (cancelledRef.current) return;

        setSaveError(
          String(
            caughtError instanceof Error
              ? caughtError.message
              : caughtError,
          ),
        );
      } finally {
        if (cancelledRef.current) return;
        setSaving(false);
      }
    },
    [
      listId,
      dto,
      user?.uid,
      draftStatus,
      draftListingTitle,
      draftDescription,
      draftAssigneeId,
      draftPriceRows,
      images.draftImages,
      images.releaseDraftBlobUrls,
      mainImageIndex,
      deleting,
      cancelledRef,
    ],
  );

  return {
    loading,
    error,
    saving,
    saveError,
    deleting,
    deleteError,
    isEdit,
    onEdit,
    onCancel,
    onDelete,
    onSave,
    listingTitle,
    description,
    draftListingTitle,
    setDraftListingTitle,
    draftDescription,
    setDraftDescription,
    status,
    draftStatus,
    onToggleStatus,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
    imageUrls: effectiveImageUrls,
    onAddImages: images.onAddImages,
    onRemoveImageAt: images.onRemoveImageAt,
    onClearImages: images.onClearImages,
    mainImageIndex,
    setMainImageIndex,
    priceRows: viewPriceRows,
    draftPriceRows,
    onChangePrice,
    assigneeId,
    assigneeName,
    draftAssigneeId,
    draftAssigneeName,
    assigneeCandidates,
    loadingMembers,
    onSelectAssignee,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
  };
}