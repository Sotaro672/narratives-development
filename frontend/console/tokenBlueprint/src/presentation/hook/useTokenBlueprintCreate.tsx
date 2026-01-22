// frontend/tokenBlueprint/src/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// ★ create + (optional) icon upload を application service に集約
import {
  createTokenBlueprintWithOptionalIcon,
  type CreateTokenBlueprintInput,
} from "../../application/tokenBlueprintCreateService";

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
  const createdAt = new Date().toISOString();

  // --- 戻る処理：tokenBlueprint Management へ絶対パス ---
  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  /**
   * ★ onSave が受け取る input は UI 実装に依存するため、File を運べるよう拡張して受ける
   */
  type SaveInput = Partial<TokenBlueprint> & { iconFile?: File | null };

  // --- 保存処理：create + (optional) icon upload ---
  const onSave = React.useCallback(
    async (input: SaveInput) => {
      if (!companyId) {
        throw new Error("companyId が取得できません（ログイン状態を確認してください）");
      }
      if (!memberId) {
        throw new Error("memberId が取得できません（ログイン状態を確認してください）");
      }

      const iconFile = input.iconFile ?? null;

      // entity.go（= shared TokenBlueprint 型）を正として:
      // - iconId は CreateTokenBlueprintInput には含めない（作成時に渡さない）
      // - contentFiles は string[]（ID配列）として扱う
      const payload: CreateTokenBlueprintInput = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: assignee,
        companyId,
        createdBy: memberId,

        // contentFiles: UI側は string[] を渡す（空なら []）
        contentFiles: Array.isArray(input.contentFiles) ? input.contentFiles : [],

        // ★ UI が File を持っている場合だけ渡す
        iconFile,
      };

      // eslint-disable-next-line no-console
      console.log(
        "[TokenBlueprintCreate] payload to createTokenBlueprintWithOptionalIcon:",
        {
          name: payload.name,
          symbol: payload.symbol,
          brandId: payload.brandId,
          assigneeId: payload.assigneeId,
          createdBy: payload.createdBy,
          companyId: payload.companyId,
          contentFilesCount: (payload.contentFiles ?? []).length,
          hasIconFile: Boolean(iconFile),
          iconFile: iconFile
            ? { name: iconFile.name, type: iconFile.type, size: iconFile.size }
            : null,
        },
      );

      // create (+ icon upload/attach if possible)
      const created = await createTokenBlueprintWithOptionalIcon(payload);

      // eslint-disable-next-line no-console
      console.log("[TokenBlueprintCreate] create result:", {
        id: (created as any)?.id,
        iconId: (created as any)?.iconId,
        iconUrl: (created as any)?.iconUrl,
        iconUpload: (created as any)?.iconUpload,
      });

      // 作成後に一覧へ戻る
      navigate("/tokenBlueprint", { replace: true });
    },
    [companyId, memberId, assignee, navigate],
  );

  // --- TokenBlueprint 初期値（companyId を含む） ---
  // entity.go（= shared TokenBlueprint 型）を正として:
  // - iconId は optional だが、ここで持たせたいなら「型に存在する」のでOK（ただし input/DTO には渡さない）
  // - ただし今は TS エラーになっているので、Partial<TokenBlueprint> に存在しないと解釈されている前提で削除する
  const initialTokenBlueprint: Partial<TokenBlueprint> = {
    id: "",
    name: "",
    symbol: "",
    brandId: "",
    description: "",
    companyId,
    contentFiles: [],
    assigneeId: assignee,
    createdBy: memberId,
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
