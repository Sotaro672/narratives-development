// frontend/console/tokenBlueprint/src/application/tokenBlueprintCreateService.tsx

/**
 * TokenBlueprint 作成カードのアプリケーションサービス
 * - Brand 一覧取得
 * - brandId → brandName 解析
 * - TokenBlueprint 作成
 * - iconFile がある場合は create 後に Firebase Storage へ frontend から直接アップロード
 * - Firebase Storage の downloadURL を iconUrl として TokenBlueprint に保存
 *
 * NOTE:
 * - tokenBlueprintIcon は GCS signed URL を廃止し、Firebase Storage へ移行。
 * - entity.go 正: icon の永続化は iconId(objectPath) ではなく iconUrl を保存。
 */

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";

import {
  createTokenBlueprint,
  updateTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import {
  fetchBrandsForCurrentCompany,
  fetchBrandNameById,
} from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";

import { uploadTokenBlueprintIconToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

import type { CreateTokenBlueprintPayload } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

// ---------------------------
// Brand 一覧取得
// ---------------------------
export async function loadBrandsForCompany(): Promise<{ id: string; name: string }[]> {
  try {
    const brands = await fetchBrandsForCurrentCompany();
    return brands;
  } catch (e) {
    console.error("[tokenBlueprintCreateService] loadBrandsForCompany error:", e);
    return [];
  }
}

// ---------------------------
// brandId → brandName 解決
// ---------------------------
export async function resolveBrandName(brandId: string): Promise<string> {
  try {
    const name = await fetchBrandNameById(brandId);
    return name ?? "";
  } catch (e) {
    console.error("[tokenBlueprintCreateService] resolveBrandName error:", e);
    return "";
  }
}

// ---------------------------
// TokenBlueprint 作成（画像なし/あり両対応）
// ---------------------------

export type CreateTokenBlueprintInput = CreateTokenBlueprintPayload & {
  // UI 側で File を持っている場合だけ渡す（未選択なら undefined）
  iconFile?: File | null;
};

function normalizeIconUrlForSend(raw: unknown): string | undefined {
  const u = typeof raw === "string" ? raw.trim() : undefined;
  if (!u) return undefined;
  if (u.startsWith("blob:")) return undefined;
  return u;
}

/**
 * TokenBlueprint を作成する。
 *
 * - iconFile がない場合:
 *   通常の create のみ
 *
 * - iconFile がある場合:
 *   1. TokenBlueprint を create
 *   2. 作成後の tokenBlueprintId を使って Firebase Storage へ iconFile をアップロード
 *   3. getDownloadURL で取得した URL を iconUrl として update
 */
export async function createTokenBlueprintWithOptionalIcon(
  input: CreateTokenBlueprintInput,
): Promise<TokenBlueprint> {
  const iconFile = input.iconFile ?? null;

  const payload: CreateTokenBlueprintPayload = {
    name: input.name,
    symbol: input.symbol,
    brandId: input.brandId,
    companyId: input.companyId,
    description: input.description,
    assigneeId: input.assigneeId,
    createdBy: input.createdBy,
    iconUrl: normalizeIconUrlForSend(input.iconUrl),
    contentFiles: input.contentFiles ?? [],
  };

  // iconFile がある場合、blob URL 等を create payload で保存しない。
  // Firebase Storage upload 後に downloadURL で確定させる。
  if (iconFile) {
    delete (payload as any).iconUrl;
  }

  console.log("[tokenBlueprintCreateService] create start", {
    name: payload.name,
    symbol: payload.symbol,
    brandId: payload.brandId,
    companyId: payload.companyId,
    hasIconFile: Boolean(iconFile),
    iconFile: iconFile
      ? { name: iconFile.name, type: iconFile.type, size: iconFile.size }
      : null,
  });

  const created = await createTokenBlueprint(payload);

  console.log("[tokenBlueprintCreateService] create success", {
    id: (created as any)?.id,
  });

  if (!iconFile) {
    return created;
  }

  const tokenBlueprintId = String((created as any)?.id ?? "").trim();
  if (!tokenBlueprintId) {
    throw new Error("tokenBlueprint.id is empty after create.");
  }

  const companyId = String(input.companyId ?? "").trim();
  if (!companyId) {
    throw new Error("companyId is required before uploading token blueprint icon.");
  }

  console.log("[tokenBlueprintCreateService] Firebase Storage icon upload start", {
    tokenBlueprintId,
    companyId,
    file: {
      name: iconFile.name,
      type: iconFile.type,
      size: iconFile.size,
    },
  });

  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({
    companyId,
    tokenBlueprintId,
    file: iconFile,
  });

  console.log("[tokenBlueprintCreateService] Firebase Storage icon upload success", {
    tokenBlueprintId,
    objectPath: uploaded.objectPath,
    downloadUrl: uploaded.downloadUrl,
  });

  const updated = await updateTokenBlueprint(tokenBlueprintId, {
    iconUrl: uploaded.downloadUrl,
  });

  console.log("[tokenBlueprintCreateService] icon attach success", {
    id: (updated as any)?.id,
    iconUrl: (updated as any)?.iconUrl,
    iconId: (updated as any)?.iconId,
    objectPath: uploaded.objectPath,
  });

  return updated;
}