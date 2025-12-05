// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// TokenBlueprintCard 用ロジックフック
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

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
  // 詳細データ取得
  // ─────────────────────────────
  useEffect(() => {
    const id = tokenBlueprintId?.trim();
    if (!id) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);

        const tb = await fetchTokenBlueprintById(id);
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

  // createdAt を yyyy/mm/dd に変換
  const createdAt = useMemo(() => {
    const raw = (blueprint as any)?.createdAt;
    if (!raw) return "";

    const d = new Date(raw);
    if (isNaN(d.getTime())) return "";

    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    return `${yyyy}/${mm}/${dd}`;
  }, [blueprint]);

  // ─────────────────────────────
  // TokenBlueprintCard 用 VM / handlers
  // ─────────────────────────────
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  // ─────────────────────────────
  // UI handlers
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

      // TokenBlueprintCard の VM 構造に依存しすぎないよう any キャストで吸収
      const vmAny: any = cardVm || {};
      const fields: any = vmAny.fields ?? vmAny ?? {};

      const trimOrUndefined = (v: unknown): string | undefined =>
        typeof v === "string" ? v.trim() : undefined;

      const payload = {
        name: trimOrUndefined(fields.name ?? blueprint.name),
        symbol: trimOrUndefined(fields.symbol ?? blueprint.symbol),
        brandId: trimOrUndefined(fields.brandId ?? blueprint.brandId),
        description: trimOrUndefined(
          fields.description ?? blueprint.description,
        ),
        assigneeId: trimOrUndefined(
          fields.assigneeId ?? blueprint.assigneeId,
        ),
        iconId:
          typeof fields.iconId === "string"
            ? fields.iconId
            : (blueprint as any)?.iconId ?? null,
        contentFiles:
          (fields.contentFiles as string[] | undefined) ??
          blueprint.contentFiles ??
          [],
      };

      const updated = await updateTokenBlueprint(blueprint.id, payload);

      setBlueprint(updated);
      setAssignee(
        (prev) =>
          prev || updated.assigneeName || updated.assigneeId || "",
      );

      cardHandlers?.setEditMode?.(false);
    } catch (err) {
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
    assigneeName:
      assignee || blueprint?.assigneeName || blueprint?.assigneeId || "",
    createdByName,
    createdAt, // フォーマット済み
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
