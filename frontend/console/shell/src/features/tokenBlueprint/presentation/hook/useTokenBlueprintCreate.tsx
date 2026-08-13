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
 * - TokenBlueprintの初期値生成
 * - TokenBlueprint本体の新規作成
 * - トークンアイコンの保存
 * - 作成画面の担当者初期値管理
 * - 一覧画面への遷移
 *
 * tokenBlueprintContentsの管理、プレビュー、アップロード、
 * contentFilesの更新はpages/tokenBlueprintCreate.tsxで行う。
 *
 * member系ID:
 * - assigneeIdはfrontendからmembers document IDを送信する
 * - createdBy / updatedByはbackendの認証コンテキストを正とする
 */
type SaveInput = Partial<TokenBlueprint> & {
  iconFile?: File | null;
};

export function useTokenBlueprintCreate() {
  const navigate = useNavigate();
  const { currentMember } = useAuthContext();

  const companyId = currentMember?.companyId ?? "";

  /**
   * Firebase Auth UIDではなく、
   * Firestore membersのdocument IDを担当者IDとして使用する。
   */
  const memberId = currentMember?.id ?? "";

  const [assignee, setAssignee] = React.useState<string>(memberId);

  React.useEffect(() => {
    if (!assignee && memberId) {
      setAssignee(memberId);
    }
  }, [assignee, memberId]);

  const createdAt = React.useMemo(
    () => new Date().toISOString(),
    [],
  );

  const displayAssigneeName = React.useMemo(() => {
    const fullName = `${currentMember?.lastName ?? ""} ${
      currentMember?.firstName ?? ""
    }`.trim();

    return (
      currentMember?.displayName?.trim() ||
      fullName ||
      currentMember?.email?.trim() ||
      "未設定"
    );
  }, [
    currentMember?.displayName,
    currentMember?.lastName,
    currentMember?.firstName,
    currentMember?.email,
  ]);

  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", {
      replace: true,
    });
  }, [navigate]);

  const onSave = React.useCallback(
    async (input: SaveInput): Promise<TokenBlueprint> => {
      if (!companyId) {
        throw new Error(
          "companyIdが取得できません。ログイン状態を確認してください。",
        );
      }

      if (!memberId) {
        throw new Error(
          "memberIdが取得できません。ログイン状態を確認してください。",
        );
      }

      const iconFile = input.iconFile ?? null;
      const effectiveAssigneeId =
        input.assigneeId?.trim() ||
        assignee ||
        memberId;

      const payload: CreateTokenBlueprintInput = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: effectiveAssigneeId,
        companyId,

        iconUrl: input.iconUrl,
        iconObjectPath: input.iconObjectPath,
        iconFileName: input.iconFileName,
        iconContentType: input.iconContentType,
        iconSize: input.iconSize,

        contentFiles: input.contentFiles ?? [],
        iconFile,
      };

      const created = await createTokenBlueprintWithOptionalIcon(
        payload,
      );

      if (!created.id) {
        throw new Error(
          "create result is missing tokenBlueprint.id",
        );
      }

      setAssignee(effectiveAssigneeId);

      return created;
    },
    [
      companyId,
      memberId,
      assignee,
    ],
  );

  const initialTokenBlueprint = React.useMemo<
    Partial<TokenBlueprint>
  >(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      brandName: "",
      description: "",
      companyId,

      contentFiles: [],

      assigneeId:
        assignee ||
        memberId,

      createdBy: memberId,
      createdAt,

      updatedBy: memberId,
      updatedAt: createdAt,

      deletedAt: null,
      deletedBy: null,
    }),
    [
      companyId,
      assignee,
      memberId,
      createdAt,
    ],
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