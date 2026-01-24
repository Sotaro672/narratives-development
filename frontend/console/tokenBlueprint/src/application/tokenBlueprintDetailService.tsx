// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";

import type { SignedIconUpload } from "../../../shell/src/shared/types/tokenBlueprint";
import type { ContentFileDTO } from "../infrastructure/dto/tokenBlueprint.dto";

import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { putFileToSignedUrl } from "../infrastructure/upload/signedUrlPut";

/**
 * 詳細取得（リポジトリのラッパー）
 */
export async function fetchTokenBlueprintDetail(id: string): Promise<TokenBlueprint> {
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

  const d = raw instanceof Date ? raw : new Date(raw as any);
  if (isNaN(d.getTime())) return "";

  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}/${mm}/${dd}`;
}

/**
 * TokenBlueprintCard の VM から UpdateTokenBlueprintPayload を組み立てる
 *
 * 重要:
 * - entity.go 正: icon は iconUrl を更新対象とする（iconId は UI から送らない）
 * - entity.go 正: contentFiles は「ContentFileDTO[]」として更新する
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: any,
): Record<string, any> {
  const vmAny: any = cardVm || {};
  const fields: any = vmAny.fields ?? vmAny ?? {};

  const trimOrUndefined = (v: unknown): string | undefined =>
    typeof v === "string" ? v.trim() : undefined;

  // iconUrl は「blob:」を送らない（プレビュー専用URLのため）
  const iconUrlRaw =
    typeof fields.iconUrl === "string"
      ? fields.iconUrl.trim()
      : typeof (blueprint as any)?.iconUrl === "string"
        ? String((blueprint as any).iconUrl).trim()
        : undefined;

  const iconUrl =
    iconUrlRaw && iconUrlRaw.startsWith("blob:") ? undefined : iconUrlRaw;

  const payload: Record<string, any> = {
    name: trimOrUndefined(fields.name ?? blueprint.name),
    symbol: trimOrUndefined(fields.symbol ?? blueprint.symbol),
    brandId: trimOrUndefined(fields.brandId ?? blueprint.brandId),
    description: trimOrUndefined(fields.description ?? blueprint.description),
    assigneeId: trimOrUndefined(fields.assigneeId ?? blueprint.assigneeId),

    // iconUrl は更新対象（imageUrl_resolver に委譲）
    iconUrl,

    // contentFiles は ContentFileDTO[] を期待（string[] は許容しない）
    contentFiles: normalizeContentFilesForSend(
      fields.contentFiles ?? (blueprint as any)?.contentFiles ?? [],
    ),
  };

  // minted / metadataUri が UI にあるなら任意で反映
  if (fields.metadataUri !== undefined) {
    payload.metadataUri = String(fields.metadataUri ?? "").trim();
  }
  if (fields.minted !== undefined) payload.minted = Boolean(fields.minted);

  // iconId は entity.go 正として UI からは送らない（事故防止）
  // UI が fields.iconId を持っていても無視する

  return payload;
}

type UpdateFromCardOptions = {
  /**
   * ★ 選択されたアイコンファイル（あれば Signed URL PUT → iconUrl 反映まで行う）
   */
  iconFile?: File | null;

  /**
   * ★ デバッグ用途: 強制的に iconUpload を見たい/試したい場合に true
   * （backend 実装が update レスポンスでも iconUpload を返す想定）
   */
  forceIconUploadFlow?: boolean;
};

/**
 * TokenBlueprintCard の VM から update API を呼び出し、更新後の TokenBlueprint を返す
 *
 * 方針:
 * - iconFile がある場合:
 *    1) update（iconUrl/contentFiles 等。ただし iconUrl は PUT 後に確定したいので一旦消す）
 *       ★ update で iconUpload を返すために hasIconFile/iconContentType を送る
 *    2) update レスポンスの iconUpload を使って PUT
 *    3) publicUrl を iconUrl として再 update（確定）
 * - iconFile が無い場合: 通常 update のみ
 */
export async function updateTokenBlueprintFromCard(
  blueprint: TokenBlueprint,
  cardVm: any,
  options?: UpdateFromCardOptions,
): Promise<TokenBlueprint> {
  // ★ A案：呼び出し側が options.iconFile を渡し忘れても vm/iconFile から拾う
  const iconFile =
    options?.iconFile ??
    (cardVm?.iconFile as File | null | undefined) ??
    (cardVm?.fields?.iconFile as File | null | undefined) ??
    null;

  const forceIconUploadFlow = Boolean(options?.forceIconUploadFlow);

  // 1) まず payload を組む
  const payload = buildUpdatePayloadFromCardVm(blueprint, cardVm);

  // iconFile がある場合、PUT 前に iconUrl を確定させない（publicUrl がまだ無い）
  if (iconFile) {
    delete (payload as any).iconUrl;
  }

  // eslint-disable-next-line no-console
  console.log("[tokenBlueprintDetailService.updateTokenBlueprintFromCard] request payload:", {
    id: blueprint.id,
    payload,
    hasIconFile: Boolean(iconFile),
    iconFile: iconFile ? { name: iconFile.name, type: iconFile.type, size: iconFile.size } : null,
    forceIconUploadFlow,
  });

  // 2) update
  // ★ UpdateTokenBlueprintOptions に存在するのは hasIconFile/iconContentType のみ（iconFileName は渡さない）
  const updated = await updateTokenBlueprint(
    blueprint.id,
    payload as any,
    iconFile
      ? {
          hasIconFile: true,
          iconContentType: String(iconFile.type ?? "").trim() || "application/octet-stream",
        }
      : forceIconUploadFlow
        ? {
            hasIconFile: true,
            iconContentType: "application/octet-stream",
          }
        : undefined,
  );

  // 3) iconFile が無い & force も無いなら終了
  if (!iconFile && !forceIconUploadFlow) return updated;

  // 4) update レスポンスから iconUpload を読む（repo ではなく shared/types を正とする）
  const iconUpload = (updated as any)?.iconUpload as SignedIconUpload | undefined;

  const uploadUrl = String(iconUpload?.uploadUrl ?? "").trim();
  const publicUrl = String(iconUpload?.publicUrl ?? "").trim();
  const signedContentType = String(iconUpload?.contentType ?? "").trim();

  // eslint-disable-next-line no-console
  console.log("[tokenBlueprintDetailService.updateTokenBlueprintFromCard] iconUpload from update:", {
    id: (updated as any)?.id,
    iconUpload,
    uploadUrlPresent: Boolean(uploadUrl),
    publicUrlPresent: Boolean(publicUrl),
    signedContentType,
  });

  // iconFile が無い（forceのみ）なら PUT はできないので終了
  if (!iconFile) return updated;

  // iconUpload が無い場合はアップロードできない（backend 側の返却条件 or env不足 or repo が未対応）
  if (!uploadUrl || !publicUrl) {
    // eslint-disable-next-line no-console
    console.warn(
      "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon upload skipped: iconUpload is missing on update response.",
      { id: (updated as any)?.id, iconUpload },
    );
    return updated;
  }

  // 5) PUT（署名付きURLへブラウザから直接アップロード）
  // eslint-disable-next-line no-console
  console.log("[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon PUT start", {
    id: (updated as any)?.id,
    file: { name: iconFile.name, type: iconFile.type, size: iconFile.size },
    signedContentType,
  });

  await putFileToSignedUrl(uploadUrl, iconFile, signedContentType);

  // eslint-disable-next-line no-console
  console.log("[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon PUT success", {
    id: (updated as any)?.id,
  });

  // 6) icon を publicUrl で確定（= DBに紐付け）
  // - backend 側 imageUrl_resolver が publicUrl を保存用に加工し、加工後URLを返す想定
  const attached = await updateTokenBlueprint(
    String((updated as any)?.id ?? blueprint.id),
    {
      iconUrl: publicUrl,
    } as any,
  );

  // eslint-disable-next-line no-console
  console.log("[tokenBlueprintDetailService.updateTokenBlueprintFromCard] icon attach success", {
    id: (attached as any)?.id,
    iconId: (attached as any)?.iconId,
    iconUrl: (attached as any)?.iconUrl,
  });

  return attached;
}

// ---------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------

/**
 * entity.go 正: contentFiles は ContentFileDTO[] を送る
 * - UI が string[] を持っている/混在している可能性があるため、ここで吸収して DTO[] に正規化する
 */
function normalizeContentFilesForSend(input: unknown): ContentFileDTO[] {
  const arr = Array.isArray(input) ? input : [];

  // string[] の場合は最低限の形に落とす（backend 側で name/type/url など必須なら UI 側で揃える必要あり）
  if (arr.length > 0 && typeof arr[0] === "string") {
    return (arr as string[])
      .map((s) => String(s ?? "").trim())
      .filter(Boolean)
      .map(
        (id): ContentFileDTO =>
          ({
            id,
            name: id, // 最低限のプレースホルダ（UI で正しい値を持つならここを置換）
            type: "document",
            contentType: "application/octet-stream",
            objectPath: id,
            visibility: "private",
            size: 0,
          } as any),
      );
  }

  // object[] の場合は trim しつつ ContentFileDTO として返す
  return (arr as any[])
    .filter((x) => x && typeof x === "object")
    .map(
      (x): ContentFileDTO =>
        ({
          ...x,
          id: String(x.id ?? "").trim(),
          name: String(x.name ?? "").trim(),
          type: String(x.type ?? "").trim(),
          contentType: String(x.contentType ?? "").trim(),
          objectPath: String(x.objectPath ?? "").trim(),
          visibility: String(x.visibility ?? "").trim() || "private",
          size: Number(x.size ?? 0) || 0,
          createdBy: x.createdBy != null ? String(x.createdBy).trim() : x.createdBy,
          updatedBy: x.updatedBy != null ? String(x.updatedBy).trim() : x.updatedBy,
        } as any),
    )
    .filter((x) => Boolean((x as any).id));
}
