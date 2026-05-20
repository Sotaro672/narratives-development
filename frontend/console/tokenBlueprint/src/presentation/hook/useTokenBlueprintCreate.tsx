// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type {
  TokenBlueprint,
  ContentFile,
} from "../../domain/entity/tokenBlueprint";
import type { FirebaseStorageTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import {
  createTokenBlueprintWithOptionalIcon,
  type CreateTokenBlueprintInput,
} from "../../application/tokenBlueprintCreateService";

import {
  fetchTokenBlueprintById,
  patchTokenBlueprintContentFiles,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import { uploadTokenBlueprintContentToFirebaseStorage } from "../../infrastructure/storage/tokenBlueprintAssetStorage";

/**
 * TokenBlueprintCreate ページ用ロジック
 * - TokenBlueprint create
 * - tokenBlueprintIcon は create service 側で Firebase Storage へ upload
 * - tokenBlueprintContents は Firebase Storage へ frontend から直接 upload
 * - downloadURL / objectPath を contentFiles として backend に保存
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();

  const { currentMember } = useAuth();
  const companyId = currentMember?.companyId ?? "";
  const memberUid = currentMember?.uid ?? "";

  const [assignee, setAssignee] = React.useState<string>(memberUid);

  React.useEffect(() => {
    if (!assignee && memberUid) setAssignee(memberUid);
  }, [assignee, memberUid]);

  const createdAt = React.useMemo(() => new Date().toISOString(), []);

  const [createdBlueprint, setCreatedBlueprint] =
    React.useState<TokenBlueprint | null>(null);

  const [isUploadingContents, setIsUploadingContents] =
    React.useState<boolean>(false);

  const createdBlueprintId = React.useMemo(() => {
    return createdBlueprint?.id ?? "";
  }, [createdBlueprint]);

  const displayAssigneeName = React.useMemo(() => {
    return (
      `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`.trim() ||
      currentMember?.email ||
      "未設定"
    );
  }, [currentMember]);

  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  function guessContentType(file: File): FirebaseStorageTokenContent["type"] {
    if (file.type.startsWith("image/")) return "image";
    if (file.type.startsWith("video/")) return "video";
    if (file.type === "application/pdf") return "pdf";
    return "document";
  }

  function newContentId(): string {
    if (
      typeof crypto !== "undefined" &&
      "randomUUID" in crypto &&
      typeof crypto.randomUUID === "function"
    ) {
      return crypto.randomUUID();
    }

    return `c_${Date.now()}_${Math.random().toString(16).slice(2)}`;
  }

  function toTokenContents(
    contentFiles: ContentFile[],
  ): FirebaseStorageTokenContent[] {
    return contentFiles
      .filter((file) => Boolean(file.url))
      .map((file) => ({
        id: file.id,
        name: file.name,
        type: file.type,
        contentType: file.contentType,
        size: file.size,
        objectPath: file.objectPath,
        url: file.url as string,
      }));
  }

  type SaveInput = Partial<TokenBlueprint> & {
    iconFile?: File | null;
    assigneeId?: string;
  };

  const onSave = React.useCallback(
    async (input: SaveInput): Promise<TokenBlueprint> => {
      if (!companyId) {
        throw new Error("companyId が取得できません（ログイン状態を確認してください）");
      }

      if (!memberUid) {
        throw new Error("memberUid が取得できません（ログイン状態を確認してください）");
      }

      const iconFile = input.iconFile ?? null;
      const effectiveAssigneeId =
        input.assigneeId?.trim() || assignee || memberUid;

      const payload: CreateTokenBlueprintInput = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: effectiveAssigneeId,
        companyId,
        createdBy: memberUid,
        contentFiles: input.contentFiles ?? [],
        iconFile,
      };

      const created = await createTokenBlueprintWithOptionalIcon(payload);

      if (!created.id) {
        throw new Error("create result missing id");
      }

      setAssignee(effectiveAssigneeId);
      setCreatedBlueprint(created);

      return created;
    },
    [companyId, memberUid, assignee],
  );

  const tokenContents: FirebaseStorageTokenContent[] = React.useMemo(() => {
    return toTokenContents(createdBlueprint?.contentFiles ?? []);
  }, [createdBlueprint]);

  const onTokenContentsFilesSelected = React.useCallback(
    async (files: File[]) => {
      const blueprint = createdBlueprint;
      if (!blueprint) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      const id = blueprint.id;
      if (!id) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      if (files.length === 0) return;

      if (!companyId) {
        throw new Error("companyId is missing");
      }

      if (!memberUid) {
        throw new Error("memberUid is missing");
      }

      setIsUploadingContents(true);

      try {
        const existing = [...blueprint.contentFiles];
        const newOnes: ContentFile[] = [];

        for (const file of files) {
          const contentId = newContentId();

          const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({
            companyId,
            tokenBlueprintId: id,
            contentId,
            file,
          });

          newOnes.push({
            id: contentId,
            name: file.name || contentId,
            type: guessContentType(file),
            contentType: file.type || "application/octet-stream",
            objectPath: uploaded.objectPath,
            url: uploaded.downloadUrl,
            size: file.size,
            visibility: "private",
            createdAt: "",
            createdBy: memberUid,
            updatedAt: "",
            updatedBy: memberUid,
          });
        }

        const mergedMap = new Map<string, ContentFile>();

        for (const x of existing) {
          mergedMap.set(x.id, x);
        }

        for (const x of newOnes) {
          mergedMap.set(x.id, x);
        }

        const updated = await patchTokenBlueprintContentFiles({
          tokenBlueprintId: id,
          contentFiles: Array.from(mergedMap.values()),
        });

        setCreatedBlueprint(updated);

        try {
          const refreshed = await fetchTokenBlueprintById(id);
          setCreatedBlueprint(refreshed);
        } catch {
          // ignore
        }
      } finally {
        setIsUploadingContents(false);
      }
    },
    [createdBlueprint, companyId, memberUid],
  );

  const onDeleteTokenContent = React.useCallback(
    async (item: FirebaseStorageTokenContent, _index: number) => {
      const blueprint = createdBlueprint;
      if (!blueprint) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      const id = blueprint.id;
      if (!id) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      if (item.id.startsWith("local_")) return;

      const existing = [...blueprint.contentFiles];
      const next = existing.filter((x) => x.id !== item.id);

      const updated = await patchTokenBlueprintContentFiles({
        tokenBlueprintId: id,
        contentFiles: next,
      });

      setCreatedBlueprint(updated);

      try {
        const refreshed = await fetchTokenBlueprintById(id);
        setCreatedBlueprint(refreshed);
      } catch {
        // ignore
      }
    },
    [createdBlueprint],
  );

  const initialTokenBlueprint: Partial<TokenBlueprint> = React.useMemo(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      description: "",
      companyId,
      contentFiles: [],
      assigneeId: assignee || memberUid,
      createdBy: memberUid,
      createdAt,
      updatedBy: memberUid,
      updatedAt: createdAt,
      deletedAt: null,
      deletedBy: null,
    }),
    [companyId, assignee, memberUid, createdAt],
  );

  return {
    initialTokenBlueprint,
    assigneeName: displayAssigneeName,
    initialEditMode: true,
    createdBlueprint,
    createdBlueprintId,
    tokenContents,
    isUploadingContents,
    onTokenContentsFilesSelected,
    onDeleteTokenContent,
    onEditAssignee: () => {},
    onClickAssignee: () => {},
    onBack,
    onSave,
  };
}