// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintDetailService.tsx

import {
  normalizeContentType,
  normalizeTokenBlueprintMimeType,
} from "../../../shared/types/tokenBlueprint";

import type {
  ContentFile,
  TokenBlueprint,
} from "../../../shared/types/tokenBlueprint";

import type { ContentFileDTO } from "../infrastructure/dto/tokenBlueprint.dto";
import type { UpdateTokenBlueprintPayload } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import { safeDateLabelJa } from "../../../shared/util/dateJa";

import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import { uploadTokenBlueprintIconToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

type UpdateFromCardOptions = {
  iconFile?: File | null;
};

type ContentFileForSend =
  Partial<ContentFile>;

type TokenBlueprintCardFields =
  Partial<Omit<TokenBlueprint, "contentFiles">> & {
    iconFile?: File | null;
    contentFiles?: ContentFileForSend[];
  };

type TokenBlueprintCardVm =
  TokenBlueprintCardFields & {
    fields?: TokenBlueprintCardFields;
  };

/**
 * 詳細取得。
 */
export async function fetchTokenBlueprintDetail(
  id: string,
): Promise<TokenBlueprint> {
  if (!id) {
    throw new Error(
      "id is required",
    );
  }

  return fetchTokenBlueprintById(id);
}

/**
 * createdAtをyyyy/mm/ddへフォーマットする。
 *
 * APIレスポンスの日時はISO stringを正とする。
 */
export function formatCreatedAt(
  raw: string,
): string {
  return safeDateLabelJa(
    raw,
    "",
  );
}

/**
 * TokenBlueprintCardのViewModelから
 * UpdateTokenBlueprintPayloadを組み立てる。
 *
 * 正仕様:
 * - iconUrlはFirebase Storage downloadURL
 * - iconObjectPathはFirebase Storage objectPath
 * - iconFileName / iconContentType / iconSizeも保存する
 * - contentFiles[].urlはFirebase Storage downloadURL
 * - contentFiles[].objectPathはFirebase Storage objectPath
 * - contentFiles[].isPublicはboolean
 * - contentFiles[].name / sizeも保存する
 *
 * 更新対象:
 * - name
 * - symbol
 * - assigneeId
 * - iconUrl
 * - iconObjectPath
 * - iconFileName
 * - iconContentType
 * - iconSize
 * - contentFiles
 *
 * 更新対象外:
 * - brandId
 * - brandName
 * - companyId
 * - minted
 * - metadataUri
 * - createdAt
 * - createdBy
 * - createdByName
 * - updatedAt
 * - updatedBy
 * - updatedByName
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: TokenBlueprintCardVm,
): UpdateTokenBlueprintPayload {
  const fields =
    getCardFields(cardVm);

  const iconUrlRaw =
    fields.iconUrl ??
    blueprint.iconUrl;

  const iconUrl =
    typeof iconUrlRaw === "string" &&
    iconUrlRaw.startsWith("blob:")
      ? undefined
      : iconUrlRaw;

  return {
    name:
      fields.name ??
      blueprint.name,

    symbol:
      fields.symbol ??
      blueprint.symbol,

    assigneeId:
      fields.assigneeId ??
      blueprint.assigneeId,

    iconUrl,

    iconObjectPath:
      fields.iconObjectPath ??
      blueprint.iconObjectPath,

    iconFileName:
      fields.iconFileName ??
      blueprint.iconFileName,

    iconContentType:
      fields.iconContentType ??
      blueprint.iconContentType,

    iconSize:
      fields.iconSize ??
      blueprint.iconSize,

    contentFiles:
      buildContentFilesForSend(
        fields.contentFiles ??
          blueprint.contentFiles ??
          [],
      ),
  };
}

/**
 * TokenBlueprintCardのViewModelから更新APIを呼び出し、
 * 更新後のTokenBlueprintを返す。
 *
 * iconFileがある場合:
 * 1. icon関連項目を除外して通常更新する
 * 2. Firebase StorageへiconFileをアップロードする
 * 3. アップロード結果をicon情報として再更新する
 *
 * iconFileがない場合:
 * - 通常更新のみを行う
 */
export async function updateTokenBlueprintFromCard(
  blueprint: TokenBlueprint,
  cardVm: TokenBlueprintCardVm,
  options?: UpdateFromCardOptions,
): Promise<TokenBlueprint> {
  const iconFile =
    options?.iconFile ??
    cardVm.iconFile ??
    cardVm.fields?.iconFile ??
    null;

  const payload =
    buildUpdatePayloadFromCardVm(
      blueprint,
      cardVm,
    );

  if (iconFile) {
    delete payload.iconUrl;
    delete payload.iconObjectPath;
    delete payload.iconFileName;
    delete payload.iconContentType;
    delete payload.iconSize;
  }

  const updated =
    await updateTokenBlueprint(
      blueprint.id,
      payload,
    );

  if (!iconFile) {
    return updated;
  }

  const tokenBlueprintId =
    updated.id ||
    blueprint.id;

  if (!tokenBlueprintId) {
    throw new Error(
      "tokenBlueprint.id is required after update",
    );
  }

  const companyId =
    updated.companyId ||
    blueprint.companyId;

  if (!companyId) {
    throw new Error(
      "companyId is required before uploading token blueprint icon",
    );
  }

  const uploaded =
    await uploadTokenBlueprintIconToFirebaseStorage({
      companyId,
      tokenBlueprintId,
      file: iconFile,
    });

  return updateTokenBlueprint(
    tokenBlueprintId,
    {
      iconUrl:
        uploaded.downloadUrl,

      iconObjectPath:
        uploaded.objectPath,

      iconFileName:
        uploaded.fileName,

      iconContentType:
        uploaded.contentType,

      iconSize:
        uploaded.size,
    },
  );
}

/**
 * contentFilesをbackendへ送信するDTOへ変換する。
 *
 * 正仕様:
 * - id: string
 * - name: string
 * - type: "image" | "video" | "pdf" | "document"
 * - contentType: string
 * - isPublic: boolean
 * - createdAt: ISO string
 * - createdBy: string
 * - updatedAt: ISO string
 * - updatedBy: string
 * - url: Firebase Storage downloadURL
 * - objectPath: Firebase Storage objectPath
 * - size: number
 *
 * isPublicはbooleanを正とするため、
 * 文字列変換や公開状態の正規化は行わない。
 */
function buildContentFilesForSend(
  input: ContentFileForSend[],
): ContentFileDTO[] {
  return input
    .map(
      (
        content,
      ): ContentFileDTO | null => {
        if (
          typeof content.isPublic !==
          "boolean"
        ) {
          return null;
        }

        const nowIso =
          new Date().toISOString();

        const id = String(
          content.id ?? "",
        );

        const name = String(
          content.name ?? "",
        );

        const type =
          normalizeContentType(
            content.type,
          );

        const contentType =
          normalizeTokenBlueprintMimeType(
            content.contentType,
          );

        const createdAt =
          toIsoStringOrNow(
            content.createdAt ??
              nowIso,
          );

        const createdBy =
          String(
            content.createdBy ?? "",
          );

        const updatedAt =
          toIsoStringOrNow(
            content.updatedAt ??
              nowIso,
          );

        const updatedBy =
          String(
            content.updatedBy ?? "",
          );

        const url = String(
          content.url ?? "",
        );

        const objectPath =
          String(
            content.objectPath ?? "",
          );

        const rawSize =
          Number(
            content.size ?? 0,
          );

        const size =
          Number.isFinite(rawSize) &&
          rawSize >= 0
            ? rawSize
            : 0;

        if (
          !id ||
          !name ||
          !url ||
          !objectPath ||
          !createdAt ||
          !createdBy ||
          !updatedAt ||
          !updatedBy
        ) {
          return null;
        }

        return {
          id,
          name,
          type,
          contentType,
          isPublic:
            content.isPublic,
          createdAt,
          createdBy,
          updatedAt,
          updatedBy,
          url,
          objectPath,
          size,
        };
      },
    )
    .filter(
      (
        content,
      ): content is ContentFileDTO => {
        return content !== null;
      },
    );
}

function toIsoStringOrNow(
  value: unknown,
): string {
  if (value instanceof Date) {
    if (
      Number.isNaN(
        value.getTime(),
      )
    ) {
      return new Date().toISOString();
    }

    return value.toISOString();
  }

  const raw =
    String(value ?? "");

  if (!raw) {
    return new Date().toISOString();
  }

  const parsed =
    new Date(raw);

  if (
    Number.isNaN(
      parsed.getTime(),
    )
  ) {
    return new Date().toISOString();
  }

  return parsed.toISOString();
}

function getCardFields(
  cardVm: TokenBlueprintCardVm,
): TokenBlueprintCardFields {
  return (
    cardVm.fields ??
    cardVm
  );
}