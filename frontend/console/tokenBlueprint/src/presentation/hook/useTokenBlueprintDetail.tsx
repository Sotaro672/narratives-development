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

import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;

  /**
   * TokenContentsCard へ渡す contents
   * - images 互換は削除済みのため、GCSTokenContent[] に統一
   */
  tokenContents: GCSTokenContent[];

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

// ─────────────────────────────
// 内部ヘルパ：tokenBlueprint.contentFiles の差分吸収
// - 新: ContentFile[]（{ id, name, type, url, size }）
// - 旧: string[]（URL or objectPath 等）
// 返り値は TokenContentsCard 用の GCSTokenContent[]（null を含めない）
// ─────────────────────────────
function toTokenContents(contents: unknown): GCSTokenContent[] {
  if (!Array.isArray(contents)) return [];

  const out: GCSTokenContent[] = [];

  for (let i = 0; i < contents.length; i++) {
    const x: any = contents[i];

    // 旧: string[] だった場合（URL 文字列想定）
    if (typeof x === "string") {
      const url = x.trim();
      if (!url) continue;

      out.push({
        id: `legacy_${i + 1}`,
        name: `legacy_${i + 1}`,
        type: "document",
        url,
        size: 0,
      });
      continue;
    }

    // 新: object の場合（ContentFile 相当）
    if (x && typeof x === "object") {
      const id = String(x.id ?? "").trim() || `content_${i + 1}`;
      const name = String(x.name ?? "").trim() || id;
      const type = String(x.type ?? "").trim();
      const url = String(x.url ?? "").trim();
      const size = Number(x.size ?? 0) || 0;

      // 必須の url が無いものは落とす（null を返さない）
      if (!url) continue;

      // type は domain 側の union に合わせる（不正なら document にフォールバック）
      const normalizedType: GCSTokenContent["type"] =
        type === "image" || type === "video" || type === "pdf" || type === "document"
          ? type
          : "document";

      out.push({
        id,
        name,
        type: normalizedType,
        url,
        size,
      });
    }
  }

  return out;
}

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
        setAssignee((prev) => prev || (tb as any).assigneeName || tb.assigneeId || "");
      } catch (_e) {
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
    () => String((blueprint as any)?.createdBy ?? ""),
    [blueprint],
  );

  // createdAt（サービスでフォーマット）
  const createdAt = useMemo(
    () => formatCreatedAt((blueprint as any)?.createdAt),
    [blueprint],
  );

  // Card に渡す initialIconUrl
  const initialIconUrl = useMemo(() => {
    const url = String((blueprint as any)?.iconUrl ?? "").trim();
    return url || undefined;
  }, [blueprint]);

  // ─────────────────────────────
  // TokenBlueprintCard 用 VM / handlers
  // ─────────────────────────────
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl, // ここに blueprint.iconUrl が入る想定
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  // TokenContents（TokenContentsCard へ渡す）
  const tokenContents: GCSTokenContent[] = useMemo(() => {
    return toTokenContents((blueprint as any)?.contentFiles);
  }, [blueprint]);

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

      setBlueprint(updated);
      setAssignee((prev) => prev || (updated as any).assigneeName || updated.assigneeId || "");

      cardHandlers?.setEditMode?.(false);
    } catch (_err) {
      // noop (or show toast)
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
    assigneeName: assignee || (blueprint as any)?.assigneeName || blueprint?.assigneeId || "",
    createdByName,
    createdAt,
    tokenContents,
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
