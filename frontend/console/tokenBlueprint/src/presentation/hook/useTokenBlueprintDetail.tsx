// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// TokenBlueprintCard 用ロジックフック
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";

// アプリケーション層サービス
import {
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
  formatCreatedAt,
} from "../../application/tokenBlueprintDetailService";

import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import {
  issueTokenContentsUploadURLs,
  patchTokenBlueprintContentFiles,
  type IssueTokenContentsUploadURLsRequest,
  type IssueTokenContentsUploadURLsResponse,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;

  tokenContents: GCSTokenContent[];

  cardVm: any;
  isEditMode: boolean;

  // 追加: contents アップロード中フラグ（任意でUI制御に使える）
  isUploadingContents: boolean;
};

type UseTokenBlueprintDetailHandlers = {
  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => void;
  onDelete: () => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  cardHandlers: any;

  // 追加: TokenContentsCard から呼ぶ
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

/**
 * 署名付きURL（GCS V4 Signed URL 等）を壊さない cache buster
 *
 * - 署名付きURLはクエリ文字列が署名対象なので、`&v=...` の追加は署名不一致で 403 になります。
 * - そのため、署名URLと判定できるものには一切パラメータを付与しない。
 * - 既に `v=` 等が付いている場合もそのまま返す（上書きしない）。
 */
function cacheBuster(url: string, t?: Date | number | string): string {
  const u = String(url || "").trim();
  if (!u) return "";

  // 署名付きURL判定（GCS V4 / GCS signed / AWS S3 presigned / 代表的な署名パラメータ）
  // ※ここは「安全側」に倒す（疑わしきは付与しない）
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

  // 既に v= が付いているならそのまま返す（重複させない）
  try {
    const parsed = new URL(u, typeof window !== "undefined" ? window.location.origin : "http://local");
    if (parsed.searchParams.has("v")) return u;
  } catch {
    // URL として parse できない文字列なら従来どおり append を試す
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

// tokenBlueprint.contentFiles の差分吸収（url が無い場合は contentsUrl から組み立てる）
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

    // 旧: string[] だった場合（URL 文字列想定）
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

    // 新: object の場合（ContentFile 相当）
    if (x && typeof x === "object") {
      const id = String(x.id ?? "").trim() || `content_${i + 1}`;
      const name = String(x.name ?? "").trim() || id;
      const type = String(x.type ?? "").trim();
      const size = Number(x.size ?? 0) || 0;

      // url が無い場合は contentsUrl + "/" + contentId で作る（パス規約: {tokenBlueprintId}/{contentId}）
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
  }

  return out;
}

function actorIdFromStorage(): string {
  const candidates = [
    localStorage.getItem("actorId"),
    localStorage.getItem("uid"),
    localStorage.getItem("userId"),
  ];
  return String(candidates.find((v) => v && v.trim()) || "").trim();
}

// レスポンスの「upload」ネストに対応しつつ、安全にURLを取り出す
function getSignedUploadUrl(item: any): string {
  // 新: { upload: { uploadUrl } }
  const nested = String(item?.upload?.uploadUrl ?? "").trim();
  if (nested) return nested;

  // 旧: { uploadUrl } 互換（残しておく）
  const flat = String(item?.uploadUrl ?? "").trim();
  return flat;
}

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [assignee, setAssignee] = useState<string>("");
  const [isUploadingContents, setIsUploadingContents] = useState<boolean>(false);

  // 詳細データ取得
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
        setAssignee((prev) => prev || (tb as any).assigneeName || tb.assigneeId || "");
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

  const createdByName = useMemo(() => String((blueprint as any)?.createdBy ?? ""), [blueprint]);

  const createdAt = useMemo(() => formatCreatedAt((blueprint as any)?.createdAt), [blueprint]);

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
  }, [cardHandlers]);

  const handleSave = useCallback(async () => {
    if (loading) return;
    if (!blueprint) return;

    try {
      setLoading(true);

      const updated = await updateTokenBlueprintFromCard(blueprint, cardVm);

      setBlueprint(updated);
      setAssignee((prev) => prev || (updated as any).assigneeName || updated.assigneeId || "");

      cardHandlers?.setEditMode?.(false);
    } catch (_err) {
      // noop (or show toast)
    } finally {
      setLoading(false);
    }
  }, [loading, blueprint, cardVm, cardHandlers]);

  const handleDelete = useCallback(() => {
    if (!blueprint) return;
    navigate("/tokenBlueprint", { replace: true });
  }, [blueprint, navigate]);

  const handleEditAssignee = useCallback(() => {
    setAssignee("new-assignee-id");
  }, []);

  const handleClickAssignee = useCallback(() => {
    // TODO
  }, []);

  // ─────────────────────────────
  // token-contents upload flow
  // POST signed urls -> PUT upload -> PATCH contentFiles
  // ─────────────────────────────

  const onTokenContentsFilesSelected = useCallback(
    async (files: File[]) => {
      const id = tokenBlueprintId?.trim();
      if (!id) return;
      if (!blueprint) return;

      setIsUploadingContents(true);

      try {
        const actorId = actorIdFromStorage();

        // 1) build request (contentId is generated here to keep stable mapping)
        const req: IssueTokenContentsUploadURLsRequest = {
          files: files.map((f) => {
            const contentId =
              typeof crypto !== "undefined" &&
              "randomUUID" in crypto &&
              typeof (crypto as any).randomUUID === "function"
                ? (crypto as any).randomUUID()
                : `c_${Date.now()}_${Math.random().toString(16).slice(2)}`;

            return {
              contentId,
              name: f.name || contentId,
              type: guessContentType(f),
              contentType: String(f.type || "").trim() || "application/octet-stream",
              size: typeof f.size === "number" ? f.size : 0,
              visibility: "private",
            };
          }),
        };

        // 2) issue signed urls (Cloud Run via apiPostJson)
        const issued: IssueTokenContentsUploadURLsResponse = await issueTokenContentsUploadURLs({
          tokenBlueprintId: id,
          actorId,
          body: req,
        });

        if (!issued?.items || issued.items.length === 0) {
          throw new Error("no upload url items returned");
        }
        if (issued.items.length !== files.length) {
          throw new Error(`upload url items mismatch: items=${issued.items.length} files=${files.length}`);
        }

        // 3) PUT uploads (match by contentId)
        const fileByContentId = new Map<string, File>();
        for (let i = 0; i < req.files.length; i++) {
          fileByContentId.set(req.files[i].contentId, files[i]);
        }

        for (const item of issued.items as any[]) {
          const cid = String(item?.contentId ?? "").trim();
          const file = fileByContentId.get(cid);
          if (!file) {
            throw new Error(`missing local file for contentId=${cid}`);
          }

          const contentType =
            String(item?.contentFile?.contentType ?? file.type ?? "").trim() || "application/octet-stream";

          // ★ uploadUrl は item.upload.uploadUrl（ネスト）を優先して読む
          const signedPutUrl = getSignedUploadUrl(item);
          if (!signedPutUrl) {
            throw new Error(`missing signed uploadUrl for contentId=${cid}`);
          }

          const putRes = await fetch(signedPutUrl, {
            method: "PUT",
            headers: {
              "Content-Type": contentType,
            },
            body: file,
          });

          if (!putRes.ok) {
            const text = await putRes.text().catch(() => "");
            throw new Error(`PUT to signed url failed: ${putRes.status} ${text}`);
          }
        }

        // 4) merge and PATCH contentFiles (replace-all)
        const existing = Array.isArray((blueprint as any)?.contentFiles)
          ? ([...(blueprint as any).contentFiles] as any[])
          : [];

        const newOnes = (issued.items as any[]).map((it) => {
          return {
            ...(it.contentFile ?? {}),
            url: String(it.url || "").trim(), // optional but useful for immediate rendering
          };
        });

        const merged = [...existing, ...newOnes];

        const updated = await patchTokenBlueprintContentFiles({
          tokenBlueprintId: id,
          actorId,
          contentFiles: merged,
        });

        setBlueprint(updated);

        // refresh (optional)
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
    [tokenBlueprintId, blueprint, navigate],
  );

  const onDeleteTokenContent = useCallback(
    async (item: GCSTokenContent, _index: number) => {
      const id = tokenBlueprintId?.trim();
      if (!id) return;
      if (!blueprint) return;

      const contentId = String(item?.id ?? "").trim();
      if (!contentId) return;

      // ローカルプレビューはサーバに無いのでPATCHしない
      if (contentId.startsWith("local_")) return;

      const existing = Array.isArray((blueprint as any)?.contentFiles)
        ? ([...(blueprint as any).contentFiles] as any[])
        : [];

      const next = existing.filter((x: any) => String(x?.id ?? "").trim() !== contentId);

      const actorId = actorIdFromStorage();

      const updated = await patchTokenBlueprintContentFiles({
        tokenBlueprintId: id,
        actorId,
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
    assigneeName: assignee || (blueprint as any)?.assigneeName || blueprint?.assigneeId || "",
    createdByName,
    createdAt,
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
    onEditAssignee: handleEditAssignee,
    onClickAssignee: handleClickAssignee,
    cardHandlers,

    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  };

  return { vm, handlers };
}
