// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintDetail.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import type {
  ContentFile,
  FirebaseStorageTokenContent,
  TokenBlueprint,
} from "../../../../shared/types/tokenBlueprint";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import { useTokenBlueprintCard } from "./useTokenBlueprintCard";

import {
  deleteTokenBlueprintDetail,
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
} from "../../application/tokenBlueprintDetailService";

import {
  uploadAndAppendTokenBlueprintContents,
} from "../../application/tokenBlueprintContentService";

import {
  patchTokenBlueprintContentFiles,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;

  title: string;

  assigneeId: string;
  assigneeName: string;

  minted: boolean;

  createdByName: string;
  createdAt: string;

  updatedByName: string;
  updatedAt: string;

  tokenContents: FirebaseStorageTokenContent[];

  cardVm: any;

  isEditMode: boolean;
  isUploadingContents: boolean;
};

type UseTokenBlueprintDetailHandlers = {
  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => Promise<void>;
  onDelete: () => Promise<void>;

  onSelectAssignee: (id: string) => void;

  onEditAssignee: () => void;
  onClickAssignee: () => void;

  cardHandlers: any;

  onTokenContentsFilesSelected: (
    files: File[],
  ) => Promise<void>;

  onDeleteTokenContent: (
    item: FirebaseStorageTokenContent,
    index: number,
  ) => Promise<void>;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

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

  if (Number.isNaN(parsed.getTime())) {
    return fallback;
  }

  return parsed.toISOString();
}

function toTokenContents(
  contentFiles: ContentFile[],
): FirebaseStorageTokenContent[] {
  return contentFiles
    .filter((contentFile) => {
      return Boolean(
        contentFile.id &&
        contentFile.url &&
        contentFile.objectPath,
      );
    })
    .map((contentFile) => {
      const nowIso = new Date().toISOString();

      return {
        id: contentFile.id,
        name: contentFile.name,
        type: contentFile.type,

        contentType:
          contentFile.contentType ||
          "application/octet-stream",

        size:
          Number.isFinite(contentFile.size) &&
          contentFile.size >= 0
            ? contentFile.size
            : 0,

        objectPath: contentFile.objectPath,
        url: contentFile.url,
        isPublic: contentFile.isPublic,

        createdAt:
          toIsoStringOrFallback(
            contentFile.createdAt,
            nowIso,
          ),

        createdBy:
          contentFile.createdBy ||
          "",

        updatedAt:
          toIsoStringOrFallback(
            contentFile.updatedAt,
            nowIso,
          ),

        updatedBy:
          contentFile.updatedBy ||
          "",
      };
    });
}

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();

  const {
    tokenBlueprintId,
  } = useParams<{
    tokenBlueprintId: string;
  }>();

  const {
    currentMember,
  } = useAuthContext();

  const memberId =
    currentMember?.id ??
    "";

  const currentCompanyId =
    currentMember?.companyId ??
    "";

  const [
    blueprint,
    setBlueprint,
  ] = useState<TokenBlueprint | null>(
    null,
  );

  const [
    loading,
    setLoading,
  ] = useState<boolean>(
    false,
  );

  const [
    assigneeId,
    setAssigneeId,
  ] = useState<string>("");

  const [
    assigneeName,
    setAssigneeName,
  ] = useState<string>("");

  const [
    isUploadingContents,
    setIsUploadingContents,
  ] = useState<boolean>(
    false,
  );

  useEffect(() => {
    const id =
      tokenBlueprintId?.trim();

    if (!id) {
      return;
    }

    let cancelled = false;

    const load =
      async (): Promise<void> => {
        try {
          setLoading(true);

          const result =
            await fetchTokenBlueprintDetail(id);

          if (cancelled) {
            return;
          }

          setBlueprint(result);
          setAssigneeId(result.assigneeId);

          setAssigneeName(
            result.assigneeName ||
            result.assigneeId,
          );
        } catch {
          if (!cancelled) {
            navigate(
              "/tokenBlueprint",
              {
                replace: true,
              },
            );
          }
        } finally {
          if (!cancelled) {
            setLoading(false);
          }
        }
      };

    void load();

    return () => {
      cancelled = true;
    };
  }, [
    tokenBlueprintId,
    navigate,
  ]);

  const minted =
    useMemo(() => {
      return blueprint?.minted ?? false;
    }, [blueprint]);

  const createdByName =
    useMemo(() => {
      return (
        blueprint?.createdByName ||
        blueprint?.createdBy ||
        ""
      );
    }, [blueprint]);

  const updatedByName =
    useMemo(() => {
      return (
        blueprint?.updatedByName ||
        blueprint?.updatedBy ||
        ""
      );
    }, [blueprint]);

  const createdAt =
    useMemo(() => {
      return safeDateTimeLabelJa(
        blueprint?.createdAt ?? "",
        "",
      );
    }, [blueprint]);

  const updatedAt =
    useMemo(() => {
      return safeDateTimeLabelJa(
        blueprint?.updatedAt ?? "",
        "",
      );
    }, [blueprint]);

  const initialIconUrl =
    useMemo(() => {
      return (
        blueprint?.iconUrl ||
        undefined
      );
    }, [blueprint]);

  const {
    vm: cardVm,
    handlers: cardHandlers,
  } = useTokenBlueprintCard({
    initialTokenBlueprint:
      blueprint ?? {},

    initialBurnAt: "",

    initialIconUrl,

    initialEditMode: false,
  });

  const isEditMode: boolean =
    cardVm?.isEditMode ??
    false;

  const tokenContents =
    useMemo<
      FirebaseStorageTokenContent[]
    >(() => {
      return toTokenContents(
        blueprint?.contentFiles ??
        [],
      );
    }, [blueprint]);

  const handleBack =
    useCallback(() => {
      navigate(
        "/tokenBlueprint",
        {
          replace: true,
        },
      );
    }, [navigate]);

  const handleEdit =
    useCallback(() => {
      cardHandlers?.setEditMode?.(
        true,
      );
    }, [cardHandlers]);

  const handleCancel =
    useCallback(() => {
      cardHandlers?.reset?.();
      cardHandlers?.setEditMode?.(
        false,
      );

      if (!blueprint) {
        setAssigneeId("");
        setAssigneeName("");
        return;
      }

      setAssigneeId(
        blueprint.assigneeId,
      );

      setAssigneeName(
        blueprint.assigneeName ||
        blueprint.assigneeId,
      );
    }, [
      cardHandlers,
      blueprint,
    ]);

  const handleSave =
    useCallback(
      async (): Promise<void> => {
        if (
          loading ||
          !blueprint
        ) {
          return;
        }

        try {
          setLoading(true);

          const sourceBlueprint:
            TokenBlueprint = {
              ...blueprint,
              assigneeId,
              assigneeName,
            };

          const updated =
            await updateTokenBlueprintFromCard(
              sourceBlueprint,
              cardVm,
            );

          setBlueprint(updated);
          setAssigneeId(
            updated.assigneeId,
          );

          setAssigneeName(
            updated.assigneeName ||
            updated.assigneeId,
          );

          cardHandlers?.setEditMode?.(
            false,
          );

          window.alert(
            "編集が完了しました。",
          );
        } catch {
          // 保存失敗時は現在の編集状態を維持する。
        } finally {
          setLoading(false);
        }
      },
      [
        loading,
        blueprint,
        assigneeId,
        assigneeName,
        cardVm,
        cardHandlers,
      ],
    );

  const handleDelete =
    useCallback(
      async (): Promise<void> => {
        if (
          loading ||
          !blueprint ||
          blueprint.minted
        ) {
          return;
        }

        const id = blueprint.id;

        if (!id) {
          return;
        }

        try {
          setLoading(true);

          await deleteTokenBlueprintDetail(
            id,
          );

          navigate(
            "/tokenBlueprint",
            {
              replace: true,
            },
          );
        } catch {
          setLoading(false);
        }
      },
      [
        loading,
        blueprint,
        navigate,
      ],
    );

  const handleSelectAssignee =
    useCallback(
      (
        id: string,
      ) => {
        if (!id) {
          return;
        }

        const nextName =
          currentMember?.id === id
            ? currentMember.email ||
              currentMember.id
            : id;

        setAssigneeId(id);
        setAssigneeName(nextName);
      },
      [currentMember],
    );

  const handleEditAssignee =
    useCallback(() => {
      // 担当者選択UIの編集イベント用。
    }, []);

  const handleClickAssignee =
    useCallback(() => {
      // 担当者選択UIのクリックイベント用。
    }, []);

  const onTokenContentsFilesSelected =
    useCallback(
      async (
        files: File[],
      ): Promise<void> => {
        const id =
          tokenBlueprintId?.trim();

        if (
          !id ||
          !blueprint ||
          files.length === 0
        ) {
          return;
        }

        const companyId =
          blueprint.companyId ||
          currentCompanyId;

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

        setIsUploadingContents(true);

        try {
          const updated =
            await uploadAndAppendTokenBlueprintContents(
              {
                companyId,
                tokenBlueprintId: id,
                actorId: memberId,
                files,
                existingContentFiles:
                  blueprint.contentFiles,
              },
            );

          setBlueprint(updated);

          try {
            const refreshed =
              await fetchTokenBlueprintDetail(
                id,
              );

            setBlueprint(refreshed);
          } catch {
            // 更新レスポンスをそのまま使用する。
          }
        } finally {
          setIsUploadingContents(false);
        }
      },
      [
        tokenBlueprintId,
        blueprint,
        memberId,
        currentCompanyId,
      ],
    );

  const onDeleteTokenContent =
    useCallback(
      async (
        item:
          FirebaseStorageTokenContent,
        _index: number,
      ): Promise<void> => {
        const id =
          tokenBlueprintId?.trim();

        if (
          !id ||
          !blueprint
        ) {
          return;
        }

        if (
          item.id.startsWith("local_",)
        ) {
          return;
        }

        const nextContentFiles =
          blueprint.contentFiles.filter(
            (contentFile) => {
              return (
                contentFile.id !==
                item.id
              );
            },
          );

        const updated =
          await patchTokenBlueprintContentFiles(
            {
              tokenBlueprintId: id,
              contentFiles:
                nextContentFiles,
            },
          );

        setBlueprint(updated);

        try {
          const refreshed =
            await fetchTokenBlueprintDetail(
              id,
            );

          setBlueprint(refreshed);
        } catch {
          // 更新レスポンスをそのまま使用する。
        }
      },
      [
        tokenBlueprintId,
        blueprint,
      ],
    );

  const vm:
    UseTokenBlueprintDetailVM = {
      blueprint,

      title: "トークン設計",

      assigneeId:
        assigneeId ||
        blueprint?.assigneeId ||
        "",

      assigneeName:
        assigneeName ||
        blueprint?.assigneeName ||
        blueprint?.assigneeId ||
        "",

      minted,
      createdByName,
      createdAt,
      updatedByName,
      updatedAt,
      tokenContents,
      cardVm,
      isEditMode,
      isUploadingContents,
    };

  const handlers:
    UseTokenBlueprintDetailHandlers = {
      onBack: handleBack,
      onEdit: handleEdit,
      onCancel: handleCancel,
      onSave: handleSave,
      onDelete: handleDelete,
      onSelectAssignee:handleSelectAssignee,
      onEditAssignee:handleEditAssignee,
      onClickAssignee:handleClickAssignee,
      cardHandlers,
      onTokenContentsFilesSelected,
      onDeleteTokenContent,
    };

  return {
    vm,
    handlers,
  };
}