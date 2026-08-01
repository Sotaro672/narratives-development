// frontend/console/shell/src/features/tokenBlueprint/infrastructure/dto/tokenBlueprint.mapper.ts

import {
  normalizeContentType,
  normalizeTokenBlueprintMimeType,
} from "../../../../shared/types/tokenBlueprint";

import type {
  ContentFile,
  TokenBlueprint,
} from "../../../../shared/types/tokenBlueprint";

import type {
  PageResult,
} from "../../../../shared/types/common/common";

import type {
  ContentFileDTO,
  TokenBlueprintDTO,
} from "./tokenBlueprint.dto";

type RawRecord = Record<string, unknown>;

function asRecord(
  value: unknown,
): RawRecord {
  return (
    value &&
    typeof value === "object"
      ? value as RawRecord
      : {}
  );
}

function toStringValue(
  value: unknown,
  fallback = "",
): string {
  if (value == null) {
    return fallback;
  }

  return String(value);
}

function toNullableStringValue(
  value: unknown,
): string | null {
  if (value == null) {
    return null;
  }

  const stringValue =
    String(value);

  return stringValue || null;
}

function toNumberValue(
  value: unknown,
  fallback = 0,
): number {
  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return value;
  }

  if (typeof value === "string") {
    const numberValue =
      Number(value);

    if (
      Number.isFinite(
        numberValue,
      )
    ) {
      return numberValue;
    }
  }

  return fallback;
}

function normalizeDateString(
  value: unknown,
): string {
  if (value instanceof Date) {
    return Number.isNaN(
      value.getTime(),
    )
      ? ""
      : value.toISOString();
  }

  return toStringValue(
    value,
  );
}

/**
 * APIレスポンスのcontentFilesを、
 * shared/types/tokenBlueprint.tsで定義された
 * ContentFileへ変換する。
 *
 * backend正仕様:
 * - id: ContentFile ID
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage object path
 * - name: 元ファイル名または表示名
 * - isPublic: 公開状態
 * - size: byte size
 * - createdAt / updatedAt: ISO string
 *
 * id、createdBy、updatedBy、isPublicは
 * Backend DTOの型を正とし、値の変換や補完を行わない。
 *
 * frontendでの表示・差し替え・削除に必要な
 * id / url / objectPathが存在するものだけを採用する。
 */
function normalizeContentFile(
  raw: unknown,
): ContentFile | null {
  const obj =
    asRecord(raw);

  /**
   * IDはBackend DTOの値をそのまま使用する。
   */
  const id =
    obj.id as ContentFileDTO["id"];

  const name =
    toStringValue(
      obj.name,
    );

  const type =
    normalizeContentType(
      obj.type,
    );

  const contentType =
    normalizeTokenBlueprintMimeType(
      obj.contentType,
    );

  const url =
    toStringValue(
      obj.url,
    );

  const objectPath =
    toStringValue(
      obj.objectPath,
    );

  /**
   * booleanはBackend DTOの値をそのまま使用する。
   */
  const isPublic =
    obj.isPublic as ContentFileDTO["isPublic"];

  const size =
    toNumberValue(
      obj.size,
      0,
    );

  const createdAt =
    normalizeDateString(
      obj.createdAt,
    );

  /**
   * Member IDはBackend DTOの値をそのまま使用する。
   */
  const createdBy =
    obj.createdBy as ContentFileDTO["createdBy"];

  const updatedAt =
    normalizeDateString(
      obj.updatedAt,
    );

  /**
   * Member IDはBackend DTOの値をそのまま使用する。
   */
  const updatedBy =
    obj.updatedBy as ContentFileDTO["updatedBy"];

  if (
    !id ||
    !url ||
    !objectPath
  ) {
    return null;
  }

  return {
    id,
    name,
    type,
    contentType,
    url,
    objectPath,
    isPublic,

    size:
      Number.isFinite(size) &&
      size >= 0
        ? size
        : 0,

    createdAt,
    createdBy,
    updatedAt,
    updatedBy,
  };
}

function normalizeContentFiles(
  contentFiles: unknown,
): ContentFile[] {
  if (
    !Array.isArray(
      contentFiles,
    )
  ) {
    return [];
  }

  return contentFiles
    .map((file) => {
      return normalizeContentFile(
        file,
      );
    })
    .filter(
      (
        file,
      ): file is ContentFile => {
        return file !== null;
      },
    );
}

export function normalizeTokenBlueprint(
  raw: unknown,
): TokenBlueprint {
  const obj =
    asRecord(raw);

  /**
   * TokenBlueprint IDはBackend DTOの値を
   * そのまま使用する。
   */
  const id =
    obj.id as TokenBlueprintDTO["id"];

  const name =
    toStringValue(
      obj.name,
    );

  const symbol =
    toStringValue(
      obj.symbol,
    );

  /**
   * 各IDはBackend DTOの値をそのまま使用する。
   */
  const brandId =
    obj.brandId as TokenBlueprintDTO["brandId"];

  const brandName =
    toStringValue(
      obj.brandName,
    );

  const companyId =
    obj.companyId as TokenBlueprintDTO["companyId"];

  const description =
    toStringValue(
      obj.description,
    );

  const iconUrl =
    toNullableStringValue(
      obj.iconUrl,
    );

  const iconObjectPath =
    toNullableStringValue(
      obj.iconObjectPath,
    );

  const iconFileName =
    toNullableStringValue(
      obj.iconFileName,
    );

  const iconContentType =
    toNullableStringValue(
      obj.iconContentType,
    );

  const iconSizeRaw =
    obj.iconSize;

  const iconSize =
    iconSizeRaw == null
      ? null
      : toNumberValue(
          iconSizeRaw,
          0,
        );

  const contentFiles =
    normalizeContentFiles(
      obj.contentFiles as
        | ContentFileDTO[]
        | undefined,
    );

  const assigneeId =
    obj.assigneeId as TokenBlueprintDTO["assigneeId"];

  const assigneeName =
    toStringValue(
      obj.assigneeName,
    );

  /**
   * booleanはBackend DTOの値をそのまま使用する。
   */
  const minted =
    obj.minted as TokenBlueprintDTO["minted"];

  const createdAt =
    normalizeDateString(
      obj.createdAt,
    );

  const createdBy =
    obj.createdBy as TokenBlueprintDTO["createdBy"];

  const createdByName =
    toStringValue(
      obj.createdByName,
    );

  const updatedAt =
    normalizeDateString(
      obj.updatedAt,
    );

  const updatedBy =
    obj.updatedBy as TokenBlueprintDTO["updatedBy"];

  const updatedByName =
    toStringValue(
      obj.updatedByName,
    );

  const deletedAt =
    toNullableStringValue(
      obj.deletedAt,
    );

  const deletedBy =
    obj.deletedBy as TokenBlueprintDTO["deletedBy"];

  const metadataUri =
    toStringValue(
      obj.metadataUri,
    );

  return {
    id,
    name,
    symbol,

    brandId,
    brandName,
    companyId,

    description,

    iconUrl,
    iconObjectPath,
    iconFileName,
    iconContentType,
    iconSize,

    minted,

    contentFiles,

    assigneeId,
    assigneeName,

    createdAt,
    createdBy,
    createdByName,

    updatedAt,
    updatedBy,
    updatedByName,

    deletedAt,
    deletedBy,

    metadataUri,
  };
}

export function normalizePageResult(
  raw: unknown,
): PageResult<TokenBlueprint> {
  const obj =
    asRecord(raw);

  const itemsRaw =
    obj.items as
      | TokenBlueprintDTO[]
      | undefined;

  return {
    items:
      Array.isArray(itemsRaw)
        ? itemsRaw.map(
            (item) => {
              return normalizeTokenBlueprint(
                item,
              );
            },
          )
        : [],

    totalCount:
      toNumberValue(
        obj.totalCount,
        0,
      ),

    totalPages:
      toNumberValue(
        obj.totalPages,
        0,
      ),

    page:
      toNumberValue(
        obj.page,
        1,
      ),

    perPage:
      toNumberValue(
        obj.perPage,
        50,
      ),
  };
}