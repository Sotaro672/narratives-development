// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

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
 *
 * IMPORTANT:
 * - assigneeId / createdBy / updatedBy / actorId は Firebase UID を送信する
 * - currentMember.id は Firestore member docId のため、この hook では使わない
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
    return String((createdBlueprint as any)?.id ?? "").trim();
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

  function guessContentType(file: File): GCSTokenContent["type"] {
    const mime = String(file.type || "").toLowerCase();
    if (mime.startsWith("image/")) return "image";
    if (mime.startsWith("video/")) return "video";
    if (mime === "application/pdf") return "pdf";
    return "document";
  }

  function newContentId(): string {
    if (
      typeof crypto !== "undefined" &&
      "randomUUID" in crypto &&
      typeof (crypto as any).randomUUID === "function"
    ) {
      return (crypto as any).randomUUID();
    }

    return `c_${Date.now()}_${Math.random().toString(16).slice(2)}`;
  }

  function cacheBusterSafe(url: string, t?: Date | number | string): string {
    const u = String(url || "").trim();
    if (!u) return "";

    const lower = u.toLowerCase();
    const looksSigned =
      lower.includes("x-goog-algorithm=") ||
      lower.includes("x-goog-signature=") ||
      lower.includes("x-goog-credential=") ||
      lower.includes("x-amz-algorithm=") ||
      lower.includes("x-amz-signature=");

    if (looksSigned) return u;

    let ts: number | null = null;
    if (t instanceof Date) ts = t.getTime();
    else if (typeof t === "number") ts = t;
    else if (typeof t === "string") {
      const d = Date.parse(t);
      if (!Number.isNaN(d)) ts = d;
    }
    if (!ts) return u;

    try {
      const asUrl = new URL(u);
      asUrl.searchParams.set("v", String(ts));
      return asUrl.toString();
    } catch {
      const sep = u.includes("?") ? "&" : "?";
      return `${u}${sep}v=${ts}`;
    }
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

      if (typeof x === "string") {
        const url = x.trim();
        if (!url) continue;

        out.push({
          id: `legacy_${i + 1}`,
          name: `legacy_${i + 1}`,
          type: "document",
          url,
          size: 0,
        });

        continue;
      }

      if (x && typeof x === "object") {
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
          type === "image" ||
          type === "video" ||
          type === "pdf" ||
          type === "document"
            ? type
            : "document";

        const ver = x.updatedAt ?? x.createdAt ?? blueprintVer;

        out.push({
          id,
          name,
          type: normalizedType,
          url: cacheBusterSafe(url, ver),
          size,
        });
      }
    }

    return out;
  }

  function actorId(): string {
    return String(memberUid || "").trim();
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

      // input.assigneeId が渡ってきた場合はそれを優先。
      // ただし、呼び出し元も uid を渡す必要がある。
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
        contentFiles: Array.isArray(input.contentFiles)
          ? (input.contentFiles as any)
          : [],
        iconFile,
      };

      const created = await createTokenBlueprintWithOptionalIcon(payload);

      const createdId = String((created as any)?.id ?? "").trim();
      if (!createdId) {
        throw new Error("create result missing id");
      }

      setAssignee(effectiveAssigneeId);
      setCreatedBlueprint(created as TokenBlueprint);

      return created as TokenBlueprint;
    },
    [companyId, memberUid, assignee],
  );

  const contentsBaseUrl = React.useMemo(() => {
    const url = String((createdBlueprint as any)?.contentsUrl ?? "").trim();
    return url || undefined;
  }, [createdBlueprint]);

  const blueprintVer = React.useMemo(() => {
    return (createdBlueprint as any)?.updatedAt ?? (createdBlueprint as any)?.createdAt;
  }, [createdBlueprint]);

  const tokenContents: GCSTokenContent[] = React.useMemo(() => {
    return toTokenContents(
      (createdBlueprint as any)?.contentFiles,
      contentsBaseUrl,
      blueprintVer,
    );
  }, [createdBlueprint, contentsBaseUrl, blueprintVer]);

  const onTokenContentsFilesSelected = React.useCallback(
    async (files: File[]) => {
      const id = String((createdBlueprint as any)?.id ?? "").trim();
      if (!id) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      if (!files || files.length === 0) return;

      if (!companyId) {
        throw new Error("companyId is missing");
      }

      const actor = actorId();
      if (!actor) {
        throw new Error("actorId is missing (currentMember.uid)");
      }

      setIsUploadingContents(true);

      try {
        const existing = Array.isArray((createdBlueprint as any)?.contentFiles)
          ? ([...(createdBlueprint as any).contentFiles] as any[])
          : [];

        const newOnes: any[] = [];

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
            contentType:
              String(file.type || "").trim() || "application/octet-stream",
            objectPath: uploaded.objectPath,
            url: uploaded.downloadUrl,
            size: typeof file.size === "number" ? file.size : 0,
            visibility: "private",
            createdBy: actor,
            updatedBy: actor,
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
          actorId: actor,
          contentFiles: merged,
        });

        setCreatedBlueprint(updated as any);

        try {
          const refreshed = await fetchTokenBlueprintById(id);
          setCreatedBlueprint(refreshed as any);
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
    async (item: GCSTokenContent, _index: number) => {
      const id = String((createdBlueprint as any)?.id ?? "").trim();
      if (!id) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }

      const actor = actorId();
      if (!actor) {
        throw new Error("actorId is missing (currentMember.uid)");
      }

      const contentId = String(item?.id ?? "").trim();
      if (!contentId) return;

      if (contentId.startsWith("local_")) return;

      const existing = Array.isArray((createdBlueprint as any)?.contentFiles)
        ? ([...(createdBlueprint as any).contentFiles] as any[])
        : [];

      const next = existing.filter((x: any) => {
        return String(x?.id ?? "").trim() !== contentId;
      });

      const updated = await patchTokenBlueprintContentFiles({
        tokenBlueprintId: id,
        actorId: actor,
        contentFiles: next,
      });

      setCreatedBlueprint(updated as any);

      try {
        const refreshed = await fetchTokenBlueprintById(id);
        setCreatedBlueprint(refreshed as any);
      } catch {
        // ignore
      }
    },
    [createdBlueprint, memberUid],
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