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
 * - contentFiles[].objectPath は Firebase Storage object path
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
 * レスポンスを正として contentFiles は ContentFileDTO[] として扱う。
 *
 * 正レスポンス:
 * - id: string
 * - name: string
 * - type: string
 * - contentType: string
 * - size: number
 * - objectPath: string
 * - visibility: string
 * - createdAt: ISO string
 * - createdBy: string
 * - updatedAt: ISO string
 * - updatedBy: string
 * - url: Firebase Storage downloadURL
 */
function normalizeContentFilesForSend(input: ContentFileDTO[]): ContentFileDTO[] {
  return input
    .map((x) => ({
      id: x.id,
      name: x.name,
      type: x.type,
      contentType: x.contentType,
      size: x.size,
      objectPath: x.objectPath,
      visibility: x.visibility,
      createdAt: x.createdAt,
      createdBy: x.createdBy,
      updatedAt: x.updatedAt,
      updatedBy: x.updatedBy,
      url: x.url,
    }))
    .filter((x) => Boolean(x.id && x.objectPath && x.url));
}

function getCardFields(cardVm: any): Partial<TokenBlueprint> & {
  iconFile?: File | null;
  contentFiles?: ContentFileDTO[];
} {
  return cardVm?.fields ?? cardVm ?? {};
}