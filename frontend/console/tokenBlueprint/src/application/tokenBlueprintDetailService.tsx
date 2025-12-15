// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";
import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
  putFileToSignedUrl,
  attachTokenBlueprintIcon,
  type UpdateTokenBlueprintPayload,
  type SignedIconUpload,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * 詳細取得（リポジトリのラッパー）
 */
export async function fetchTokenBlueprintDetail(
  id: string,
): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) {
    throw new Error("id is empty");
  }
  return fetchTokenBlueprintById(trimmed);
}

/**
 * createdAt を yyyy/mm/dd にフォーマット
 */
export function formatCreatedAt(raw: unknown): string {
  if (!raw) return "";

  let d: Date;
  if (raw instanceof Date) {
    d = raw;
  } else {
    d = new Date(raw as any);
  }

  if (isNaN(d.getTime())) return "";

  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}/${mm}/${dd}`;
}

/**
 * TokenBlueprintCard の VM から UpdateTokenBlueprintPayload を組み立てる
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: any,
): UpdateTokenBlueprintPayload {
  const vmAny: any = cardVm || {};
  const fields: any = vmAny.fields ?? vmAny ?? {};

  const trimOrUndefined = (v: unknown): string | undefined =>
    typeof v === "string" ? v.trim() : undefined;

  const payload: UpdateTokenBlueprintPayload = {
    name: trimOrUndefined(fields.name ?? blueprint.name),
    symbol: trimOrUndefined(fields.symbol ?? blueprint.symbol),
    brandId: trimOrUndefined(fields.brandId ?? blueprint.brandId),
    description: trimOrUndefined(fields.description ?? blueprint.description),
    assigneeId: trimOrUndefined(fields.assigneeId ?? blueprint.assigneeId),

    /**
     * 注意:
     * - 画像アップロードと iconId の確定は別フロー（PUT完了後に objectPath を入れる）
     * - ここでは「UI が iconId を文字列で持っていても」通常はそのまま反映してしまうと、
     *   PUTせずに iconId だけ更新される事故が起きるので、呼び出し側で制御する。
     */
    iconId:
      typeof fields.iconId === "string"
        ? fields.iconId
        : (blueprint as any)?.iconId ?? null,

    contentFiles:
      (fields.contentFiles as string[] | undefined) ??
      blueprint.contentFiles ??
      [],
  };

  return payload;
}

type UpdateFromCardOptions = {
  /**
   * ★ 選択されたアイコンファイル（あれば Signed URL PUT → iconId 反映まで行う）
   */
  iconFile?: File | null;

  /**
   * ★ デバッグ用途: 強制的に iconUpload を見たい/試したい場合に true
   * （現状の backend 実装が update レスポンスでも iconUpload を返す想定）
   */
  forceIconUploadFlow?: boolean;
};

/**
 * TokenBlueprintCard の VM から update API を呼び出し、更新後の TokenBlueprint を返す
 *
 * ★重要（今回の不具合対策）:
 * - iconFile がある場合は「PUT前に iconId を入れない」
 * - update → (iconUpload取得) → PUT → iconId(objectPath)更新 の順で行う
 */
export async function updateTokenBlueprintFromCard(
  blueprint: TokenBlueprint,
  cardVm: any,
  options?: UpdateFromCardOptions,
): Promise<TokenBlueprint> {
  const iconFile = options?.iconFile ?? null;
  const forceIconUploadFlow = Boolean(options?.forceIconUploadFlow);

  // 1) まず通常の payload を組み立てる
  const payload = buildUpdatePayloadFromCardVm(blueprint, cardVm);

  // ★ 画像がある場合、ここで iconId を更新してはいけない（PUT前に確定してしまう）
  // - UI 側で "id/icon" を入れていても無視して、あとで objectPath を確定させる
  if (iconFile) {
    delete (payload as any).iconId;
  }

  // デバッグ用: 更新リクエストペイロードを確認
  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] request payload:",
    {
      id: blueprint.id,
      payload,
      hasIconFile: Boolean(iconFile),
      iconFile: iconFile
        ? { name: iconFile.name, type: iconFile.type, size: iconFile.size }
        : null,
      forceIconUploadFlow,
    },
  );

  // 2) まず update（バックエンドが iconUpload を返すならここで受け取る）
  const updated = await updateTokenBlueprint(blueprint.id, payload);

  // 3) iconFile が無いならここで終了
  if (!iconFile && !forceIconUploadFlow) {
    return updated;
  }

  // 4) update レスポンスから iconUpload を読む（backend 側が返す前提）
  const iconUpload = (updated as any)?.iconUpload as SignedIconUpload | undefined;

  const uploadUrl = String(iconUpload?.uploadUrl ?? "").trim();
  const objectPath = String(iconUpload?.objectPath ?? "").trim();
  const signedContentType = String(iconUpload?.contentType ?? "").trim();

  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] iconUpload from update:",
    {
      id: (updated as any)?.id,
      iconUpload,
      uploadUrlPresent: Boolean(uploadUrl),
      objectPathPresent: Boolean(objectPath),
      signedContentType,
    },
  );

  // iconFile が無い（forceのみ）なら PUT はできないので終了
  if (!iconFile) {
    return updated;
  }

  // iconUpload が無い場合はアップロードできない（backend 側の返却条件 or env不足）
  if (!uploadUrl || !objectPath) {
    // eslint-disable-next-line no-console
    console.warn(
      "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon upload skipped: iconUpload is missing on update response.",
      { id: (updated as any)?.id, iconUpload },
    );
    return updated;
  }

  // 5) PUT（署名付きURLへブラウザから直接アップロード）
  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon PUT start",
    {
      id: (updated as any)?.id,
      objectPath,
      file: { name: iconFile.name, type: iconFile.type, size: iconFile.size },
      signedContentType,
    },
  );

  await putFileToSignedUrl(uploadUrl, iconFile, signedContentType);

  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon PUT success",
    {
      id: (updated as any)?.id,
      objectPath,
    },
  );

  // 6) iconId を objectPath で確定（= DBに紐付け）
  const attached = await attachTokenBlueprintIcon({
    tokenBlueprintId: String((updated as any)?.id ?? blueprint.id),
    objectPath,
  });

  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon attach success",
    {
      id: (attached as any)?.id,
      iconId: (attached as any)?.iconId,
      iconUrl: (attached as any)?.iconUrl,
    },
  );

  return attached;
}
