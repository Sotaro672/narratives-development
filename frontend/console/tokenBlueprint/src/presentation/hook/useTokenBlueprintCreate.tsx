// frontend/tokenBlueprint/src/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import {
  createTokenBlueprint,
  type CreateTokenBlueprintPayload,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * TokenBlueprintCreate ページ用ロジック
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();

  // currentMember から companyId / memberId を取得
  const { currentMember } = useAuth();
  const companyId = currentMember?.companyId ?? "";
  const memberId = currentMember?.id ?? "";

  // --- 管理情報（新規作成時） ---
  const [assignee, setAssignee] = React.useState(memberId);

  // 作成者/更新者 = currentMember
  const createdBy = memberId;
  const createdAt = new Date().toISOString();

  // --- 戻る処理：tokenBlueprint Management へ絶対パス ---
  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  // --- 保存処理：実 API 呼び出しに変更 ---
  const onSave = React.useCallback(
    async (input: Partial<TokenBlueprint>) => {
      if (!companyId) {
        throw new Error("companyId が取得できません（ログイン状態を確認してください）");
      }
      if (!memberId) {
        throw new Error("memberId が取得できません（ログイン状態を確認してください）");
      }

      // --- 必須項目不足による 400 BadRequest を防止 ---
      const payload: CreateTokenBlueprintPayload = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: assignee,
        companyId,
        createdBy: memberId, // ★ currentMember.id をそのまま渡す
        iconId: input.iconId ?? null,
        contentFiles: input.contentFiles ?? [],
      };

      // ★ tokenBlueprintCreateService に渡す payload を確認するログ
      console.log(
        "[TokenBlueprintCreate] payload to tokenBlueprintCreateService:",
        payload,
      );

      await createTokenBlueprint(payload);

      // 作成後に一覧へ戻る
      navigate("/tokenBlueprint", { replace: true });
    },
    [companyId, memberId, assignee, navigate],
  );

  // --- TokenBlueprint 初期値（companyId を含む） ---
  const initialTokenBlueprint: Partial<TokenBlueprint> = {
    id: "",
    name: "",
    symbol: "",
    brandId: "",
    description: "",
    companyId,
    iconId: null,
    contentFiles: [],
    assigneeId: assignee,
    createdBy: memberId, // ★ 初期値にも currentMember.id を設定
    createdAt,
    updatedBy: memberId,
    updatedAt: createdAt,
    deletedAt: null,
    deletedBy: null,
  };

  return {
    // UI へ渡す値
    initialTokenBlueprint,
    assigneeName: assignee,
    initialEditMode: true, // ← 作成時は常に edit モードで開始

    // UI トリガー
    onEditAssignee: () => setAssignee(memberId),
    onClickAssignee: () => {},

    onBack,
    onSave,
  };
}
