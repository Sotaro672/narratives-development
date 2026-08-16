// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
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
 * - 担当者選択
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
 *
 * assignee:
 * - 担当者選択はuseAssigneeSelectionを正とする
 * - 初期担当者はcurrentMemberのmembers document IDとする
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();

  const {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({
    defaultToCurrentMember: true,
  });

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
   *
   * assigneeIdの状態管理自体はuseAssigneeSelectionを正とする。
   */
  const initialTokenBlueprint = React.useMemo(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      brandName: "",
      description: "",
      assigneeId,
      minted: false,
    }),
    [assigneeId],
  );

  const onEditAssignee = React.useCallback(() => {}, []);

  const onClickAssignee = React.useCallback(() => {}, []);

  return {
    initialTokenBlueprint,

    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,
    onEditAssignee,
    onClickAssignee,

    initialEditMode: true,

    onBack,
    onSave,
  };
}