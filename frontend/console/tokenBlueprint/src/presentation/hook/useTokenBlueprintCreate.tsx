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
   * - TokenBlueprintCard 側で `iconFile` を載せて onSave に渡せるようにしておく（まだなら後で hook 側を更新）
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

      // --- 必須項目不足による 400 BadRequest を防止 ---
      const payload: CreateTokenBlueprintInput = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: assignee,
        companyId,
        createdBy: memberId, // ★ currentMember.id をそのまま渡す

        // ★ create 時点は基本 null（objectPath は後から付く）
        iconId: null,

        contentFiles: input.contentFiles ?? [],

        // ★ UI が File を持っている場合だけ渡す
        iconFile,
      };

      // ログ（service 層に入る前に「File が乗っているか」を確認する）
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
