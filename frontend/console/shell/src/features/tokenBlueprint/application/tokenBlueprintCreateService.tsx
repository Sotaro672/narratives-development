// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintCreateService.tsx

/**
 * TokenBlueprint作成カードのアプリケーションサービス。
 *
 * - Brand一覧取得
 * - TokenBlueprint作成
 * - iconFileがある場合は、作成後にFirebase Storageへfrontendから直接アップロードする
 * - Firebase StorageのdownloadURL / objectPath / fileName / contentType / sizeを
 *   TokenBlueprintのicon情報としてbackendへ保存する
 *
 * 方針:
 * - ブランド名は/brandsの一覧レスポンスitems[].nameを正とする
 * - brandIdからbrandNameの個別名前解決は行わない
 * - companyIdはTokenBlueprint作成APIへ送信せず、Firebase Storageの保存先決定にのみ使用する
 * - createdByはTokenBlueprint作成APIへ送信せず、backendの認証コンテキストを正とする
 * - tokenBlueprintIconはGCS signed URLを廃止し、Firebase Storageへ移行済み
 * - iconの永続化はiconIdやGCS objectではなく、
 *   Firebase StorageのdownloadURLとobjectPathを保存する
 */

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import type { CreateTokenBlueprintPayload } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import {
  createTokenBlueprint,
  updateTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { fetchBrandsForCurrentCompany } from "../../brand/infrastructure/http/brandRepositoryHTTP";
import { uploadTokenBlueprintIconToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

// ---------------------------------------------------------
// Brand一覧取得
// ---------------------------------------------------------

/**
 * /brandsの一覧レスポンスを正とする。
 *
 * 正レスポンス:
 * {
 *   items: [
 *     {
 *       id: string,
 *       name: string,
 *       brandIcon?: Firebase Storage downloadURL,
 *       brandBackgroundImage?: Firebase Storage downloadURL,
 *       memberName?: string
 *     }
 *   ]
 * }
 */
export async function loadBrandsForCompany(): Promise<
  {
    id: string;
    name: string;
  }[]
> {
  try {
    return await fetchBrandsForCurrentCompany();
  } catch {
    return [];
  }
}

// ---------------------------------------------------------
// TokenBlueprint作成
// ---------------------------------------------------------

/**
 * TokenBlueprint作成時のApplication入力。
 *
 * companyIdはbackendへのcreate payloadには含めず、
 * Firebase Storageの保存先決定にのみ利用する。
 */
export type CreateTokenBlueprintInput = CreateTokenBlueprintPayload & {
  companyId: string;
  iconFile?: File | null;
};

function normalizeIconUrlForSend(
  raw: unknown,
): string | undefined {
  const url =
    typeof raw === "string"
      ? raw.trim()
      : undefined;

  if (!url || url.startsWith("blob:")) {
    return undefined;
  }

  return url;
}

function normalizeOptionalString(
  raw: unknown,
): string | undefined {
  if (raw == null) {
    return undefined;
  }

  const value = String(raw).trim();
  return value || undefined;
}

function normalizeOptionalNumber(
  raw: unknown,
): number | undefined {
  if (raw == null) {
    return undefined;
  }

  const value = Number(raw);

  if (!Number.isFinite(value)) {
    return undefined;
  }

  return value >= 0 ? value : 0;
}

/**
 * TokenBlueprintを作成する。
 *
 * iconFileがない場合:
 * - 通常のcreateだけを行う
 * - iconUrl / iconObjectPathなどがinputにある場合はその値を送信する
 *
 * iconFileがある場合:
 * 1. icon情報を含めずTokenBlueprintを作成する
 * 2. 作成後のtokenBlueprintIdを使ってFirebase StorageへiconFileをアップロードする
 * 3. 取得したdownloadURL / objectPath / fileName / contentType / sizeを
 *    TokenBlueprintのicon情報として更新する
 *
 * companyId:
 * - create APIへは送信しない
 * - Firebase Storage uploadにのみ使用する
 *
 * createdBy:
 * - frontendから送信しない
 * - backend認証コンテキストを正とする
 */
export async function createTokenBlueprintWithOptionalIcon(
  input: CreateTokenBlueprintInput,
): Promise<TokenBlueprint> {
  const iconFile = input.iconFile ?? null;

  const payload: CreateTokenBlueprintPayload = {
    name: input.name,
    symbol: input.symbol,
    brandId: input.brandId,
    description: input.description,
    assigneeId: input.assigneeId,

    iconUrl: normalizeIconUrlForSend(input.iconUrl),
    iconObjectPath: normalizeOptionalString(input.iconObjectPath),
    iconFileName: normalizeOptionalString(input.iconFileName),
    iconContentType: normalizeOptionalString(input.iconContentType),
    iconSize: normalizeOptionalNumber(input.iconSize),

    contentFiles: input.contentFiles ?? [],
  };

  /*
   * iconFileがある場合、
   * blob URLや未確定のicon情報を作成時に保存しない。
   *
   * Firebase Storageへのアップロード完了後に、
   * 確定したdownloadURLとobjectPathで更新する。
   */
  if (iconFile) {
    delete payload.iconUrl;
    delete payload.iconObjectPath;
    delete payload.iconFileName;
    delete payload.iconContentType;
    delete payload.iconSize;
  }

  const created = await createTokenBlueprint(payload);

  if (!iconFile) {
    return created;
  }

  const tokenBlueprintId = created.id;

  if (!tokenBlueprintId) {
    throw new Error(
      "tokenBlueprint.id is required after create",
    );
  }

  const companyId = input.companyId;

  if (!companyId) {
    throw new Error(
      "companyId is required before uploading token blueprint icon",
    );
  }

  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({
    companyId,
    tokenBlueprintId,
    file: iconFile,
  });

  return updateTokenBlueprint(
    tokenBlueprintId,
    {
      iconUrl: uploaded.downloadUrl,
      iconObjectPath: uploaded.objectPath,
      iconFileName: uploaded.fileName,
      iconContentType: uploaded.contentType,
      iconSize: uploaded.size,
    },
  );
}