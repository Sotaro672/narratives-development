// frontend/console/tokenOperation/src/presentation/hook/useTokenOperationDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useTokenBlueprintCard } from "../../../../tokenBlueprint/src/presentation/hook/useTokenBlueprintCard";
import type { TokenBlueprint } from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { fetchTokenBlueprintById } from "../../../../tokenBlueprint/src/infrastructure/repository/tokenBlueprintRepositoryHTTP";

type UseTokenOperationDetailReturn = {
  title: string;
  loading: boolean;
  error: string | null;
  blueprint: TokenBlueprint | null;
  cardVm: any;
  cardHandlers: any;
  assignee: string;
  creator: string;
  createdAt: string;
  onBack: () => void;
  handleSave: () => void;
};

export function useTokenOperationDetail(): UseTokenOperationDetailReturn {
  const navigate = useNavigate();
  const { tokenOperationId } = useParams<{ tokenOperationId: string }>();

  const [blueprint, setBlueprint] = React.useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);

  // Backend から TokenBlueprint を取得
  React.useEffect(() => {
    const id = tokenOperationId?.trim();
    if (!id) {
      setLoading(false);
      return;
    }

    (async () => {
      try {
        const tb = await fetchTokenBlueprintById(id);
        setBlueprint(tb);
        setError(null);
      } catch (e) {
        console.error("[useTokenOperationDetail] fetch error:", e);
        setBlueprint(null);
        setError("トークン設計の取得に失敗しました。");
      } finally {
        setLoading(false);
      }
    })();
  }, [tokenOperationId]);

  // TokenBlueprintCard 用 VM / handlers
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as any,
    initialBurnAt: "",
    initialIconUrl: blueprint?.iconId ?? "",
    initialEditMode: false,
  });

  // 管理情報（現状モック）
  const [assignee] = React.useState("member_sato");
  const [creator] = React.useState("member_yamada");
  const [createdAt] = React.useState("2025-11-06T20:55:00Z"); // ISO8601 形式に寄せる

  // 戻る
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  // 保存ボタンのアクション（モック）
  const handleSave = React.useCallback(() => {
    alert("トークン運用情報を保存しました（モック）");
  }, []);

  const title = `トークン運用：${tokenOperationId ?? "不明ID"}`;

  return {
    title,
    loading,
    error,
    blueprint,
    cardVm,
    cardHandlers,
    assignee,
    creator,
    createdAt,
    onBack,
    handleSave,
  };
}
