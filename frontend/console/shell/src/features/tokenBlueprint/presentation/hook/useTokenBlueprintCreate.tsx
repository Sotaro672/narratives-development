// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import {
  createTokenBlueprintWithOptionalIcon,
  type CreateTokenBlueprintInput,
} from "../../application/tokenBlueprintCreateService";

/**
 * TokenBlueprint作成ページ用ロジック。
 *
 * 責務:
 * - 作成カードの初期値生成
 * - TokenBlueprint本体の新規作成
 * - トークンアイコンの保存
 * - 初期担当者の設定
 * - 一覧画面への遷移
 *
 * tokenBlueprintContentsの管理、プレビュー、upload、
 * contentFilesの更新はpages/tokenBlueprintCreate.tsxで行う。
 *
 * persisted field:
 * - companyId / createdAt / createdBy / updatedAt / updatedBy等はBackendを正とする
 * - Frontendでは仮値を生成しない
 *
 * member系ID:
 * - assigneeIdはmembers document IDを送信する
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();
  const { currentMember } = useAuthContext();

  /**
   * Firebase Auth UIDではなくFirestore membersのdocument ID。
   */
  const memberId = currentMember?.id ?? "";

  const displayAssigneeName = React.useMemo(() => {
    return currentMember?.displayName || currentMember?.email || "未設定";
  }, [currentMember?.displayName, currentMember?.email]);

  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  const onSave = React.useCallback(
    async (input: CreateTokenBlueprintInput): Promise<TokenBlueprint> => {
      if (!input.assigneeId) {
        throw new Error("assigneeId is required");
      }

      return createTokenBlueprintWithOptionalIcon(input);
    },
    [],
  );

  /**
   * TokenBlueprintCard表示用の初期値。
   *
   * Backendで永続化されるTokenBlueprint responseを模倣せず、
   * 作成フォームで必要な値だけを持つ。
   */
  const initialTokenBlueprint = React.useMemo(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      brandName: "",
      description: "",
      assigneeId: memberId,
      minted: false,
    }),
    [memberId],
  );

  return {
    initialTokenBlueprint,
    assigneeName: displayAssigneeName,
    initialEditMode: true,
    onEditAssignee: () => {},
    onClickAssignee: () => {},
    onBack,
    onSave,
  };
}