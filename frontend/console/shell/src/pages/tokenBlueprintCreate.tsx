// frontend/console/shell/src/pages/tokenBlueprintCreate.tsx

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../features/tokenBlueprint/presentation/components/tokenContentsCard";
import TokenBlueprintCreateProgressModal from "../features/tokenBlueprint/presentation/components/tokenBlueprintCreateProgressModal";
import { useTokenBlueprintCard } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCard";
import { useTokenBlueprintCreate } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate";
import type { ContentFile } from "../shared/types/tokenBlueprint";
import { createTokenBlueprintContentId } from "../features/tokenBlueprint/application/tokenBlueprintContentService";
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
  const {
    initialTokenBlueprint,
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
    progress,
    progressOpen,
    saving,
    onCloseProgress,
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

  const [pending, setPending] = useState<PendingContent[]>([]);
  const pendingRef = useRef<PendingContent[]>([]);

  useEffect(() => {
    pendingRef.current = pending;
  }, [pending]);

  useEffect(() => {
    return () => {
      revokePendingPreviews(pendingRef.current);
    };
  }, []);

  const handleTokenContentsFilesSelected = useCallback(
    (files: File[]): void => {
      if (saving || files.length === 0) {
        return;
      }

      setPending((previousItems) => {
        const nextItems = [...previousItems];

        for (const file of files) {
          nextItems.push({
            id: createTokenBlueprintContentId(),
            file,
            previewUrl: URL.createObjectURL(file),
            type: guessTokenBlueprintContentType(file),
            contentType: getTokenBlueprintContentType(file),
          });
        }

        return nextItems;
      });
    },
    [saving],
  );

  const handleDeleteTokenContent = useCallback(
    async (item: ContentFile, _index: number): Promise<void> => {
      if (saving) {
        return;
      }

      setPending((previousItems) => {
        const target = previousItems.find(
          (pendingItem) => pendingItem.id === item.id,
        );

        if (target) {
          URL.revokeObjectURL(target.previewUrl);
        }

        return previousItems.filter(
          (pendingItem) => pendingItem.id !== item.id,
        );
      });
    },
    [saving],
  );

  /**
   * TokenContentsCard表示専用。
   *
   * objectPath・監査情報は未upload段階なので永続化には使用しない。
   * Create Operationにはid / file / typeだけを渡し、
   * contentType等の確定値はApplication層で構築する。
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
    if (saving) {
      return;
    }

    if (!assigneeId) {
      window.alert("担当者を選択してください。");
      return;
    }

    try {
      await onSave({
        name: vm.name,
        symbol: vm.symbol,
        brandId: vm.brandId,
        description: vm.description,
        assigneeId,
        iconFile: selectedIconFile ?? null,
        contents: pending.map((item) => ({
          id: item.id,
          file: item.file,
          type: item.type,
        })),
      });

      revokePendingPreviews(pending);
      setPending([]);
    } catch (error) {
      window.alert(
        error instanceof Error
          ? error.message
          : "トークン設計の保存に失敗しました。",
      );
    }
  }, [
    saving,
    assigneeId,
    vm.name,
    vm.symbol,
    vm.brandId,
    vm.description,
    selectedIconFile,
    pending,
    onSave,
  ]);

  const progressCanClose =
    progress.phase === "failed_retryable" ||
    progress.phase === "failed_fatal";

  return (
    <>
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
          assigneeId={assigneeId || undefined}
          assigneeName={assigneeName}
          assigneeCandidates={assigneeCandidates}
          loadingMembers={loadingMembers}
          onSelectAssignee={onSelectAssignee}
          onEditAssignee={onEditAssignee}
          onClickAssignee={onClickAssignee}
        />
      </PageStyle>

      <TokenBlueprintCreateProgressModal
        open={progressOpen}
        progress={progress}
        onClose={progressCanClose ? onCloseProgress : undefined}
      />
    </>
  );
}