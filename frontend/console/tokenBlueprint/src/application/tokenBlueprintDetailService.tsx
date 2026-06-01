// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type {
  TokenBlueprint,
  ContentFile,
} from "../domain/entity/tokenBlueprint";
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
export async function fetchTokenBlueprintDetail(
  id: string,
): Promise<TokenBlueprint> {
  const tokenBlueprintId = id.trim();
  if (!tokenBlueprintId) {
    throw new Error("id is empty");
  }

  return fetchTokenBlueprintById(tokenBlueprintId);
}

/**
 * createdAt を yyyy/mm/dd にフォーマット
 *
 * API レスポンスの日時は ISO string を正とする。
 */
export function formatCreatedAt(raw: string): string {
  return safeDateLabelJa(raw, "");
}

/**
 * TokenBlueprintCard の VM から UpdateTokenBlueprintPayload を組み立てる
 *
 * 正レスポンス:
 * - iconUrl は Firebase Storage downloadURL
 * - contentFiles[].url は Firebase Storage downloadURL
 *
 * objectPath / name / size は廃止済み。
 *
 * update 対象:
 * - name
 * - symbol
 * - assigneeId
 * - iconUrl
 * - contentFiles
 *
 * update 対象外:
 * - brandId / brandName
 * - companyId
 * - minted
 * - metadataUri
 * - createdAt / createdBy / createdByName
 * - updatedAt / updatedBy / updatedByName
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: any,
): Record<string, any> {
  const fields = getCardFields(cardVm);

  const iconUrlRaw = fields.iconUrl ?? blueprint.iconUrl;
  const iconUrl =
    typeof iconUrlRaw === "string" && iconUrlRaw.startsWith("blob:")
      ? undefined
      : iconUrlRaw;

  return {
    name: fields.name ?? blueprint.name,
    symbol: fields.symbol ?? blueprint.symbol,
    assigneeId: fields.assigneeId ?? blueprint.assigneeId,
    iconUrl,
    contentFiles: normalizeContentFilesForSend(
      fields.contentFiles ?? blueprint.contentFiles ?? [],
    ),
  };
}

type UpdateFromCardOptions = {
  iconFile?: File | null;
};

type ContentFileForSend = Partial<ContentFile> & Partial<ContentFileDTO>;

/**
 * TokenBlueprintCard の VM から update API を呼び出し、更新後の TokenBlueprint を返す
 *
 * 方針:
 * - iconFile がある場合:
 *    1) iconUrl を除外して通常 update
 *    2) update 後の tokenBlueprintId / companyId を使って Firebase Storage へ iconFile を直接 upload
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
    delete payload.iconUrl;
  }

  const updated = await updateTokenBlueprint(blueprint.id, payload as any);

  if (!iconFile) {
    return updated;
  }

  const tokenBlueprintId = updated.id || blueprint.id;
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprint.id is empty after update.");
  }

  const companyId = updated.companyId || blueprint.companyId;
  if (!companyId) {
    throw new Error("companyId is required before uploading token blueprint icon.");
  }

  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({
    companyId,
    tokenBlueprintId,
    file: iconFile,
  });

  return updateTokenBlueprint(tokenBlueprintId, {
    iconUrl: uploaded.downloadUrl,
  } as any);
}

/**
 * レスポンスを正として contentFiles は backend ContentFile struct に絞って送信する。
 *
 * 正レスポンス:
 * - id: string
 * - type: string
 * - contentType: string
 * - visibility: string
 * - createdAt: ISO string
 * - createdBy: string
 * - updatedAt: ISO string
 * - updatedBy: string
 * - url: Firebase Storage downloadURL
 *
 * objectPath / name / size は廃止済み。
 */
function normalizeContentFilesForSend(
  input: ContentFileForSend[],
): ContentFileDTO[] {
  return input
    .map((x) => ({
      id: String(x.id ?? "").trim(),
      type: String(x.type ?? "").trim(),
      contentType: String(x.contentType ?? "").trim(),
      visibility: String(x.visibility ?? "private").trim(),
      createdAt: x.createdAt,
      createdBy: x.createdBy,
      updatedAt: x.updatedAt,
      updatedBy: x.updatedBy,
      url: String(x.url ?? "").trim(),
    }))
    .filter((x) => Boolean(x.id && x.type && x.url));
}

function getCardFields(cardVm: any): Partial<TokenBlueprint> & {
  iconFile?: File | null;
  contentFiles?: ContentFileForSend[];
} {
  return cardVm?.fields ?? cardVm ?? {};
}