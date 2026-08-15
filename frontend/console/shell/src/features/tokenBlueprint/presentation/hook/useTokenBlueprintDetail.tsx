// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import type { ContentFile, TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import {
  deleteTokenBlueprintDetail,
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
} from "../../application/tokenBlueprintDetailService";
import { uploadAndAppendTokenBlueprintContents } from "../../application/tokenBlueprintContentService";
import { patchTokenBlueprintContentFiles } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { useTokenBlueprintCard } from "./useTokenBlueprintCard";

type TokenBlueprintCardHookResult = ReturnType<typeof useTokenBlueprintCard>;

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeId: string;
  assigneeName: string;
  minted: boolean;
  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;
  tokenContents: ContentFile[];
  cardVm: TokenBlueprintCardHookResult["vm"];
  isEditMode: boolean;
  isUploadingContents: boolean;
};

type UseTokenBlueprintDetailHandlers = {
  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => Promise<void>;
  onDelete: () => Promise<void>;
  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  cardHandlers: TokenBlueprintCardHookResult["handlers"];
  onTokenContentsFilesSelected: (files: File[]) => Promise<void>;
  onDeleteTokenContent: (item: ContentFile, index: number) => Promise<void>;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();
  const { currentMember } = useAuthContext();
  const memberId = currentMember?.id ?? "";

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = useState(false);
  const [assigneeId, setAssigneeId] = useState("");
  const [assigneeName, setAssigneeName] = useState("");
  const [isUploadingContents, setIsUploadingContents] = useState(false);

  useEffect(() => {
    const id = tokenBlueprintId;

    if (!id) {
      return;
    }

    let cancelled = false;

    const load = async (): Promise<void> => {
      try {
        setLoading(true);

        const result = await fetchTokenBlueprintDetail(id);

        if (cancelled) {
          return;
        }

        setBlueprint(result);
        setAssigneeId(result.assigneeId);
        setAssigneeName(result.assigneeName ?? "");
      } catch {
        if (!cancelled) {
          navigate("/tokenBlueprint", { replace: true });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  const minted = blueprint?.minted ?? false;
  const createdByName = blueprint?.createdByName ?? "";
  const updatedByName = blueprint?.updatedByName ?? "";
  const createdAt = safeDateTimeLabelJa(blueprint?.createdAt ?? "", "");
  const updatedAt = safeDateTimeLabelJa(blueprint?.updatedAt ?? "", "");
  const initialIconUrl = blueprint?.iconUrl ?? undefined;
  const tokenContents: ContentFile[] = blueprint?.contentFiles ?? [];

  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: blueprint ?? undefined,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  const isEditMode = cardVm.isEditMode;

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  const handleEdit = useCallback(() => {
    cardHandlers.setEditMode?.(true);
  }, [cardHandlers]);

  const handleCancel = useCallback(() => {
    cardHandlers.reset?.();
    cardHandlers.setEditMode?.(false);

    if (!blueprint) {
      setAssigneeId("");
      setAssigneeName("");
      return;
    }

    setAssigneeId(blueprint.assigneeId);
    setAssigneeName(blueprint.assigneeName ?? "");
  }, [cardHandlers, blueprint]);

  const handleSave = useCallback(async (): Promise<void> => {
    if (loading || !blueprint) {
      return;
    }

    try {
      setLoading(true);

      const sourceBlueprint: TokenBlueprint = {
        ...blueprint,
        assigneeId,
        assigneeName,
      };

      const updated = await updateTokenBlueprintFromCard(sourceBlueprint, cardVm);

      setBlueprint(updated);
      setAssigneeId(updated.assigneeId);
      setAssigneeName(updated.assigneeName ?? "");
      cardHandlers.setEditMode?.(false);

      window.alert("編集が完了しました。");
    } catch {
      // 保存失敗時は現在の編集状態を維持する。
    } finally {
      setLoading(false);
    }
  }, [loading, blueprint, assigneeId, assigneeName, cardVm, cardHandlers]);

  const handleDelete = useCallback(async (): Promise<void> => {
    if (loading || !blueprint || blueprint.minted) {
      return;
    }

    try {
      setLoading(true);
      await deleteTokenBlueprintDetail(blueprint.id);
      navigate("/tokenBlueprint", { replace: true });
    } catch {
      setLoading(false);
    }
  }, [loading, blueprint, navigate]);

  const handleSelectAssignee = useCallback((id: string) => {
    if (!id) {
      return;
    }

    setAssigneeId(id);
  }, []);

  const handleEditAssignee = useCallback(() => {
    // AdminCard互換イベント。
  }, []);

  const handleClickAssignee = useCallback(() => {
    // AdminCard互換イベント。
  }, []);

  const onTokenContentsFilesSelected = useCallback(async (files: File[]): Promise<void> => {
    const id = tokenBlueprintId;

    if (!id || !blueprint || files.length === 0) {
      return;
    }

    if (!blueprint.companyId) {
      throw new Error("companyId is required");
    }

    if (!memberId) {
      throw new Error("memberId is required");
    }

    setIsUploadingContents(true);

    try {
      const updated = await uploadAndAppendTokenBlueprintContents({
        companyId: blueprint.companyId,
        tokenBlueprintId: id,
        actorId: memberId,
        files,
        existingContentFiles: blueprint.contentFiles,
      });

      setBlueprint(updated);
      setAssigneeId(updated.assigneeId);
      setAssigneeName(updated.assigneeName ?? "");
    } finally {
      setIsUploadingContents(false);
    }
  }, [tokenBlueprintId, blueprint, memberId]);

  const onDeleteTokenContent = useCallback(async (
    item: ContentFile,
    _index: number,
  ): Promise<void> => {
    const id = tokenBlueprintId;

    if (!id || !blueprint) {
      return;
    }

    const nextContentFiles = blueprint.contentFiles.filter(
      (contentFile) => contentFile.id !== item.id,
    );

    const updated = await patchTokenBlueprintContentFiles({
      tokenBlueprintId: id,
      contentFiles: nextContentFiles,
    });

    setBlueprint(updated);
    setAssigneeId(updated.assigneeId);
    setAssigneeName(updated.assigneeName ?? "");
  }, [tokenBlueprintId, blueprint]);

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: "トークン設計",
    assigneeId,
    assigneeName,
    minted,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    tokenContents,
    cardVm,
    isEditMode,
    isUploadingContents,
  };

  const handlers: UseTokenBlueprintDetailHandlers = {
    onBack: handleBack,
    onEdit: handleEdit,
    onCancel: handleCancel,
    onSave: handleSave,
    onDelete: handleDelete,
    onSelectAssignee: handleSelectAssignee,
    onEditAssignee: handleEditAssignee,
    onClickAssignee: handleClickAssignee,
    cardHandlers,
    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  };

  return { vm, handlers };
}