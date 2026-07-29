// frontend/console/shell/src/pages/tokenBlueprintCreate.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";

import AdminCard from "../features/admin/presentation/components/AdminCard";
import TokenBlueprintCard from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../features/tokenBlueprint/presentation/components/tokenContentsCard";

import { useAdminCard as useAdminCardHook } from "../features/admin/presentation/hook/useAdminCard";
import { useTokenBlueprintCard } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCard";
import { useTokenBlueprintCreate } from "../features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate";

import type {
  ContentFile,
  FirebaseStorageTokenContent,
  TokenBlueprint,
} from "../shared/types/tokenBlueprint";

import { patchTokenBlueprintContentFiles } from "../features/tokenBlueprint/infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../features/tokenBlueprint/infrastructure/storage/tokenBlueprintAssetStorage";

import "../styles/tokenBlueprint.css";

function guessContentType(
  file: File,
): FirebaseStorageTokenContent["type"] {
  const mime = file.type.toLowerCase();

  if (mime.startsWith("image/")) {
    return "image";
  }

  if (mime.startsWith("video/")) {
    return "video";
  }

  if (mime === "application/pdf") {
    return "pdf";
  }

  return "document";
}

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

type AssigneeCandidateLike = {
  uid?: string | null;
  name?: string | null;
  displayName?: string | null;
  fullName?: string | null;
  email?: string | null;
};

type PendingContent = {
  id: string;
  file: File;
  previewUrl: string;
  type: FirebaseStorageTokenContent["type"];
};

export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  const {
    initialTokenBlueprint,
    assigneeName: initialAssigneeName,
    onEditAssignee,
    onClickAssignee,
    onBack,
    onSave,
    initialEditMode,
  } = useTokenBlueprintCreate();

  const {
    vm,
    handlers,
    selectedIconFile,
  } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode,
  });

  const {
    assigneeCandidates,
    loadingMembers,
    getAssigneeNameById,
    getDefaultAssigneeName,
  } = useAdminCardHook();

  const normalizeAssigneeDocId = useCallback(
    (rawId: string): string => {
      const key = rawId.trim();

      if (!key) {
        return "";
      }

      const matched = (
        assigneeCandidates as AssigneeCandidateLike[]
      ).find((candidate) => {
        const candidateDocId =
          candidate.uid?.trim() ?? "";

        return candidateDocId === key;
      });

      return matched?.uid?.trim() || key;
    },
    [assigneeCandidates],
  );

  const getCandidateNameByDocId = useCallback(
    (docId: string): string => {
      const key = docId.trim();

      if (!key) {
        return "";
      }

      const matched = (
        assigneeCandidates as AssigneeCandidateLike[]
      ).find((candidate) => {
        const candidateDocId =
          candidate.uid?.trim() ?? "";

        return candidateDocId === key;
      });

      return (
        matched?.name?.trim() ||
        matched?.displayName?.trim() ||
        matched?.fullName?.trim() ||
        matched?.email?.trim() ||
        ""
      );
    },
    [assigneeCandidates],
  );

  const initialAssigneeId = useMemo(() => {
    const raw =
      initialTokenBlueprint.assigneeId?.trim() ?? "";

    if (!raw) {
      return null;
    }

    return normalizeAssigneeDocId(raw) || raw;
  }, [
    initialTokenBlueprint.assigneeId,
    normalizeAssigneeDocId,
  ]);

  const companyId = useMemo(() => {
    return initialTokenBlueprint.companyId?.trim() ?? "";
  }, [initialTokenBlueprint.companyId]);

  const createdBy = useMemo(() => {
    return initialTokenBlueprint.createdBy?.trim() ?? "";
  }, [initialTokenBlueprint.createdBy]);

  const [
    assigneeId,
    setAssigneeId,
  ] = useState<string | null>(
    initialAssigneeId,
  );

  const [
    selectedAssigneeName,
    setSelectedAssigneeName,
  ] = useState<string>(
    initialAssigneeName ?? "未設定",
  );

  const [
    pending,
    setPending,
  ] = useState<PendingContent[]>([]);

  const [
    isSaving,
    setIsSaving,
  ] = useState(false);

  const [
    isUploadingContents,
    setIsUploadingContents,
  ] = useState(false);

  useEffect(() => {
    if (initialAssigneeId) {
      setAssigneeId(initialAssigneeId);
    }
  }, [initialAssigneeId]);

  useEffect(() => {
    return () => {
      for (const pendingItem of pending) {
        URL.revokeObjectURL(
          pendingItem.previewUrl,
        );
      }
    };

    // pendingはコンポーネント破棄時の初回登録値を参照する。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    let cancelled = false;

    const resolveInitialAssigneeName =
      async (): Promise<void> => {
        if (assigneeId) {
          const localName =
            getCandidateNameByDocId(
              assigneeId,
            );

          if (localName) {
            if (!cancelled) {
              setSelectedAssigneeName(
                localName,
              );
            }

            return;
          }

          const resolved =
            await getAssigneeNameById(
              assigneeId,
            );

          if (!cancelled) {
            setSelectedAssigneeName(
              resolved || "未設定",
            );
          }

          return;
        }

        const fallback =
          initialAssigneeName?.trim() ||
          getDefaultAssigneeName() ||
          "未設定";

        if (!cancelled) {
          setSelectedAssigneeName(
            fallback,
          );
        }
      };

    void resolveInitialAssigneeName();

    return () => {
      cancelled = true;
    };
  }, [
    assigneeId,
    getCandidateNameByDocId,
    getAssigneeNameById,
    getDefaultAssigneeName,
    initialAssigneeName,
  ]);

  const handleSelectAssignee = useCallback(
    async (
      docId: string,
    ): Promise<void> => {
      const normalized =
        normalizeAssigneeDocId(
          docId,
        );

      if (!normalized) {
        return;
      }

      setAssigneeId(
        normalized,
      );

      const localName =
        getCandidateNameByDocId(
          normalized,
        );

      if (localName) {
        setSelectedAssigneeName(
          localName,
        );

        return;
      }

      const resolved =
        await getAssigneeNameById(
          normalized,
        );

      setSelectedAssigneeName(
        resolved || "未設定",
      );
    },
    [
      normalizeAssigneeDocId,
      getCandidateNameByDocId,
      getAssigneeNameById,
    ],
  );

  const handleTokenContentsFilesSelected =
    useCallback(
      async (
        files: File[],
      ): Promise<void> => {
        if (files.length === 0) {
          return;
        }

        setPending((previousItems) => {
          const nextItems = [
            ...previousItems,
          ];

          for (const file of files) {
            const id =
              `local_${createContentId()}`;

            const previewUrl =
              URL.createObjectURL(
                file,
              );

            nextItems.push({
              id,
              file,
              previewUrl,
              type:
                guessContentType(
                  file,
                ),
            });
          }

          return nextItems;
        });
      },
      [],
    );

  const handleDeleteTokenContent =
    useCallback(
      async (
        item: FirebaseStorageTokenContent,
        _index: number,
      ): Promise<void> => {
        setPending((previousItems) => {
          const target =
            previousItems.find(
              (pendingItem) => {
                return (
                  pendingItem.id ===
                  item.id
                );
              },
            );

          if (target?.previewUrl) {
            URL.revokeObjectURL(
              target.previewUrl,
            );
          }

          return previousItems.filter(
            (pendingItem) => {
              return (
                pendingItem.id !==
                item.id
              );
            },
          );
        });
      },
      [],
    );

  const pendingContents =
    useMemo<
      FirebaseStorageTokenContent[]
    >(() => {
      const nowIso =
        new Date().toISOString();

      const actor =
        createdBy ||
        assigneeId ||
        "";

      return pending.map(
        (pendingItem) => {
          return {
            id:
              pendingItem.id,

            name:
              pendingItem.file.name ||
              pendingItem.id,

            type:
              pendingItem.type,

            contentType:
              pendingItem.file.type ||
              "application/octet-stream",

            url:
              pendingItem.previewUrl,

            objectPath:
              "",

            isPublic:
              false,

            size:
              pendingItem.file.size,

            createdAt:
              nowIso,

            createdBy:
              actor,

            updatedAt:
              nowIso,

            updatedBy:
              actor,
          };
        },
      );
    }, [
      pending,
      createdBy,
      assigneeId,
    ]);

  const uploadContentsAfterCreate =
    useCallback(
      async (
        tokenBlueprintId: string,
        pendingItems: PendingContent[],
      ): Promise<void> => {
        if (
          !tokenBlueprintId ||
          pendingItems.length === 0
        ) {
          return;
        }

        if (!companyId) {
          throw new Error(
            "companyId is required",
          );
        }

        const actor =
          createdBy ||
          assigneeId ||
          "";

        if (!actor) {
          throw new Error(
            "createdBy is required",
          );
        }

        const newContentFiles:
          ContentFile[] = [];

        for (
          const pendingItem of pendingItems
        ) {
          const contentId =
            createContentId();

          const file =
            pendingItem.file;

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
              pendingItem.type,

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
              actor,

            updatedAt:
              nowIso,

            updatedBy:
              actor,
          });
        }

        await patchTokenBlueprintContentFiles({
          tokenBlueprintId,
          contentFiles:
            newContentFiles,
        });
      },
      [
        companyId,
        createdBy,
        assigneeId,
      ],
    );

  const handleSave =
    useCallback(
      async (): Promise<void> => {
        if (
          isSaving ||
          isUploadingContents
        ) {
          return;
        }

        setIsSaving(
          true,
        );

        try {
          const assigneeDocId =
            assigneeId
              ? normalizeAssigneeDocId(
                  assigneeId,
                )
              : undefined;

          const input:
            Partial<TokenBlueprint> & {
              iconFile?: File | null;
            } = {
              name:
                vm.name,

              symbol:
                vm.symbol,

              brandId:
                vm.brandId,

              description:
                vm.description,

              contentFiles:
                [],

              iconFile:
                selectedIconFile ??
                null,

              assigneeId:
                assigneeDocId ||
                undefined,
            };

          const created =
            await onSave(
              input,
            );

          const createdId =
            created.id;

          if (!createdId) {
            throw new Error(
              "作成結果にtokenBlueprint.idがありません。",
            );
          }

          if (
            pending.length > 0
          ) {
            setIsUploadingContents(
              true,
            );

            try {
              await uploadContentsAfterCreate(
                createdId,
                pending,
              );

              for (
                const pendingItem of pending
              ) {
                URL.revokeObjectURL(
                  pendingItem.previewUrl,
                );
              }

              setPending(
                [],
              );
            } finally {
              setIsUploadingContents(
                false,
              );
            }
          }

          window.alert(
            "トークン設計が完了しました。",
          );

          navigate(
            `/tokenBlueprint/${encodeURIComponent(
              createdId,
            )}`,
            {
              replace: true,
            },
          );
        } catch (error) {
          // eslint-disable-next-line no-console
          console.error(
            "[TokenBlueprintCreate.page] save failed",
            error,
          );

          window.alert(
            error instanceof Error
              ? error.message
              : "トークン設計の保存に失敗しました。",
          );
        } finally {
          setIsSaving(
            false,
          );
        }
      },
      [
        assigneeId,
        normalizeAssigneeDocId,
        isSaving,
        isUploadingContents,
        vm.name,
        vm.symbol,
        vm.brandId,
        vm.description,
        selectedIconFile,
        onSave,
        pending,
        uploadContentsAfterCreate,
        navigate,
      ],
    );

  const title =
    useMemo(
      () => {
        return "トークン設計を作成";
      },
      [],
    );

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={handleSave}
    >
      <div>
        <TokenBlueprintCard
          vm={vm}
          handlers={handlers}
        />

        <div
          style={{
            marginTop: 16,
          }}
        >
          <TokenContentsCard
            mode="edit"
            contents={pendingContents}
            onFilesSelected={
              handleTokenContentsFilesSelected
            }
            onDelete={
              handleDeleteTokenContent
            }
          />
        </div>
      </div>

      <AdminCard
        title="管理情報"
        mode="edit"
        assigneeId={
          assigneeId ??
          undefined
        }
        assigneeName={
          selectedAssigneeName
        }
        assigneeCandidates={
          assigneeCandidates
        }
        loadingMembers={
          loadingMembers
        }
        onSelectAssignee={
          handleSelectAssignee
        }
        onEditAssignee={
          onEditAssignee
        }
        onClickAssignee={
          onClickAssignee
        }
      />
    </PageStyle>
  );
}