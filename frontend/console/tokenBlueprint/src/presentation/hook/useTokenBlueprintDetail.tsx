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

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintDetail] effect start", {
      tokenBlueprintIdRaw: tokenBlueprintId ?? "",
      tokenBlueprintId: id ?? "",
    });

    if (!id) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);

        // eslint-disable-next-line no-console
        console.log("[useTokenBlueprintDetail] fetchTokenBlueprintDetail start", {
          id,
        });

        const tb = await fetchTokenBlueprintDetail(id);

        // eslint-disable-next-line no-console
        console.log("[useTokenBlueprintDetail] fetchTokenBlueprintDetail success (raw)", {
          id,
          tb,
        });

        // eslint-disable-next-line no-console
        console.log("[useTokenBlueprintDetail] fetchTokenBlueprintDetail success (fields)", {
          id: (tb as any)?.id,
          name: (tb as any)?.name,
          symbol: (tb as any)?.symbol,
          brandId: (tb as any)?.brandId,
          brandName: (tb as any)?.brandName,
          assigneeId: (tb as any)?.assigneeId,
          assigneeName: (tb as any)?.assigneeName,
          minted: (tb as any)?.minted,
          iconId: (tb as any)?.iconId,
          iconUrl: (tb as any)?.iconUrl,
          metadataUri: (tb as any)?.metadataUri,
          createdAt: (tb as any)?.createdAt,
          updatedAt: (tb as any)?.updatedAt,
        });

        if (cancelled) return;

        setBlueprint(tb);
        setAssignee((prev) => prev || tb.assigneeName || tb.assigneeId || "");
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useTokenBlueprintDetail] fetchTokenBlueprintDetail failed", {
          id,
          error: e,
        });

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

  // ★ Card に渡す initialIconUrl を可視化
  const initialIconUrl = useMemo(() => {
    const url = String((blueprint as any)?.iconUrl ?? "").trim();
    const out = url || undefined;

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintDetail] initialIconUrl computed", {
      blueprintId: String((blueprint as any)?.id ?? ""),
      rawIconUrl: (blueprint as any)?.iconUrl,
      computed: out ?? "",
      iconId: String((blueprint as any)?.iconId ?? ""),
      hasBlueprint: Boolean(blueprint),
    });

    return out;
  }, [blueprint]);

  // ─────────────────────────────
  // TokenBlueprintCard 用 VM / handlers
  // ─────────────────────────────
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl, // ★ ここに blueprint.iconUrl が入る想定
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  // ★ cardVm 側の iconUrl の最終値もログで追う
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintDetail] cardVm updated", {
      blueprintId: String((blueprint as any)?.id ?? ""),
      cardIconUrl: String(cardVm?.iconUrl ?? ""),
      cardBrandId: String(cardVm?.brandId ?? ""),
      cardBrandName: String(cardVm?.brandName ?? ""),
      cardMinted: Boolean(cardVm?.minted),
      isEditMode: Boolean(cardVm?.isEditMode),
    });
  }, [cardVm, blueprint]);

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

      const updated = await updateTokenBlueprintFromCard(blueprint, cardVm);

      // eslint-disable-next-line no-console
      console.log("[useTokenBlueprintDetail] updateTokenBlueprintFromCard result", {
        id: (updated as any)?.id,
        iconId: (updated as any)?.iconId,
        iconUrl: (updated as any)?.iconUrl,
        minted: (updated as any)?.minted,
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
  }, [loading, blueprint, cardVm, cardHandlers]);

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
