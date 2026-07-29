// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../auth/presentation/hook/useCurrentMember";

import {
  guessTokenBlueprintContentType,
} from "../../../../shared/types/tokenBlueprint";

import type {
  ContentFile,
  FirebaseStorageTokenContent,
  TokenBlueprint,
} from "../../../../shared/types/tokenBlueprint";

import {
  createTokenBlueprintWithOptionalIcon,
  type CreateTokenBlueprintInput,
} from "../../application/tokenBlueprintCreateService";

import {
  fetchTokenBlueprintById,
  patchTokenBlueprintContentFiles,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import {
  uploadTokenBlueprintContentToFirebaseStorage,
} from "../../infrastructure/storage/tokenBlueprintAssetStorage";

/**
 * TokenBlueprintCreateページ用ロジック。
 *
 * - TokenBlueprintを作成する
 * - tokenBlueprintIconはcreate service側で
 *   Firebase Storageへアップロードする
 * - tokenBlueprintContentsはfrontendから
 *   Firebase Storageへ直接アップロードする
 * - downloadURL / objectPathをcontentFilesとしてbackendへ保存する
 *
 * 保存するmember系ID:
 * - createdBy: members document ID
 * - updatedBy: members document ID
 * - assigneeId: members document ID
 * - contentFiles[].createdBy: members document ID
 * - contentFiles[].updatedBy: members document ID
 */

function createContentId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `c_${Date.now()}_${Math.random()
    .toString(16)
    .slice(2)}`;
}

function toIsoStringOrFallback(
  value: unknown,
  fallback: string,
): string {
  const raw = String(
    value ?? "",
  ).trim();

  if (!raw) {
    return fallback;
  }

  const parsed = new Date(raw);

  if (
    Number.isNaN(
      parsed.getTime(),
    )
  ) {
    return fallback;
  }

  return parsed.toISOString();
}

type SaveInput =
  Partial<TokenBlueprint> & {
    iconFile?: File | null;
  };

export function useTokenBlueprintCreate() {
  const navigate =
    useNavigate();

  const {
    currentMember,
  } = useAuth();

  const companyId =
    currentMember?.companyId ??
    "";

  /**
   * Firebase Auth UIDではなく、
   * Firestore membersのdocument IDを保存する。
   */
  const memberId =
    currentMember?.id ??
    "";

  const [
    assignee,
    setAssignee,
  ] = React.useState<string>(
    memberId,
  );

  React.useEffect(() => {
    if (
      !assignee &&
      memberId
    ) {
      setAssignee(
        memberId,
      );
    }
  }, [
    assignee,
    memberId,
  ]);

  const createdAt =
    React.useMemo(
      () => {
        return new Date().toISOString();
      },
      [],
    );

  const [
    createdBlueprint,
    setCreatedBlueprint,
  ] = React.useState<TokenBlueprint | null>(
    null,
  );

  const [
    isUploadingContents,
    setIsUploadingContents,
  ] = React.useState<boolean>(
    false,
  );

  const createdBlueprintId =
    React.useMemo(() => {
      return (
        createdBlueprint?.id ??
        ""
      );
    }, [createdBlueprint]);

  const displayAssigneeName =
    React.useMemo(() => {
      const fullName =
        `${currentMember?.lastName ?? ""} ${
          currentMember?.firstName ?? ""
        }`.trim();

      return (
        fullName ||
        currentMember?.email ||
        "未設定"
      );
    }, [currentMember]);

  const onBack =
    React.useCallback(() => {
      navigate(
        "/tokenBlueprint",
        {
          replace: true,
        },
      );
    }, [navigate]);

  const toTokenContents =
    React.useCallback(
      (
        contentFiles: ContentFile[],
      ): FirebaseStorageTokenContent[] => {
        return contentFiles
          .filter((file) => {
            return Boolean(
              file.id &&
                file.url &&
                file.objectPath,
            );
          })
          .map((file) => {
            const nowIso =
              new Date().toISOString();

            return {
              id: file.id,
              name: file.name,
              type: file.type,

              contentType:
                file.contentType ||
                "application/octet-stream",

              size:
                Number.isFinite(
                  file.size,
                ) &&
                file.size >= 0
                  ? file.size
                  : 0,

              objectPath:
                file.objectPath,

              url:
                file.url,

              isPublic:
                file.isPublic,

              createdAt:
                toIsoStringOrFallback(
                  file.createdAt,
                  nowIso,
                ),

              createdBy:
                file.createdBy ||
                memberId,

              updatedAt:
                toIsoStringOrFallback(
                  file.updatedAt,
                  nowIso,
                ),

              updatedBy:
                file.updatedBy ||
                memberId,
            };
          });
      },
      [memberId],
    );

  const onSave =
    React.useCallback(
      async (
        input: SaveInput,
      ): Promise<TokenBlueprint> => {
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

        const iconFile =
          input.iconFile ??
          null;

        const effectiveAssigneeId =
          input.assigneeId?.trim() ||
          assignee ||
          memberId;

        const payload: CreateTokenBlueprintInput = {
          name:
            input.name?.trim() ??
            "",

          symbol:
            input.symbol?.trim() ??
            "",

          brandId:
            input.brandId?.trim() ??
            "",

          description:
            input.description?.trim() ??
            "",

          assigneeId:
            effectiveAssigneeId,

          companyId,

          createdBy:
            memberId,

          iconUrl:
            input.iconUrl,

          iconObjectPath:
            input.iconObjectPath,

          iconFileName:
            input.iconFileName,

          iconContentType:
            input.iconContentType,

          iconSize:
            input.iconSize,

          contentFiles:
            input.contentFiles ??
            [],

          iconFile,
        };

        const created =
          await createTokenBlueprintWithOptionalIcon(
            payload,
          );

        if (!created.id) {
          throw new Error(
            "create result is missing tokenBlueprint.id",
          );
        }

        setAssignee(
          effectiveAssigneeId,
        );

        setCreatedBlueprint(
          created,
        );

        return created;
      },
      [
        companyId,
        memberId,
        assignee,
      ],
    );

  const tokenContents =
    React.useMemo<
      FirebaseStorageTokenContent[]
    >(() => {
      return toTokenContents(
        createdBlueprint?.contentFiles ??
          [],
      );
    }, [
      createdBlueprint,
      toTokenContents,
    ]);

  const onTokenContentsFilesSelected =
    React.useCallback(
      async (
        files: File[],
      ): Promise<void> => {
        const blueprint =
          createdBlueprint;

        if (!blueprint) {
          throw new Error(
            "TokenBlueprintが未作成です。先に保存してください。",
          );
        }

        const tokenBlueprintId =
          blueprint.id;

        if (!tokenBlueprintId) {
          throw new Error(
            "tokenBlueprint.idがありません。先に保存してください。",
          );
        }

        if (
          files.length === 0
        ) {
          return;
        }

        if (!companyId) {
          throw new Error(
            "companyId is required",
          );
        }

        if (!memberId) {
          throw new Error(
            "memberId is required",
          );
        }

        setIsUploadingContents(
          true,
        );

        try {
          const existing = [
            ...blueprint.contentFiles,
          ];

          const newContentFiles:
            ContentFile[] = [];

          for (
            const file of files
          ) {
            const contentId =
              createContentId();

            const nowIso =
              new Date().toISOString();

            const uploaded =
              await uploadTokenBlueprintContentToFirebaseStorage(
                {
                  companyId,
                  tokenBlueprintId,
                  contentId,
                  file,
                },
              );

            newContentFiles.push({
              id:
                contentId,

              name:
                uploaded.fileName ||
                file.name ||
                contentId,

              type:
                uploaded.kind ??
                guessTokenBlueprintContentType(
                  file,
                ),

              contentType:
                uploaded.contentType ||
                file.type ||
                "application/octet-stream",

              objectPath:
                uploaded.objectPath,

              url:
                uploaded.downloadUrl,

              size:
                Number.isFinite(
                  uploaded.size,
                ) &&
                uploaded.size >= 0
                  ? uploaded.size
                  : file.size,

              isPublic:
                false,

              createdAt:
                nowIso,

              createdBy:
                memberId,

              updatedAt:
                nowIso,

              updatedBy:
                memberId,
            });
          }

          const mergedContentFiles =
            new Map<
              string,
              ContentFile
            >();

          for (
            const content of existing
          ) {
            mergedContentFiles.set(
              content.id,
              content,
            );
          }

          for (
            const content of newContentFiles
          ) {
            mergedContentFiles.set(
              content.id,
              content,
            );
          }

          const updated =
            await patchTokenBlueprintContentFiles(
              {
                tokenBlueprintId,
                contentFiles:
                  Array.from(
                    mergedContentFiles.values(),
                  ),
              },
            );

          setCreatedBlueprint(
            updated,
          );

          try {
            const refreshed =
              await fetchTokenBlueprintById(
                tokenBlueprintId,
              );

            setCreatedBlueprint(
              refreshed,
            );
          } catch {
            // 更新レスポンスをそのまま使用する。
          }
        } finally {
          setIsUploadingContents(
            false,
          );
        }
      },
      [
        createdBlueprint,
        companyId,
        memberId,
      ],
    );

  const onDeleteTokenContent =
    React.useCallback(
      async (
        item:
          FirebaseStorageTokenContent,
        _index: number,
      ): Promise<void> => {
        const blueprint =
          createdBlueprint;

        if (!blueprint) {
          throw new Error(
            "TokenBlueprintが未作成です。先に保存してください。",
          );
        }

        const tokenBlueprintId =
          blueprint.id;

        if (!tokenBlueprintId) {
          throw new Error(
            "tokenBlueprint.idがありません。先に保存してください。",
          );
        }

        if (
          item.id.startsWith(
            "local_",
          )
        ) {
          return;
        }

        const nextContentFiles =
          blueprint.contentFiles.filter(
            (content) => {
              return (
                content.id !==
                item.id
              );
            },
          );

        const updated =
          await patchTokenBlueprintContentFiles(
            {
              tokenBlueprintId,
              contentFiles:
                nextContentFiles,
            },
          );

        setCreatedBlueprint(
          updated,
        );

        try {
          const refreshed =
            await fetchTokenBlueprintById(
              tokenBlueprintId,
            );

          setCreatedBlueprint(
            refreshed,
          );
        } catch {
          // 更新レスポンスをそのまま使用する。
        }
      },
      [createdBlueprint],
    );

  const initialTokenBlueprint =
    React.useMemo<
      Partial<TokenBlueprint>
    >(
      () => {
        return {
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

          createdBy:
            memberId,

          createdAt,

          updatedBy:
            memberId,

          updatedAt:
            createdAt,

          deletedAt:
            null,

          deletedBy:
            null,
        };
      },
      [
        companyId,
        assignee,
        memberId,
        createdAt,
      ],
    );

  return {
    initialTokenBlueprint,
    assigneeName:
      displayAssigneeName,
    initialEditMode:
      true,

    createdBlueprint,
    createdBlueprintId,
    tokenContents,
    isUploadingContents,

    onTokenContentsFilesSelected,
    onDeleteTokenContent,

    onEditAssignee:
      () => {},

    onClickAssignee:
      () => {},

    onBack,
    onSave,
  };
}