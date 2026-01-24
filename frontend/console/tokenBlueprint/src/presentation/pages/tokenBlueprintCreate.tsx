// frontend/console/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx
import { useCallback, useMemo, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../components/tokenContentsCard";

import { useTokenBlueprintCreate } from "../hook/useTokenBlueprintCreate";
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

import {
  issueTokenContentsUploadURLs,
  patchTokenBlueprintContentFiles,
  type IssueTokenContentsUploadURLsRequest,
  type IssueTokenContentsUploadURLsResponse,
} from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";

function guessContentType(file: File): GCSTokenContent["type"] {
  const mime = String(file.type || "").toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "document";
}

// レスポンスの「upload」ネストに対応しつつ、安全にURLを取り出す
function getSignedUploadUrl(item: any): string {
  const nested = String(item?.upload?.uploadUrl ?? "").trim();
  if (nested) return nested;
  const flat = String(item?.uploadUrl ?? "").trim();
  return flat;
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

export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  const {
    initialTokenBlueprint,
    assigneeName,
    onEditAssignee,
    onClickAssignee,
    onBack,
    onSave, // TokenBlueprint（少なくとも id）を返す
    initialEditMode,
  } = useTokenBlueprintCreate();

  const { vm, handlers, selectedIconFile } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode,
  });

  // ★ actorId は localStorage ではなく「ログイン中 memberId」を使う
  // hook が作っている initialTokenBlueprint.createdBy は memberId のはず
  const actorId = useMemo(() => {
    return String((initialTokenBlueprint as any)?.createdBy ?? "").trim();
  }, [initialTokenBlueprint]);

  const [pending, setPending] = useState<PendingContent[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploadingContents, setIsUploadingContents] = useState(false);

  // unmount cleanup（object URL）
  useEffect(() => {
    return () => {
      for (const p of pending) URL.revokeObjectURL(p.previewUrl);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // create画面の TokenContentsCard: 選択された files を pending に積む（ローカルプレビューあり）
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

  // create画面の削除: pending の対応する index を落とす
  const handleDeleteTokenContent = useCallback(async (_item: GCSTokenContent, index: number) => {
    setPending((prev) => {
      const target = prev[index];
      if (target?.previewUrl) URL.revokeObjectURL(target.previewUrl);
      return prev.filter((_, i) => i !== index);
    });
  }, []);

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

      // ★ ここが今回の原因：localStorage ではなく currentMember.id を使う
      if (!actorId) {
        throw new Error("actorId is missing (currentMember.id / initialTokenBlueprint.createdBy)");
      }

      const files = pendingItems.map((p) => p.file);

      // 1) issue signed PUT urls
      const req: IssueTokenContentsUploadURLsRequest = {
        files: files.map((f) => {
          const contentId = uuidLike();
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

      const issued: IssueTokenContentsUploadURLsResponse = await issueTokenContentsUploadURLs({
        tokenBlueprintId,
        actorId, // ★ header X-Actor-Id
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

      // contentId -> File を対応付け
      const fileByContentId = new Map<string, File>();
      for (let i = 0; i < req.files.length; i++) {
        fileByContentId.set(req.files[i].contentId, files[i]);
      }

      // 2) PUT uploads
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

      // 3) PATCH contentFiles（replace-all）
      // backend が items[].url（閲覧用署名/表示用）を返す場合はそれも落とさず保持して送る
      const newOnes = (issued.items as any[]).map((it) => ({
        ...(it.contentFile ?? {}),
        url: String(it.url ?? it.contentFile?.url ?? "").trim() || undefined,
      }));

      await patchTokenBlueprintContentFiles({
        tokenBlueprintId,
        actorId,
        contentFiles: newOnes,
      });
    },
    [actorId],
  );

  const handleSave = useCallback(async () => {
    if (isSaving || isUploadingContents) return;

    setIsSaving(true);
    try {
      const input: Partial<TokenBlueprint> & { iconFile?: File | null } = {
        name: vm.name,
        symbol: vm.symbol,
        brandId: vm.brandId,
        description: vm.description,
        contentFiles: [] as any,
        iconFile: selectedIconFile ?? null,
      };

      // 1) create（id を返す）
      const created = await onSave(input);
      const createdId = String((created as any)?.id ?? "").trim();
      if (!createdId) {
        throw new Error(
          "created tokenBlueprint id is missing (onSave must return created entity with id)",
        );
      }

      // 2) contents upload（あれば）
      if (pending.length > 0) {
        setIsUploadingContents(true);
        try {
          await uploadContentsAfterCreate(createdId, pending);

          // object URL cleanup + pending clear
          for (const p of pending) URL.revokeObjectURL(p.previewUrl);
          setPending([]);
        } finally {
          setIsUploadingContents(false);
        }
      }

      // 3) detailへ遷移
      navigate(`/tokenBlueprint/${createdId}`, { replace: true });
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error("[TokenBlueprintCreate.page] save failed", e);
    } finally {
      setIsSaving(false);
    }
  }, [
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
      {/* 左カラム：トークン設計フォーム */}
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

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        mode="edit"
        assigneeName={assigneeName}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}
