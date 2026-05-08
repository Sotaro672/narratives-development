// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

import {
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
} from "../../application/tokenBlueprintDetailService";

import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

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

  tokenContents: GCSTokenContent[];

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
  onDeleteTokenContent: (item: GCSTokenContent, index: number) => Promise<void>;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

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

function cacheBuster(url: string, t?: Date | number | string): string {
  const u = String(url || "").trim();
  if (!u) return "";

  const lower = u.toLowerCase();
  const isSignedUrl =
    lower.includes("x-goog-signature=") ||
    lower.includes("x-goog-credential=") ||
    lower.includes("x-goog-algorithm=") ||
    lower.includes("x-goog-date=") ||
    lower.includes("x-amz-signature=") ||
    lower.includes("x-amz-credential=") ||
    lower.includes("x-amz-algorithm=") ||
    lower.includes("x-amz-date=") ||
    lower.includes("signature=") ||
    lower.includes("googleaccessid=");

  if (isSignedUrl) return u;

  try {
    const parsed = new URL(
      u,
      typeof window !== "undefined" ? window.location.origin : "http://local",
    );
    if (parsed.searchParams.has("v")) return u;
  } catch {
    // noop
  }

  let ts: number | null = null;

  if (t instanceof Date) ts = t.getTime();
  else if (typeof t === "number") ts = t;
  else if (typeof t === "string") {
    const d = Date.parse(t);
    if (!Number.isNaN(d)) ts = d;
  }

  if (!ts) return u;

  const sep = u.includes("?") ? "&" : "?";
  return `${u}${sep}v=${ts}`;
}

function toTokenContents(
  contents: unknown,
  contentsBaseUrl?: string,
  blueprintVer?: unknown,
): GCSTokenContent[] {
  if (!Array.isArray(contents)) return [];

  const base = String(contentsBaseUrl || "").trim().replace(/\/+$/, "");
  const out: GCSTokenContent[] = [];

  for (let i = 0; i < contents.length; i++) {
    const x: any = contents[i];
    if (!x || typeof x !== "object") continue;

    const id = String(x.id ?? "").trim() || `content_${i + 1}`;
    const name = String(x.name ?? "").trim() || id;
    const type = String(x.type ?? "").trim();
    const size = Number(x.size ?? 0) || 0;

    let url = String(x.url ?? "").trim();
    if (!url && base && id) {
      url = `${base}/${encodeURIComponent(id)}`;
    }
    if (!url) continue;

    const normalizedType: GCSTokenContent["type"] =
      type === "image" || type === "video" || type === "pdf" || type === "document"
        ? type
        : "document";

    const ver = x.updatedAt ?? x.createdAt ?? blueprintVer;

    out.push({
      id,
      name,
      type: normalizedType,
      url: cacheBuster(url, ver),
      size,
    });
  }

  return out;
}

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();
  const { currentMember } = useAuth();

  const memberId = String(currentMember?.id ?? "").trim();
  const currentCompanyId = String(currentMember?.companyId ?? "").trim();

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

        const nextAssigneeId = String((tb as any)?.assigneeId ?? "").trim();
        const nextAssigneeName = String((tb as any)?.assigneeName ?? "").trim();

        setAssigneeId(nextAssigneeId);
        setAssigneeName(nextAssigneeName || nextAssigneeId);
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
    return Boolean((blueprint as any)?.minted);
  }, [blueprint]);

  const createdByName = useMemo(() => {
    const name = String((blueprint as any)?.createdByName ?? "").trim();
    if (name) return name;
    return String((blueprint as any)?.createdBy ?? "").trim();
  }, [blueprint]);

  const updatedByName = useMemo(() => {
    const name = String((blueprint as any)?.updatedByName ?? "").trim();
    if (name) return name;
    return String((blueprint as any)?.updatedBy ?? "").trim();
  }, [blueprint]);

  const createdAt = useMemo(() => {
    return safeDateTimeLabelJa((blueprint as any)?.createdAt, "");
  }, [blueprint]);

  const updatedAt = useMemo(() => {
    return safeDateTimeLabelJa((blueprint as any)?.updatedAt, "");
  }, [blueprint]);

  const initialIconUrl = useMemo(() => {
    const url = String((blueprint as any)?.iconUrl ?? "").trim();
    return url || undefined;
  }, [blueprint]);

  const contentsBaseUrl = useMemo(() => {
    const url = String((blueprint as any)?.contentsUrl ?? "").trim();
    return url || undefined;
  }, [blueprint]);

  const blueprintVer = useMemo(() => {
    return (blueprint as any)?.updatedAt ?? (blueprint as any)?.createdAt;
  }, [blueprint]);

  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as Partial<TokenBlueprint>,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  const isEditMode: boolean = cardVm?.isEditMode ?? false;

  const tokenContents: GCSTokenContent[] = useMemo(() => {
    return toTokenContents((blueprint as any)?.contentFiles, contentsBaseUrl, blueprintVer);
  }, [blueprint, contentsBaseUrl, blueprintVer]);

  const handleBack = useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  const handleEdit = useCallback(() => {
    cardHandlers?.setEditMode?.(true);
  }, [cardHandlers]);

  const handleCancel = useCallback(() => {
    cardHandlers?.reset?.();
    cardHandlers?.setEditMode?.(false);

    const initialAssigneeId = String((blueprint as any)?.assigneeId ?? "").trim();
    const initialAssigneeName = String((blueprint as any)?.assigneeName ?? "").trim();

    setAssigneeId(initialAssigneeId);
    setAssigneeName(initialAssigneeName || initialAssigneeId);
  }, [cardHandlers, blueprint]);

  const handleSave = useCallback(async () => {
    if (loading) return;
    if (!blueprint) return;

    try {
      setLoading(true);

      const sourceBlueprint = {
        ...(blueprint as any),
        assigneeId,
        assigneeName,
      } as TokenBlueprint;

      const updated = await updateTokenBlueprintFromCard(sourceBlueprint, cardVm);

      setBlueprint(updated);

      const nextAssigneeId = String((updated as any)?.assigneeId ?? assigneeId ?? "").trim();
      const nextAssigneeName = String(
        (updated as any)?.assigneeName ?? assigneeName ?? nextAssigneeId,
      ).trim();

      setAssigneeId(nextAssigneeId);
      setAssigneeName(nextAssigneeName || nextAssigneeId);

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
      const nextId = String(id ?? "").trim();
      if (!nextId) return;

      let nextName = "";
      if (currentMember?.id === nextId) {
        nextName = currentMember.fullName || currentMember.email || currentMember.id;
      } else {
        nextName = nextId;
      }

      setAssigneeId(nextId);
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
      if (!files || files.length === 0) return;

      if (!memberId) {
        throw new Error("actorId is missing (currentMember.id)");
      }

      const companyId = String(
        (blueprint as any)?.companyId ?? currentCompanyId ?? "",
      ).trim();

      if (!companyId) {
        throw new Error("companyId is missing");
      }

      setIsUploadingContents(true);

      try {
        const existing = Array.isArray((blueprint as any)?.contentFiles)
          ? ([...(blueprint as any).contentFiles] as any[])
          : [];

        const newOnes: any[] = [];

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
            contentType: String(file.type || "").trim() || "application/octet-stream",
            size: typeof file.size === "number" ? file.size : 0,
            objectPath: uploaded.objectPath,
            url: uploaded.downloadUrl,
            visibility: "private",
            createdBy: memberId,
            updatedBy: memberId,
          });
        }

        const mergedMap = new Map<string, any>();

        for (const x of existing) {
          const xid = String(x?.id ?? "").trim();
          if (xid) mergedMap.set(xid, x);
        }

        for (const x of newOnes) {
          const xid = String(x?.id ?? "").trim();
          if (xid) mergedMap.set(xid, x);
        }

        const merged = Array.from(mergedMap.values());

        const updated = await patchTokenBlueprintContentFiles({
          tokenBlueprintId: id,
          actorId: memberId,
          contentFiles: merged,
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
    async (item: GCSTokenContent, _index: number) => {
      const id = tokenBlueprintId?.trim();
      if (!id) return;
      if (!blueprint) return;
      if (!memberId) {
        throw new Error("actorId is missing (currentMember.id)");
      }

      const contentId = String(item?.id ?? "").trim();
      if (!contentId) return;

      if (contentId.startsWith("local_")) return;

      const existing = Array.isArray((blueprint as any)?.contentFiles)
        ? ([...(blueprint as any).contentFiles] as any[])
        : [];

      const next = existing.filter((x: any) => String(x?.id ?? "").trim() !== contentId);

      const updated = await patchTokenBlueprintContentFiles({
        tokenBlueprintId: id,
        actorId: memberId,
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
    [tokenBlueprintId, blueprint, memberId],
  );

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: "トークン設計",
    assigneeId: assigneeId || String((blueprint as any)?.assigneeId ?? "").trim(),
    assigneeName:
      assigneeName ||
      String((blueprint as any)?.assigneeName ?? "").trim() ||
      String((blueprint as any)?.assigneeId ?? "").trim(),
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