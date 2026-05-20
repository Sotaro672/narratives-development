// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.mapper.ts

import type {
  TokenBlueprint,
  ContentFile,
} from "../../domain/entity/tokenBlueprint";

/**
 * API レスポンスの contentFiles を domain の ContentFile に変換する。
 *
 * 正レスポンス:
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage object path
 * - createdAt / updatedAt: ISO string
 *
 * backend domain では ContentFile.url は必須 validate されていないため、
 * frontend 表示で必要な url が存在するものだけを採用する。
 */
function normalizeContentFiles(contentFiles: ContentFile[]): ContentFile[] {
  return contentFiles.filter((file) => file.id && file.objectPath && file.url);
}

export function normalizeTokenBlueprint(raw: TokenBlueprint): TokenBlueprint {
  return {
    id: raw.id,
    name: raw.name,
    symbol: raw.symbol,

    brandId: raw.brandId,
    brandName: raw.brandName,
    companyId: raw.companyId,

    description: raw.description,

    minted: raw.minted,

    contentFiles: normalizeContentFiles(raw.contentFiles ?? []),

    assigneeId: raw.assigneeId,
    assigneeName: raw.assigneeName,

    createdAt: raw.createdAt,
    createdBy: raw.createdBy,
    createdByName: raw.createdByName,

    updatedAt: raw.updatedAt,
    updatedBy: raw.updatedBy,
    updatedByName: raw.updatedByName,

    metadataUri: raw.metadataUri,
    iconUrl: raw.iconUrl,
  };
}

export function normalizePageResult(raw: {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
}): {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
} {
  return {
    items: raw.items.map((item) => normalizeTokenBlueprint(item)),
    totalCount: raw.totalCount,
    totalPages: raw.totalPages,
    page: raw.page,
    perPage: raw.perPage,
  };
}