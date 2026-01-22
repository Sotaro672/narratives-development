// frontend/console/tokenBlueprint/src/application/tokenBlueprintCreateService.tsx

/**
 * TokenBlueprint 作成カードのアプリケーションサービス
 * - Brand 一覧取得
 * - brandId → brandName 解析
 * - TokenBlueprint 作成（必要なら create 時に iconUpload を発行）
 * - （任意）create レスポンスの iconUpload を使って「PUT → iconUrl 反映」まで一括実行
 *
 * NOTE:
 * - Repository 分割（案A）により、Brand API と signedUrl PUT は別モジュールへ移動済み。
 * - entity.go 正: icon の永続化は iconId(objectPath) ではなく iconUrl を保存（backend resolver に委譲）。
 */

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";

import { createTokenBlueprint, updateTokenBlueprint } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

// Brand API は repositoryHTTP から外す（分割案A）
import {
  fetchBrandsForCurrentCompany,
  fetchBrandNameById,
} from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";

// signed URL PUT も repositoryHTTP から外す（分割案A）
import { putFileToSignedUrl } from "../infrastructure/upload/signedUrlPut";

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

/**
 * TokenBlueprint を作成する。
 * - iconFile がある場合:
 *   create 時に backend に X-Icon-* を送って iconUpload を返してもらう
 *   → GCS PUT → publicUrl を backend に渡して iconUrl 保存（resolver が iconId 等へ加工する想定）
 * - iconFile がない場合:
 *   通常の create のみ
 *
 * 重要:
 * - backend 側で署名URL発行が無効（例: signer 未設定）だと iconUpload が返らないため、
 *   その場合は「画像アップロードはスキップ」して作成だけ成功させます。
 */
export async function createTokenBlueprintWithOptionalIcon(
  input: CreateTokenBlueprintInput,
): Promise<TokenBlueprint> {
  const iconFile = input.iconFile ?? null;

  // entity.go 正:
  // - Create payload に iconId は存在しない（iconUrl を渡す設計）
  const payload: CreateTokenBlueprintPayload = {
    name: input.name,
    symbol: input.symbol,
    brandId: input.brandId,
    companyId: input.companyId,
    description: input.description,
    assigneeId: input.assigneeId,
    createdBy: input.createdBy,
    iconUrl: input.iconUrl === undefined ? undefined : input.iconUrl,
    contentFiles: input.contentFiles ?? [],
  };

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

  // 1) まず create
  // - iconFile がある場合のみ「iconUpload を返してもらう」ためのヘッダ情報を渡す
  const tb = await createTokenBlueprint(
    payload,
    iconFile
      ? {
          iconFileName: iconFile.name,
          iconContentType: iconFile.type || "application/octet-stream",
        }
      : undefined,
  );

  console.log("[tokenBlueprintCreateService] create success", {
    id: (tb as any)?.id,
    iconUpload: (tb as any)?.iconUpload,
  });

  // 2) 画像が無いならここで終了
  if (!iconFile) return tb;

  // 3) create レスポンスに iconUpload が無い場合はスキップ
  const iconUpload = (tb as any)?.iconUpload as
    | {
        uploadUrl?: string;
        objectPath?: string;
        publicUrl?: string;
        expiresAt?: string;
        contentType?: string;
      }
    | undefined;

  const uploadUrl = String(iconUpload?.uploadUrl ?? "").trim();
  const publicUrl = String(iconUpload?.publicUrl ?? "").trim();
  const signedContentType = String(iconUpload?.contentType ?? "").trim();

  if (!uploadUrl || !publicUrl) {
    console.warn(
      "[tokenBlueprintCreateService] icon upload skipped: iconUpload is missing on create response. " +
        "Backend 側の署名URL発行が無効（signer未設定等）の可能性があります。",
      { id: (tb as any)?.id, iconUpload },
    );
    return tb;
  }

  // 4) ブラウザ → 署名付きURLへ PUT
  console.log("[tokenBlueprintCreateService] icon PUT start", {
    id: (tb as any)?.id,
    file: { name: iconFile.name, type: iconFile.type, size: iconFile.size },
    signedContentType,
  });

  await putFileToSignedUrl(uploadUrl, iconFile, signedContentType);

  console.log("[tokenBlueprintCreateService] icon PUT success", {
    id: (tb as any)?.id,
    publicUrl,
  });

  // 5) backend に publicUrl を渡して icon を紐付ける（entity.go 正: iconUrl を保存）
  const updated = await updateTokenBlueprint(String((tb as any)?.id ?? ""), {
    iconUrl: publicUrl,
  });

  console.log("[tokenBlueprintCreateService] icon attach success", {
    id: (updated as any)?.id,
    iconUrl: (updated as any)?.iconUrl,
    iconId: (updated as any)?.iconId,
  });

  return updated;
}
