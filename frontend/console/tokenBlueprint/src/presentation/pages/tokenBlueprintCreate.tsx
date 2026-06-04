// frontend/console/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx
import { useCallback, useMemo, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../components/tokenContentsCard";

import { useTokenBlueprintCreate } from "../hook/useTokenBlueprintCreate";
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import { useAdminCard as useAdminCardHook } from "../../../../admin/src/presentation/hook/useAdminCard";

import type {
  TokenBlueprint,
  ContentFile,
  FirebaseStorageTokenContent,
} from "../../domain/entity/tokenBlueprint";

import { patchTokenBlueprintContentFiles } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../../infrastructure/storage/tokenBlueprintAssetStorage";

function guessContentType(file: File): FirebaseStorageTokenContent["type"] {
  const mime = file.type.toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
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

  const { vm, handlers, selectedIconFile } = useTokenBlueprintCard({
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
      if (!key) return "";

      const matched = (assigneeCandidates as AssigneeCandidateLike[]).find(
        (candidate) => {
          const candidateDocId = candidate.uid?.trim() ?? "";
          return candidateDocId === key;
        },
      );

      return matched?.uid?.trim() || key;
    },
    [assigneeCandidates],
  );

  const getCandidateNameByDocId = useCallback(
    (docId: string): string => {
      const key = docId.trim();
      if (!key) return "";

      const matched = (assigneeCandidates as AssigneeCandidateLike[]).find(
        (candidate) => {
          const candidateDocId = candidate.uid?.trim() ?? "";
          return candidateDocId === key;
        },
      );

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
    const raw = initialTokenBlueprint.assigneeId?.trim() ?? "";
    if (!raw) return null;

    return normalizeAssigneeDocId(raw) || raw;
  }, [initialTokenBlueprint, normalizeAssigneeDocId]);

  const companyId = useMemo(() => {
    return initialTokenBlueprint.companyId?.trim() ?? "";
  }, [initialTokenBlueprint]);

  const createdBy = useMemo(() => {
    return initialTokenBlueprint.createdBy?.trim() ?? "";
  }, [initialTokenBlueprint.createdBy]);

  const [assigneeId, setAssigneeId] = useState<string | null>(
    initialAssigneeId,
  );

  const [selectedAssigneeName, setSelectedAssigneeName] = useState<string>(
    initialAssigneeName ?? "未設定",
  );

  const [pending, setPending] = useState<PendingContent[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploadingContents, setIsUploadingContents] = useState(false);

  useEffect(() => {
    if (initialAssigneeId) {
      setAssigneeId(initialAssigneeId);
    }
  }, [initialAssigneeId]);

  useEffect(() => {
    return () => {
      for (const p of pending) {
        URL.revokeObjectURL(p.previewUrl);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    let cancelled = false;

    const resolveInitialAssigneeName = async () => {
      if (assigneeId) {
        const localName = getCandidateNameByDocId(assigneeId);
        if (localName) {
          if (!cancelled) {
            setSelectedAssigneeName(localName);
          }
          return;
        }

        const resolved = await getAssigneeNameById(assigneeId);
        if (!cancelled) {
          setSelectedAssigneeName(resolved || "未設定");
        }
        return;
      }

      const fallback =
        initialAssigneeName?.trim() || getDefaultAssigneeName() || "未設定";

      if (!cancelled) {
        setSelectedAssigneeName(fallback);
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
    async (docId: string) => {
      const normalized = normalizeAssigneeDocId(docId);
      if (!normalized) return;

      setAssigneeId(normalized);

      const localName = getCandidateNameByDocId(normalized);
      if (localName) {
        setSelectedAssigneeName(localName);
        return;
      }

      const resolved = await getAssigneeNameById(normalized);
      setSelectedAssigneeName(resolved || "未設定");
    },
    [normalizeAssigneeDocId, getCandidateNameByDocId, getAssigneeNameById],
  );

  const handleTokenContentsFilesSelected = useCallback(async (files: File[]) => {
    if (files.length === 0) return;

    setPending((prev) => {
      const next = [...prev];

      for (const f of files) {
        const id = `local_${uuidLike()}`;
        const previewUrl = URL.createObjectURL(f);

        next.push({
          id,
          file: f,
          previewUrl,
          type: guessContentType(f),
        });
      }

      return next;
    });
  }, []);

  const handleDeleteTokenContent = useCallback(
    async (_item: FirebaseStorageTokenContent, index: number) => {
      setPending((prev) => {
        const target = prev[index];

        if (target?.previewUrl) {
          URL.revokeObjectURL(target.previewUrl);
        }

        return prev.filter((_, i) => i !== index);
      });
    },
    [],
  );

  const pendingContents: FirebaseStorageTokenContent[] = useMemo(() => {
    const nowIso = new Date().toISOString();
    const actor = createdBy || assigneeId || "";

    return pending.map((p) => ({
      id: p.id,
      name: p.file.name || p.id,
      type: p.type,
      contentType: p.file.type || "application/octet-stream",
      url: p.previewUrl,
      objectPath: "",
      visibility: "private",
      size: p.file.size,
      createdAt: nowIso,
      createdBy: actor,
      updatedAt: nowIso,
      updatedBy: actor,
    }));
  }, [pending, createdBy, assigneeId]);

  const uploadContentsAfterCreate = useCallback(
    async (tokenBlueprintId: string, pendingItems: PendingContent[]) => {
      if (!tokenBlueprintId || pendingItems.length === 0) return;

      if (!companyId) {
        throw new Error("companyId is missing");
      }

      const actor = createdBy || assigneeId || "";
      if (!actor) {
        throw new Error("createdBy is missing");
      }

      const newOnes: ContentFile[] = [];

      for (const pendingItem of pendingItems) {
        const contentId = uuidLike();
        const file = pendingItem.file;
        const nowIso = new Date().toISOString();

        const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({
          companyId,
          tokenBlueprintId,
          contentId,
          file,
        });

        newOnes.push({
          id: contentId,
          name: uploaded.fileName || file.name || contentId,
          type: uploaded.kind ?? pendingItem.type,
          contentType:
            uploaded.contentType || file.type || "application/octet-stream",
          objectPath: uploaded.objectPath,
          url: uploaded.downloadUrl,
          size:
            Number.isFinite(uploaded.size) && uploaded.size >= 0
              ? uploaded.size
              : file.size,
          visibility: "private",
          createdAt: nowIso,
          createdBy: actor,
          updatedAt: nowIso,
          updatedBy: actor,
        });
      }

      await patchTokenBlueprintContentFiles({
        tokenBlueprintId,
        contentFiles: newOnes,
      });
    },
    [companyId, createdBy, assigneeId],
  );

  const handleSave = useCallback(async () => {
    if (isSaving || isUploadingContents) return;

    setIsSaving(true);

    try {
      const assigneeDocId = assigneeId
        ? normalizeAssigneeDocId(assigneeId)
        : undefined;

      const input: Partial<TokenBlueprint> & {
        iconFile?: File | null;
        assigneeId?: string;
      } = {
        name: vm.name,
        symbol: vm.symbol,
        brandId: vm.brandId,
        description: vm.description,
        contentFiles: [],
        iconFile: selectedIconFile ?? null,
        assigneeId: assigneeDocId || undefined,
      };

      const created = await onSave(input);
      const createdId = created.id;

      if (!createdId) {
        throw new Error(
          "created tokenBlueprint id is missing (onSave must return created entity with id)",
        );
      }

      if (pending.length > 0) {
        setIsUploadingContents(true);

        try {
          await uploadContentsAfterCreate(createdId, pending);

          for (const p of pending) {
            URL.revokeObjectURL(p.previewUrl);
          }

          setPending([]);
        } finally {
          setIsUploadingContents(false);
        }
      }

      window.alert("トークン設計が完了しました。");
      navigate(`/tokenBlueprint/${createdId}`, { replace: true });
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error("[TokenBlueprintCreate.page] save failed", e);
      window.alert(
        e instanceof Error ? e.message : "トークン設計の保存に失敗しました。",
      );
    } finally {
      setIsSaving(false);
    }
  }, [
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
  ]);

  const title = useMemo(() => "トークン設計を作成", []);

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack} onSave={handleSave}>
      <div>
        <TokenBlueprintCard vm={vm} handlers={handlers} />

        <div style={{ marginTop: 16 }}>
          <TokenContentsCard
            mode="edit"
            contents={pendingContents}
            onFilesSelected={handleTokenContentsFilesSelected}
            onDelete={handleDeleteTokenContent}
          />
        </div>
      </div>

      <AdminCard
        title="管理情報"
        mode="edit"
        assigneeId={assigneeId ?? undefined}
        assigneeName={selectedAssigneeName}
        assigneeCandidates={assigneeCandidates}
        loadingMembers={loadingMembers}
        onSelectAssignee={handleSelectAssignee}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}