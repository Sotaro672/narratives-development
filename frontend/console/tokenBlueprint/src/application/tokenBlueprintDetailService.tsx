// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type {
  TokenBlueprint,
  ContentFile,
  ContentType,
  ContentVisibility,
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
  const tokenBlueprintId = id;
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
 * 正仕様:
 * - iconUrl は Firebase Storage downloadURL
 * - iconObjectPath は Firebase Storage objectPath
 * - iconFileName / iconContentType / iconSize も保存する
 * - contentFiles[].url は Firebase Storage downloadURL
 * - contentFiles[].objectPath は Firebase Storage objectPath
 * - contentFiles[].name / size も backend ContentFile として保存する
 *
 * update 対象:
 * - name
 * - symbol
 * - assigneeId
 * - iconUrl / iconObjectPath / iconFileName / iconContentType / iconSize
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
    iconObjectPath: fields.iconObjectPath ?? blueprint.iconObjectPath,
    iconFileName: fields.iconFileName ?? blueprint.iconFileName,
    iconContentType: fields.iconContentType ?? blueprint.iconContentType,
    iconSize: fields.iconSize ?? blueprint.iconSize,

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
 *    1) iconUrl / iconObjectPath 系を除外して通常 update
 *    2) update 後の tokenBlueprintId / companyId を使って Firebase Storage へ iconFile を直接 upload
 *    3) downloadURL / objectPath / fileName / contentType / size を icon 情報として再 update
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
    delete payload.iconObjectPath;
    delete payload.iconFileName;
    delete payload.iconContentType;
    delete payload.iconSize;
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
    iconObjectPath: uploaded.objectPath,
    iconFileName: uploaded.fileName,
    iconContentType: uploaded.contentType,
    iconSize: uploaded.size,
  } as any);
}

/**
 * レスポンスを正として contentFiles は backend ContentFile struct に合わせて送信する。
 *
 * 正仕様:
 * - id: string
 * - name: string
 * - type: "image" | "video" | "pdf" | "document"
 * - contentType: string
 * - visibility: "private" | "public"
 * - createdAt: ISO string
 * - createdBy: string
 * - updatedAt: ISO string
 * - updatedBy: string
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage objectPath
 * - size: number
 */
function normalizeContentFilesForSend(
  input: ContentFileForSend[],
): ContentFileDTO[] {
  return input
    .map((x) => {
      const nowIso = new Date().toISOString();

      const id = String(x.id ?? "");
      const name = String(x.name ?? "");
      const type = normalizeContentType(x.type);
      const contentType =
        String(x.contentType ?? "") || "application/octet-stream";
      const visibility = normalizeContentVisibility(x.visibility);
      const createdAt = toIsoStringOrNow(x.createdAt ?? nowIso);
      const createdBy = String(x.createdBy ?? "");
      const updatedAt = toIsoStringOrNow(x.updatedAt ?? nowIso);
      const updatedBy = String(x.updatedBy ?? "");
      const url = String(x.url ?? "");
      const objectPath = String(x.objectPath ?? "");

      const rawSize = Number(x.size ?? 0);
      const size = Number.isFinite(rawSize) && rawSize >= 0 ? rawSize : 0;

      return {
        id,
        name,
        type,
        contentType,
        visibility,
        createdAt,
        createdBy,
        updatedAt,
        updatedBy,
        url,
        objectPath,
        size,
      };
    })
    .filter((x) => {
      return Boolean(
        x.id &&
          x.name &&
          x.type &&
          x.url &&
          x.objectPath &&
          x.createdAt &&
          x.createdBy &&
          x.updatedAt &&
          x.updatedBy,
      );
    });
}

function normalizeContentType(value: unknown): ContentType {
  const raw = String(value ?? "").toLowerCase();

  if (
    raw === "image" ||
    raw === "video" ||
    raw === "pdf" ||
    raw === "document"
  ) {
    return raw;
  }

  return "document";
}

function normalizeContentVisibility(value: unknown): ContentVisibility {
  const raw = String(value ?? "").toLowerCase();

  if (raw === "public" || raw === "private") {
    return raw;
  }

  return "private";
}

function toIsoStringOrNow(value: unknown): string {
  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) {
      return new Date().toISOString();
    }

    return value.toISOString();
  }

  const raw = String(value ?? "");
  if (!raw) {
    return new Date().toISOString();
  }

  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return new Date().toISOString();
  }

  return parsed.toISOString();
}

function getCardFields(cardVm: any): Partial<TokenBlueprint> & {
  iconFile?: File | null;
  contentFiles?: ContentFileForSend[];
} {
  return cardVm?.fields ?? cardVm ?? {};
}