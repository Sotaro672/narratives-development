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

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import { patchTokenBlueprintContentFiles } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../../infrastructure/storage/tokenBlueprintAssetStorage";

function guessContentType(file: File): GCSTokenContent["type"] {
  const mime = String(file.type || "").toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "document";
}

function uuidLike(): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto &&
    typeof (crypto as any).randomUUID === "function"
  ) {
    return (crypto as any).randomUUID();
  }

  return `c_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

type PendingContent = {
  id: string;
  file: File;
  previewUrl: string;
  type: GCSTokenContent["type"];
};

type AssigneeCandidateLike = {
  id?: string | null;
  uid?: string | null;
  name?: string | null;
  displayName?: string | null;
  fullName?: string | null;
  email?: string | null;
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

  const getCandidateUidByIdOrUid = useCallback(
    (idOrUid: string): string => {
      const key = String(idOrUid ?? "").trim();
      if (!key) return "";

      const matched = (assigneeCandidates as AssigneeCandidateLike[]).find(
        (candidate) => {
          const candidateUid = String(candidate?.uid ?? "").trim();
          const candidateId = String(candidate?.id ?? "").trim();

          return candidateUid === key || candidateId === key;
        },
      );

      const uid = String(matched?.uid ?? "").trim();

      // uid が候補に存在する場合は必ず uid を使う。
      // 候補に uid が無い場合は、既に uid が渡っている可能性を考慮して key を返す。
      return uid || key;
    },
    [assigneeCandidates],
  );

  const getCandidateNameByIdOrUid = useCallback(
    (idOrUid: string): string => {
      const key = String(idOrUid ?? "").trim();
      if (!key) return "";

      const matched = (assigneeCandidates as AssigneeCandidateLike[]).find(
        (candidate) => {
          const candidateUid = String(candidate?.uid ?? "").trim();
          const candidateId = String(candidate?.id ?? "").trim();

          return candidateUid === key || candidateId === key;
        },
      );

      return String(
        matched?.name ??
          matched?.displayName ??
          matched?.fullName ??
          matched?.email ??
          "",
      ).trim();
    },
    [assigneeCandidates],
  );

  const initialAssigneeId = useMemo(() => {
    const candidate =
      (initialTokenBlueprint as any)?.assigneeId ??
      (initialTokenBlueprint as any)?.assigneeMemberId ??
      null;

    const raw = String(candidate ?? "").trim();
    if (!raw) return null;

    return getCandidateUidByIdOrUid(raw) || raw;
  }, [initialTokenBlueprint, getCandidateUidByIdOrUid]);

  const actorId = useMemo(() => {
    return String((initialTokenBlueprint as any)?.createdBy ?? "").trim();
  }, [initialTokenBlueprint]);

  const companyId = useMemo(() => {
    return String((initialTokenBlueprint as any)?.companyId ?? "").trim();
  }, [initialTokenBlueprint]);

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
        const localName = getCandidateNameByIdOrUid(assigneeId);
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
    getCandidateNameByIdOrUid,
    getAssigneeNameById,
    getDefaultAssigneeName,
    initialAssigneeName,
  ]);

  const handleSelectAssignee = useCallback(
    async (idOrUid: string) => {
      const normalized = String(idOrUid ?? "").trim();
      if (!normalized) return;

      const uid = getCandidateUidByIdOrUid(normalized);
      const nextAssigneeId = uid || normalized;

      setAssigneeId(nextAssigneeId);

      const localName = getCandidateNameByIdOrUid(nextAssigneeId);
      if (localName) {
        setSelectedAssigneeName(localName);
        return;
      }

      const resolved = await getAssigneeNameById(nextAssigneeId);
      setSelectedAssigneeName(resolved || "未設定");
    },
    [getCandidateUidByIdOrUid, getCandidateNameByIdOrUid, getAssigneeNameById],
  );

  const handleTokenContentsFilesSelected = useCallback(async (files: File[]) => {
    if (!files || files.length === 0) return;

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
    async (_item: GCSTokenContent, index: number) => {
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

  const pendingContents: GCSTokenContent[] = useMemo(() => {
    return pending.map((p) => ({
      id: p.id,
      name: p.file.name || p.id,
      type: p.type,
      url: p.previewUrl,
      size: typeof p.file.size === "number" ? p.file.size : 0,
    }));
  }, [pending]);

  const uploadContentsAfterCreate = useCallback(
    async (tokenBlueprintId: string, pendingItems: PendingContent[]) => {
      if (!tokenBlueprintId || pendingItems.length === 0) return;

      if (!actorId) {
        throw new Error(
          "actorId is missing (currentMember.uid / initialTokenBlueprint.createdBy)",
        );
      }

      if (!companyId) {
        throw new Error("companyId is missing");
      }

      const newOnes: any[] = [];

      for (const pendingItem of pendingItems) {
        const contentId = uuidLike();
        const file = pendingItem.file;

        const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({
          companyId,
          tokenBlueprintId,
          contentId,
          file,
        });

        newOnes.push({
          id: contentId,
          name: file.name || contentId,
          type: pendingItem.type,
          contentType:
            String(file.type || "").trim() || "application/octet-stream",
          objectPath: uploaded.objectPath,
          url: uploaded.downloadUrl,
          size: typeof file.size === "number" ? file.size : 0,
          visibility: "private",
          createdBy: actorId,
          updatedBy: actorId,
        });
      }

      await patchTokenBlueprintContentFiles({
        tokenBlueprintId,
        actorId,
        contentFiles: newOnes,
      });
    },
    [actorId, companyId],
  );

  const handleSave = useCallback(async () => {
    if (isSaving || isUploadingContents) return;

    setIsSaving(true);

    try {
      const assigneeUid = assigneeId
        ? getCandidateUidByIdOrUid(assigneeId)
        : undefined;

      const input: Partial<TokenBlueprint> & {
        iconFile?: File | null;
        assigneeId?: string;
      } = {
        name: vm.name,
        symbol: vm.symbol,
        brandId: vm.brandId,
        description: vm.description,
        contentFiles: [] as any,
        iconFile: selectedIconFile ?? null,
        assigneeId: assigneeUid || undefined,
      };

      const created = await onSave(input);
      const createdId = String((created as any)?.id ?? "").trim();

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
    getCandidateUidByIdOrUid,
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