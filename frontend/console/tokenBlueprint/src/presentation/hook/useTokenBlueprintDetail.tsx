// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  TokenBlueprint,
  ContentFile,
} from "../../domain/entity/tokenBlueprint";
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

import {
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
} from "../../application/tokenBlueprintDetailService";

import type { FirebaseStorageTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import { patchTokenBlueprintContentFiles } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../../infrastructure/storage/tokenBlueprintAssetStorage";

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
  onDelete: () => void;
  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  cardHandlers: any;

  onTokenContentsFilesSelected: (files: File[]) => Promise<void>;
  onDeleteTokenContent: (
    item: FirebaseStorageTokenContent,
    index: number,
  ) => Promise<void>;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

function guessContentType(file: File): FirebaseStorageTokenContent["type"] {
  if (file.type.startsWith("image/")) return "image";
  if (file.type.startsWith("video/")) return "video";
  if (file.type === "application/pdf") return "pdf";
  return "document";
}

function uuidLike(): string {
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

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();
  const { currentMember } = useAuth();

  const memberId = currentMember?.id ?? "";
  const currentCompanyId = currentMember?.companyId ?? "";

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [assigneeId, setAssigneeId] = useState<string>("");
  const [assigneeName, setAssigneeName] = useState<string>("");
  const [isUploadingContents, setIsUploadingContents] = useState<boolean>(false);

  useEffect(() => {
    const id = tokenBlueprintId?.trim();
    if (!id) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);

        const tb = await fetchTokenBlueprintDetail(id);
        if (cancelled) return;

        setBlueprint(tb);
        setAssigneeId(tb.assigneeId);
        setAssigneeName(tb.assigneeName || tb.assigneeId);
      } catch (_e) {
        if (!cancelled) navigate("/tokenBlueprint", { replace: true });
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  const minted = useMemo(() => {
    return blueprint?.minted ?? false;
  }, [blueprint]);

  const createdByName = useMemo(() => {
    return blueprint?.createdByName || blueprint?.createdBy || "";
  }, [blueprint]);

  const updatedByName = useMemo(() => {
    return blueprint?.updatedByName || blueprint?.updatedBy || "";
  }, [blueprint]);

  const createdAt = useMemo(() => {
    return safeDateTimeLabelJa(blueprint?.createdAt ?? "", "");
  }, [blueprint]);

  const updatedAt = useMemo(() => {
    return safeDateTimeLabelJa(blueprint?.updatedAt ?? "", "");
  }, [blueprint]);

  const initialIconUrl = useMemo(() => {
    return blueprint?.iconUrl || undefined;
  }, [blueprint]);

  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  const tokenContents: FirebaseStorageTokenContent[] = useMemo(() => {
    return toTokenContents(blueprint?.contentFiles ?? []);
  }, [blueprint]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  const handleEdit = useCallback(() => {
    cardHandlers?.setEditMode?.(true);
  }, [cardHandlers]);

  const handleCancel = useCallback(() => {
    cardHandlers?.reset?.();
    cardHandlers?.setEditMode?.(false);

    if (!blueprint) {
      setAssigneeId("");
      setAssigneeName("");
      return;
    }

    setAssigneeId(blueprint.assigneeId);
    setAssigneeName(blueprint.assigneeName || blueprint.assigneeId);
  }, [cardHandlers, blueprint]);

  const handleSave = useCallback(async () => {
    if (loading) return;
    if (!blueprint) return;

    try {
      setLoading(true);

      const sourceBlueprint = {
        ...blueprint,
        assigneeId,
        assigneeName,
      } as TokenBlueprint;

      const updated = await updateTokenBlueprintFromCard(sourceBlueprint, cardVm);

      setBlueprint(updated);
      setAssigneeId(updated.assigneeId);
      setAssigneeName(updated.assigneeName || updated.assigneeId);

      cardHandlers?.setEditMode?.(false);
      window.alert("編集が完了しました。");
    } catch (_err) {
      // noop
    } finally {
      setLoading(false);
    }
  }, [loading, blueprint, assigneeId, assigneeName, cardVm, cardHandlers]);

  const handleDelete = useCallback(() => {
    if (!blueprint) return;
    navigate("/tokenBlueprint", { replace: true });
  }, [blueprint, navigate]);

  const handleSelectAssignee = useCallback(
    (id: string) => {
      if (!id) return;

      const nextName =
        currentMember?.id === id ? currentMember.email || currentMember.id : id;

      setAssigneeId(id);
      setAssigneeName(nextName);
    },
    [currentMember],
  );

  const handleEditAssignee = useCallback(() => {
    // 担当者選択UIの編集イベント用
  }, []);

  const handleClickAssignee = useCallback(() => {
    // 担当者選択UIのクリックイベント用
  }, []);

  const onTokenContentsFilesSelected = useCallback(
    async (files: File[]) => {
      const id = tokenBlueprintId?.trim();
      if (!id) return;
      if (!blueprint) return;
      if (files.length === 0) return;

      const companyId = blueprint.companyId || currentCompanyId;
      if (!companyId) {
        throw new Error("companyId is missing");
      }

      setIsUploadingContents(true);

      try {
        const existing = [...blueprint.contentFiles];
        const newOnes: ContentFile[] = [];

        for (const file of files) {
          const contentId = uuidLike();

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
            size: file.size,
            objectPath: uploaded.objectPath,
            url: uploaded.downloadUrl,
            visibility: "private",
            createdAt: "",
            createdBy: memberId,
            updatedAt: "",
            updatedBy: memberId,
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

        setBlueprint(updated);

        try {
          const refreshed = await fetchTokenBlueprintDetail(id);
          setBlueprint(refreshed);
        } catch {
          // ignore
        }
      } finally {
        setIsUploadingContents(false);
      }
    },
    [tokenBlueprintId, blueprint, memberId, currentCompanyId],
  );

  const onDeleteTokenContent = useCallback(
    async (item: FirebaseStorageTokenContent, _index: number) => {
      const id = tokenBlueprintId?.trim();
      if (!id) return;
      if (!blueprint) return;

      const existing = [...blueprint.contentFiles];
      const next = existing.filter((x) => x.id !== item.id);

      const updated = await patchTokenBlueprintContentFiles({
        tokenBlueprintId: id,
        contentFiles: next,
      });

      setBlueprint(updated);

      try {
        const refreshed = await fetchTokenBlueprintDetail(id);
        setBlueprint(refreshed);
      } catch {
        // ignore
      }
    },
    [tokenBlueprintId, blueprint],
  );

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: "トークン設計",
    assigneeId: assigneeId || blueprint?.assigneeId || "",
    assigneeName:
      assigneeName || blueprint?.assigneeName || blueprint?.assigneeId || "",
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

  const handlers: UseTokenBlueprintDetailHandlers = {
    onBack: handleBack,
    onEdit: handleEdit,
    onCancel: handleCancel,
    onSave: handleSave,
    onDelete: handleDelete,
    onSelectAssignee: handleSelectAssignee,
    onEditAssignee: handleEditAssignee,
    onClickAssignee: handleClickAssignee,
    cardHandlers,
    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  };

  return { vm, handlers };
}