// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// TokenBlueprintCard 用ロジックフック
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import { fetchTokenBlueprintById } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

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
  // 詳細データ取得: backend /token-blueprints/:id
  // ─────────────────────────────
  useEffect(() => {
    const id = tokenBlueprintId?.trim();
    if (!id) {
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);

        const tb = await fetchTokenBlueprintById(id);
        if (cancelled) return;

        setBlueprint(tb);

        // 初回のみ assignee をセット
        setAssignee((prev) => prev || tb.assigneeName || tb.assigneeId || "");
      } catch {
        if (!cancelled) {
          navigate("/tokenBlueprint", { replace: true });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  // createdBy / createdAt は blueprint から算出
  const createdByName = useMemo(
    () => (blueprint as any)?.createdBy || "",
    [blueprint],
  );
  const createdAt = useMemo(
    () => (blueprint as any)?.createdAt || "",
    [blueprint],
  );

  // ─────────────────────────────
  // TokenBlueprintCard 用 VM/Handlers
  // ─────────────────────────────
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode: false, // 初期は閲覧モード
  });

  // カード側の isEditMode をヘッダー側にも反映
  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  // ─────────────────────────────
  // UI 用ハンドラ
  // ─────────────────────────────
  const handleBack = useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  // 編集開始（明示的に編集モード ON）
  const handleEdit = useCallback(() => {
    if (cardHandlers && typeof cardHandlers.setEditMode === "function") {
      cardHandlers.setEditMode(true);
    }
  }, [cardHandlers]);

  // キャンセル：編集内容を破棄して閲覧モードに戻す
  const handleCancel = useCallback(() => {
    if (cardHandlers) {
      if (typeof cardHandlers.reset === "function") {
        cardHandlers.reset();
      }
      if (typeof cardHandlers.setEditMode === "function") {
        cardHandlers.setEditMode(false);
      }
    }
  }, [cardHandlers]);

  // 保存：TODO で更新 API を呼ぶ想定。現時点では編集モードを抜けるだけ。
  const handleSave = useCallback(() => {
    if (loading) return;
    // TODO: cardVm の内容を使って updateTokenBlueprint を呼び出す

    if (cardHandlers && typeof cardHandlers.setEditMode === "function") {
      cardHandlers.setEditMode(false);
    }
  }, [loading, cardHandlers]);

  // 削除：TODO で deleteTokenBlueprint を呼ぶ想定
  const handleDelete = useCallback(() => {
    if (!blueprint) return;
    // TODO: deleteTokenBlueprint(blueprint.id) を実装
    navigate("/tokenBlueprint", { replace: true });
  }, [blueprint, navigate]);

  const handleEditAssignee = useCallback(() => {
    setAssignee("new-assignee-id");
  }, []);

  const handleClickAssignee = useCallback(() => {
    // TODO: 担当者の詳細画面などに遷移
  }, []);

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: blueprint ? `トークン設計：${blueprint.id}` : "トークン設計",
    assigneeName:
      assignee || blueprint?.assigneeName || blueprint?.assigneeId || "",
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
