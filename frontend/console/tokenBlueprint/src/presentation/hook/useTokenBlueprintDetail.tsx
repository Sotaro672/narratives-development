// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// TokenBlueprintCard 用ロジックフック
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";

// アプリケーション層サービス
import {
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
  formatCreatedAt,
} from "../../application/tokenBlueprintDetailService";

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;
  tokenContentsIds: string[];
  cardVm: any;
  isEditMode: boolean;
};

type UseTokenBlueprintDetailHandlers = {
  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => void;
  onDelete: () => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  cardHandlers: any;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [assignee, setAssignee] = useState<string>("");

  // ─────────────────────────────
  // 詳細データ取得（サービス経由）
  // ─────────────────────────────
  useEffect(() => {
    const id = tokenBlueprintId?.trim();
    if (!id) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);

        const tb = await fetchTokenBlueprintDetail(id);
        if (cancelled) return;

        setBlueprint(tb);
        setAssignee((prev) => prev || tb.assigneeName || tb.assigneeId || "");
      } catch {
        if (!cancelled) navigate("/tokenBlueprint", { replace: true });
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  // createdByName
  const createdByName = useMemo(
    () => (blueprint as any)?.createdBy || "",
    [blueprint],
  );

  // createdAt（サービスでフォーマット）
  const createdAt = useMemo(
    () => formatCreatedAt((blueprint as any)?.createdAt),
    [blueprint],
  );

  // ─────────────────────────────
  // TokenBlueprintCard 用 VM / handlers
  // ─────────────────────────────
  // ★重要: selectedIconFile を受け取って onSave に渡せるようにする
  const {
    vm: cardVm,
    handlers: cardHandlers,
    selectedIconFile,
  } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl: (blueprint as any)?.iconUrl ?? undefined,
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  // ─────────────────────────────
  // UI handlers（ナビゲーション周りのみ保持）
  // ─────────────────────────────
  const handleBack = useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  const handleEdit = useCallback(() => {
    cardHandlers?.setEditMode?.(true);
  }, [cardHandlers]);

  const handleCancel = useCallback(() => {
    cardHandlers?.reset?.();
    cardHandlers?.setEditMode?.(false);
  }, [cardHandlers]);

  const handleSave = useCallback(async () => {
    if (loading) return;
    if (!blueprint) return;

    try {
      setLoading(true);

      // ★ここが今回の修正ポイント:
      // - iconFile を updateTokenBlueprintFromCard に渡せるように cardVm を拡張する
      // - buildUpdatePayloadFromCardVm が vmAny.fields または vmAny を読むので、
      //   iconFile を vm 直下に載せれば拾えるようにしておく（service 側で参照する）
      const vmWithIconFile = {
        ...(cardVm ?? {}),
        iconFile: selectedIconFile ?? null,
      };

      // eslint-disable-next-line no-console
      console.log("[useTokenBlueprintDetail] save start", {
        id: blueprint.id,
        hasIconFile: Boolean(selectedIconFile),
        iconFile: selectedIconFile
          ? {
              name: selectedIconFile.name,
              type: selectedIconFile.type,
              size: selectedIconFile.size,
            }
          : null,
        isEditMode,
      });

      const updated = await updateTokenBlueprintFromCard(blueprint, vmWithIconFile);

      // eslint-disable-next-line no-console
      console.log("[useTokenBlueprintDetail] save success", {
        id: (updated as any)?.id,
        iconId: (updated as any)?.iconId,
        iconUrl: (updated as any)?.iconUrl,
      });

      setBlueprint(updated);
      setAssignee((prev) => prev || updated.assigneeName || updated.assigneeId || "");

      cardHandlers?.setEditMode?.(false);
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error("[TokenBlueprintDetail] update failed:", err);
    } finally {
      setLoading(false);
    }
  }, [loading, blueprint, cardVm, selectedIconFile, cardHandlers, isEditMode]);

  const handleDelete = useCallback(() => {
    if (!blueprint) return;
    // TODO: deleteTokenBlueprint(blueprint.id)
    navigate("/tokenBlueprint", { replace: true });
  }, [blueprint, navigate]);

  const handleEditAssignee = useCallback(() => {
    setAssignee("new-assignee-id");
  }, []);

  const handleClickAssignee = useCallback(() => {
    // TODO: 担当者詳細など
  }, []);

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: "トークン設計", // ID は表示しない
    assigneeName: assignee || blueprint?.assigneeName || blueprint?.assigneeId || "",
    createdByName,
    createdAt,
    tokenContentsIds: blueprint?.contentFiles ?? [],
    cardVm,
    isEditMode,
  };

  const handlers: UseTokenBlueprintDetailHandlers = {
    onBack: handleBack,
    onEdit: handleEdit,
    onCancel: handleCancel,
    onSave: handleSave,
    onDelete: handleDelete,
    onEditAssignee: handleEditAssignee,
    onClickAssignee: handleClickAssignee,
    cardHandlers,
  };

  return { vm, handlers };
}
