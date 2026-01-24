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
  issueTokenContentsUploadURLs,
  patchTokenBlueprintContentFiles,
  type IssueTokenContentsUploadURLsRequest,
  type IssueTokenContentsUploadURLsResponse,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * TokenBlueprintCreate ページ用ロジック（create + contents upload を hook に集約）
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();

  const { currentMember } = useAuth();
  const companyId = currentMember?.companyId ?? "";
  const memberId = currentMember?.id ?? "";

  // assignee は「memberId が遅れて来ても埋まる」ようにする
  const [assignee, setAssignee] = React.useState<string>(memberId);
  React.useEffect(() => {
    if (!assignee && memberId) setAssignee(memberId);
  }, [assignee, memberId]);

  const createdAt = React.useMemo(() => new Date().toISOString(), []);

  // create 後の blueprint を hook 内で保持（contents upload の前提）
  const [createdBlueprint, setCreatedBlueprint] = React.useState<TokenBlueprint | null>(null);

  // contents upload 中フラグ（UI disable などに利用）
  const [isUploadingContents, setIsUploadingContents] = React.useState<boolean>(false);

  const createdBlueprintId = React.useMemo(() => {
    return String((createdBlueprint as any)?.id ?? "").trim();
  }, [createdBlueprint]);

  const onBack = React.useCallback(() => {
    navigate("/tokenBlueprint", { replace: true });
  }, [navigate]);

  // -----------------------------
  // helpers
  // -----------------------------

  function guessContentType(file: File): GCSTokenContent["type"] {
    const mime = String(file.type || "").toLowerCase();
    if (mime.startsWith("image/")) return "image";
    if (mime.startsWith("video/")) return "video";
    if (mime === "application/pdf") return "pdf";
    return "document";
  }

  // 署名URL（X-Goog-...）を壊さない cache buster
  // - 署名URLは query 変更が即「署名不一致」になるため、付与しない
  function cacheBusterSafe(url: string, t?: Date | number | string): string {
    const u = String(url || "").trim();
    if (!u) return "";

    // Google Signed URL / AWS presigned URL などを雑に検知して「不変扱い」
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

  // tokenBlueprint.contentFiles -> TokenContentsCard 用 GCSTokenContent[]
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

      // legacy: string[] の可能性
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

        // backend が返す contentFiles[].url（閲覧用署名URL）を最優先で使う
        let url = String(x.url ?? "").trim();

        // url が無い場合のみ、base から組み立て（ただし private bucket では 403 になり得る）
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
          url: cacheBusterSafe(url, ver),
          size,
        });
      }
    }

    return out;
  }

  // レスポンスの upload ネストに対応しつつ、安全に PUT URL を取り出す
  function getSignedUploadUrl(item: any): string {
    const nested = String(item?.upload?.uploadUrl ?? "").trim();
    if (nested) return nested;
    const flat = String(item?.uploadUrl ?? "").trim();
    return flat;
  }

  // create hook は localStorage ではなく currentMember.id を actorId にする（必須）
  function actorId(): string {
    return String(memberId || "").trim();
  }

  // -----------------------------
  // create
  // -----------------------------

  type SaveInput = Partial<TokenBlueprint> & { iconFile?: File | null };

  /**
   * ★重要:
   * - createページ側で「作成IDを使って contents をアップロード」したいので、
   *   ここで navigate しない
   * - 作成結果（少なくとも id）を返し、hook 内にも保持する
   */
  const onSave = React.useCallback(
    async (input: SaveInput): Promise<TokenBlueprint> => {
      if (!companyId) {
        throw new Error("companyId が取得できません（ログイン状態を確認してください）");
      }
      if (!memberId) {
        throw new Error("memberId が取得できません（ログイン状態を確認してください）");
      }

      const iconFile = input.iconFile ?? null;

      const payload: CreateTokenBlueprintInput = {
        name: input.name?.trim() ?? "",
        symbol: input.symbol?.trim() ?? "",
        brandId: input.brandId?.trim() ?? "",
        description: input.description?.trim() ?? "",
        assigneeId: assignee,
        companyId,
        createdBy: memberId,

        // create の時点では contents は空 or 既存（通常は []）
        contentFiles: Array.isArray(input.contentFiles) ? (input.contentFiles as any) : [],

        iconFile,
      };

      const created = await createTokenBlueprintWithOptionalIcon(payload);

      const createdId = String((created as any)?.id ?? "").trim();
      if (!createdId) {
        throw new Error("create result missing id");
      }

      // create 後の state を保持（以後の contents upload はこの ID を使う）
      setCreatedBlueprint(created as TokenBlueprint);

      return created as TokenBlueprint;
    },
    [companyId, memberId, assignee],
  );

  // -----------------------------
  // contents upload (create page)
  // -----------------------------

  const contentsBaseUrl = React.useMemo(() => {
    const url = String((createdBlueprint as any)?.contentsUrl ?? "").trim();
    return url || undefined;
  }, [createdBlueprint]);

  const blueprintVer = React.useMemo(() => {
    return (createdBlueprint as any)?.updatedAt ?? (createdBlueprint as any)?.createdAt;
  }, [createdBlueprint]);

  const tokenContents: GCSTokenContent[] = React.useMemo(() => {
    return toTokenContents((createdBlueprint as any)?.contentFiles, contentsBaseUrl, blueprintVer);
  }, [createdBlueprint, contentsBaseUrl, blueprintVer]);

  const onTokenContentsFilesSelected = React.useCallback(
    async (files: File[]) => {
      const id = String((createdBlueprint as any)?.id ?? "").trim();
      if (!id) {
        throw new Error("tokenBlueprint is not created yet. Please save first.");
      }
      if (!files || files.length === 0) return;

      const actor = actorId();
      if (!actor) {
        throw new Error("actorId is missing (currentMember.id)");
      }

      setIsUploadingContents(true);

      try {
        // 1) build request（contentId はここで生成してマッピングを安定化）
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

        // 2) issue signed URLs
        const issued: IssueTokenContentsUploadURLsResponse = await issueTokenContentsUploadURLs({
          tokenBlueprintId: id,
          actorId: actor,
          body: req,
        });

        if (!issued?.items || issued.items.length === 0) {
          throw new Error("no upload url items returned");
        }
        if (issued.items.length !== files.length) {
          throw new Error(
            `upload url items mismatch: items=${issued.items.length} files=${files.length}`,
          );
        }

        // 3) PUT uploads（contentId で突合）
        const fileByContentId = new Map<string, File>();
        for (let i = 0; i < req.files.length; i++) {
          fileByContentId.set(req.files[i].contentId, files[i]);
        }

        for (const item of issued.items as any[]) {
          const cid = String(item?.contentId ?? "").trim();
          const file = fileByContentId.get(cid);
          if (!file) throw new Error(`missing local file for contentId=${cid}`);

          const contentType =
            String(item?.contentFile?.contentType ?? file.type ?? "").trim() ||
            "application/octet-stream";

          const signedPutUrl = getSignedUploadUrl(item);
          if (!signedPutUrl) throw new Error(`missing signed uploadUrl for contentId=${cid}`);

          const putRes = await fetch(signedPutUrl, {
            method: "PUT",
            headers: { "Content-Type": contentType },
            body: file,
          });

          if (!putRes.ok) {
            const text = await putRes.text().catch(() => "");
            throw new Error(`PUT to signed url failed: ${putRes.status} ${text}`);
          }
        }

        // 4) merge and PATCH contentFiles（replace-all）
        const existing = Array.isArray((createdBlueprint as any)?.contentFiles)
          ? ([...(createdBlueprint as any).contentFiles] as any[])
          : [];

        // issued.items[].contentFile をそのまま送れる形にする
        // - url は「閲覧用署名URL/表示用URL」が返っているなら保持（backend は無視しても OK）
        const newOnes = (issued.items as any[]).map((it) => ({
          ...(it.contentFile ?? {}),
          url: String(it.url || "").trim(), // frontend 表示で即使いたい場合に有効
        }));

        // id で後勝ち dedup（同一 id が来ても壊れない）
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

        // 5) refresh（閲覧用署名URLなど、backend 側の加工を確実に反映）
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
    [createdBlueprint, memberId],
  );

  const onDeleteTokenContent = React.useCallback(
    async (item: GCSTokenContent, _index: number) => {
      const id = String((createdBlueprint as any)?.id ?? "").trim();
      if (!id) throw new Error("tokenBlueprint is not created yet. Please save first.");

      const actor = actorId();
      if (!actor) throw new Error("actorId is missing (currentMember.id)");

      const contentId = String(item?.id ?? "").trim();
      if (!contentId) return;

      // ローカルプレビューはサーバに無いので PATCH しない
      if (contentId.startsWith("local_")) return;

      const existing = Array.isArray((createdBlueprint as any)?.contentFiles)
        ? ([...(createdBlueprint as any).contentFiles] as any[])
        : [];

      const next = existing.filter((x: any) => String(x?.id ?? "").trim() !== contentId);

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
    [createdBlueprint, memberId],
  );

  // -----------------------------
  // initial token blueprint (for card)
  // -----------------------------

  const initialTokenBlueprint: Partial<TokenBlueprint> = React.useMemo(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      description: "",
      companyId,
      contentFiles: [],
      assigneeId: assignee,
      createdBy: memberId,
      createdAt,
      updatedBy: memberId,
      updatedAt: createdAt,
      deletedAt: null,
      deletedBy: null,
    }),
    [companyId, assignee, memberId, createdAt],
  );

  return {
    // UI へ渡す値（既存）
    initialTokenBlueprint,
    assigneeName: assignee,
    initialEditMode: true,

    // create 後の情報（追加）
    createdBlueprint,
    createdBlueprintId,

    // TokenContentsCard 用（追加）
    tokenContents,
    isUploadingContents,
    onTokenContentsFilesSelected,
    onDeleteTokenContent,

    // UI トリガー（既存）
    onEditAssignee: () => setAssignee(memberId),
    onClickAssignee: () => {},

    onBack,
    onSave,
  };
}
