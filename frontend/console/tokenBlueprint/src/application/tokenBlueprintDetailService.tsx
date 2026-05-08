// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";
import type { ContentFileDTO } from "../infrastructure/dto/tokenBlueprint.dto";

import { safeDateLabelJa } from "../../../shell/src/shared/util/dateJa";

import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import { uploadTokenBlueprintIconToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

/**
 * 詳細取得（リポジトリのラッパー）
 */
export async function fetchTokenBlueprintDetail(id: string): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error("id is empty");
  }

  return fetchTokenBlueprintById(id);
}

/**
 * createdAt を yyyy/mm/dd にフォーマット
 */
export function formatCreatedAt(raw: unknown): string {
  return safeDateLabelJa(
    typeof raw === "string" ? raw : raw instanceof Date ? raw.toISOString() : "",
    "",
  );
}

/**
 * TokenBlueprintCard の VM から UpdateTokenBlueprintPayload を組み立てる
 *
 * レスポンスを正として扱う:
 * - icon は iconUrl を更新対象とする
 * - contentFiles は ContentFileDTO[] として更新する
 * - createdById / updatedBy / *_Name などレスポンスの名に寄せる
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: any,
): Record<string, any> {
  const vmAny: any = cardVm || {};
  const fields: any = vmAny.fields ?? vmAny ?? {};

  const stringOrUndefined = (v: unknown): string | undefined =>
    typeof v === "string" ? v : undefined;

  const iconUrlRaw =
    typeof fields.iconUrl === "string"
      ? fields.iconUrl
      : typeof (blueprint as any)?.iconUrl === "string"
        ? String((blueprint as any).iconUrl)
        : undefined;

  const iconUrl = iconUrlRaw && iconUrlRaw.startsWith("blob:") ? undefined : iconUrlRaw;

  const payload: Record<string, any> = {
    name: stringOrUndefined(fields.name ?? blueprint.name),
    symbol: stringOrUndefined(fields.symbol ?? blueprint.symbol),
    brandId: stringOrUndefined(fields.brandId ?? blueprint.brandId),
    description: stringOrUndefined(fields.description ?? blueprint.description),
    assigneeId: stringOrUndefined(fields.assigneeId ?? blueprint.assigneeId),
    iconUrl,
    contentFiles: normalizeContentFilesForSend(
      fields.contentFiles ?? (blueprint as any)?.contentFiles ?? [],
    ),
  };

  if (fields.metadataUri !== undefined) {
    payload.metadataUri = String(fields.metadataUri ?? "");
  }

  if (fields.minted !== undefined) {
    payload.minted = Boolean(fields.minted);
  }

  return payload;
}

type UpdateFromCardOptions = {
  iconFile?: File | null;

  /**
   * GCS signed URL 時代の互換オプション。
   * Firebase Storage 移行後は iconFile がある場合のみ upload するため、
   * この値単体では upload flow を起動しない。
   */
  forceIconUploadFlow?: boolean;
};

/**
 * TokenBlueprintCard の VM から update API を呼び出し、更新後の TokenBlueprint を返す
 *
 * 方針:
 * - iconFile がある場合:
 *    1) iconUrl を除外して通常 update
 *    2) update 後の tokenBlueprintId を使って Firebase Storage へ iconFile を直接 upload
 *    3) downloadURL を iconUrl として再 update
 * - iconFile が無い場合:
 *    通常 update のみ
 */
export async function updateTokenBlueprintFromCard(
  blueprint: TokenBlueprint,
  cardVm: any,
  options?: UpdateFromCardOptions,
): Promise<TokenBlueprint> {
  const iconFile =
    options?.iconFile ??
    (cardVm?.iconFile as File | null | undefined) ??
    (cardVm?.fields?.iconFile as File | null | undefined) ??
    null;

  const payload = buildUpdatePayloadFromCardVm(blueprint, cardVm);

  if (iconFile) {
    delete (payload as any).iconUrl;
  }

  const updated = await updateTokenBlueprint(
    blueprint.id,
    payload as any,
  );

  if (!iconFile) {
    return updated;
  }

  const tokenBlueprintId = String((updated as any)?.id ?? blueprint.id ?? "").trim();
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprint.id is empty after update.");
  }

  const companyId = String(
    (updated as any)?.companyId ??
      (blueprint as any)?.companyId ??
      (cardVm as any)?.companyId ??
      (cardVm as any)?.fields?.companyId ??
      "",
  ).trim();

  if (!companyId) {
    throw new Error("companyId is required before uploading token blueprint icon.");
  }

  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({
    companyId,
    tokenBlueprintId,
    file: iconFile,
  });

  const attached = await updateTokenBlueprint(
    tokenBlueprintId,
    {
      iconUrl: uploaded.downloadUrl,
    } as any,
  );

  return attached;
}

/**
 * レスポンスを正として contentFiles は object[] のみ扱う
 */
function normalizeContentFilesForSend(input: unknown): ContentFileDTO[] {
  const arr = Array.isArray(input) ? input : [];

  return (arr as any[])
    .filter((x) => x && typeof x === "object")
    .map(
      (x): ContentFileDTO =>
        ({
          id: String(x.id ?? ""),
          name: String(x.name ?? ""),
          type: String(x.type ?? ""),
          contentType: String(x.contentType ?? ""),
          size: Number(x.size ?? 0) || 0,
          objectPath: String(x.objectPath ?? ""),
          visibility: String(x.visibility ?? "") || "private",
          createdAt: x.createdAt != null ? String(x.createdAt) : x.createdAt,
          createdBy: x.createdBy != null ? String(x.createdBy) : x.createdBy,
          updatedAt: x.updatedAt != null ? String(x.updatedAt) : x.updatedAt,
          updatedBy: x.updatedBy != null ? String(x.updatedBy) : x.updatedBy,
          url: x.url != null ? String(x.url) : x.url,
        } as any),
    )
    .filter((x) => Boolean((x as any).id));
}