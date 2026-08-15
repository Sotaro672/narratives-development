// frontend/console/shell/src/pages/tokenBlueprintCreate.tsx

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../features/tokenBlueprint/presentation/components/tokenContentsCard";
import { useAdminCard as useAdminCardHook } from "../features/admin/presentation/hook/useAdminCard";
import { useTokenBlueprintCard } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCard";
import { useTokenBlueprintCreate } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate";
import type { ContentFile } from "../shared/types/tokenBlueprint";
import {
  createTokenBlueprintContentId,
  uploadAndAppendTokenBlueprintContents,
} from "../features/tokenBlueprint/application/tokenBlueprintContentService";
import {
  getTokenBlueprintContentType,
  guessTokenBlueprintContentType,
} from "../features/tokenBlueprint/infrastructure/storage/tokenBlueprintAssetStorage";

import "../styles/tokenBlueprint.css";

type PendingContent = {
  id: string;
  file: File;
  previewUrl: string;
  type: ContentFile["type"];
  contentType: string;
};

function revokePendingPreviews(items: PendingContent[]): void {
  for (const item of items) {
    URL.revokeObjectURL(item.previewUrl);
  }
}

export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  const {
    initialTokenBlueprint,
    assigneeName: initialAssigneeName,
    onEditAssignee,
    onClickAssignee,
    onBack,
    onSave,
    initialEditMode,
  } = useTokenBlueprintCreate();

  const { vm, handlers, selectedIconFile } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode,
  });

  const {
    assigneeCandidates,
    loadingMembers,
    getDefaultAssigneeName,
  } = useAdminCardHook();

  const initialAssigneeId = initialTokenBlueprint.assigneeId || null;
  const [assigneeId, setAssigneeId] = useState<string | null>(initialAssigneeId);
  const [pending, setPending] = useState<PendingContent[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploadingContents, setIsUploadingContents] = useState(false);
  const pendingRef = useRef<PendingContent[]>([]);

  useEffect(() => {
    pendingRef.current = pending;
  }, [pending]);

  useEffect(() => {
    if (initialAssigneeId) {
      setAssigneeId(initialAssigneeId);
    }
  }, [initialAssigneeId]);

  useEffect(() => {
    return () => {
      revokePendingPreviews(pendingRef.current);
    };
  }, []);

  const selectedAssigneeName = useMemo(() => {
    if (!assigneeId) {
      return initialAssigneeName || getDefaultAssigneeName();
    }

    const candidate = assigneeCandidates.find((item) => item.id === assigneeId);

    if (candidate) {
      return candidate.name;
    }

    if (assigneeId === initialAssigneeId) {
      return initialAssigneeName || "未設定";
    }

    return "未設定";
  }, [
    assigneeId,
    initialAssigneeId,
    initialAssigneeName,
    assigneeCandidates,
    getDefaultAssigneeName,
  ]);

  const handleSelectAssignee = useCallback((id: string): void => {
    if (!id) {
      return;
    }

    setAssigneeId(id);
  }, []);

  const handleTokenContentsFilesSelected = useCallback((files: File[]): void => {
    if (files.length === 0) {
      return;
    }

    setPending((previousItems) => {
      const nextItems = [...previousItems];

      for (const file of files) {
        nextItems.push({
          id: `local_${createTokenBlueprintContentId()}`,
          file,
          previewUrl: URL.createObjectURL(file),
          type: guessTokenBlueprintContentType(file),
          contentType: getTokenBlueprintContentType(file),
        });
      }

      return nextItems;
    });
  }, []);

  const handleDeleteTokenContent = useCallback(
    async (item: ContentFile, _index: number): Promise<void> => {
      setPending((previousItems) => {
        const target = previousItems.find((pendingItem) => pendingItem.id === item.id);

        if (target) {
          URL.revokeObjectURL(target.previewUrl);
        }

        return previousItems.filter((pendingItem) => pendingItem.id !== item.id);
      });
    },
    [],
  );

  /**
   * TokenContentsCard表示専用。
   * objectPath・監査情報は未upload段階なので永続化には使用しない。
   * create APIへこの配列を送信しないことを前提とする。
   */
  const pendingContents = useMemo<ContentFile[]>(() => {
    const nowIso = new Date().toISOString();

    return pending.map((item) => ({
      id: item.id,
      name: item.file.name,
      type: item.type,
      contentType: item.contentType,
      url: item.previewUrl,
      objectPath: "",
      isPublic: false,
      size: item.file.size,
      createdAt: nowIso,
      createdBy: "",
      updatedAt: nowIso,
      updatedBy: "",
    }));
  }, [pending]);

  const handleSave = useCallback(async (): Promise<void> => {
    if (isSaving || isUploadingContents) {
      return;
    }

    if (!assigneeId) {
      window.alert("担当者を選択してください。");
      return;
    }

    setIsSaving(true);

    try {
      const created = await onSave({
        name: vm.name,
        symbol: vm.symbol,
        brandId: vm.brandId,
        description: vm.description,
        assigneeId,
        iconFile: selectedIconFile ?? null,
      });

      if (pending.length > 0) {
        if (!created.createdBy) {
          throw new Error("作成結果にcreatedByがありません。");
        }

        setIsUploadingContents(true);

        try {
          await uploadAndAppendTokenBlueprintContents({
            companyId: created.companyId,
            tokenBlueprintId: created.id,
            actorId: created.createdBy,
            files: pending.map((item) => item.file),
            existingContentFiles: created.contentFiles,
          });

          revokePendingPreviews(pending);
          setPending([]);
        } finally {
          setIsUploadingContents(false);
        }
      }

      window.alert("トークン設計が完了しました。");
      navigate(`/tokenBlueprint/${encodeURIComponent(created.id)}`, { replace: true });
    } catch (error) {
      console.error("[TokenBlueprintCreate.page] save failed", error);
      window.alert(
        error instanceof Error
          ? error.message
          : "トークン設計の保存に失敗しました。",
      );
    } finally {
      setIsSaving(false);
    }
  }, [
    assigneeId,
    isSaving,
    isUploadingContents,
    vm.name,
    vm.symbol,
    vm.brandId,
    vm.description,
    selectedIconFile,
    onSave,
    pending,
    navigate,
  ]);

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={onBack}
      onSave={handleSave}
    >
      <div>
        <TokenBlueprintCard vm={vm} handlers={handlers} />

        <div style={{ marginTop: 16 }}>
          <TokenContentsCard
            mode="edit"
            contents={pendingContents}
            onFilesSelected={handleTokenContentsFilesSelected}
            onDelete={handleDeleteTokenContent}
          />
        </div>
      </div>

      <AdminCard
        title="管理情報"
        mode="edit"
        assigneeId={assigneeId ?? undefined}
        assigneeName={selectedAssigneeName}
        assigneeCandidates={assigneeCandidates}
        loadingMembers={loadingMembers}
        onSelectAssignee={handleSelectAssignee}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}