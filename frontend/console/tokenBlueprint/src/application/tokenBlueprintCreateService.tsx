// frontend/console/tokenBlueprint/src/application/tokenBlueprintCreateService.tsx

/**
 * TokenBlueprint 作成カードのアプリケーションサービス
 * - Brand 一覧取得
 * - brandId → brandName 解析
 * - TokenBlueprint 作成（必要なら create 時に iconUpload を発行）
 * - （任意）create レスポンスの iconUpload を使って「PUT → iconId 反映」まで一括実行
 */

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";

import {
  fetchBrandsForCurrentCompany,
  fetchBrandNameById,
  createTokenBlueprint,
  updateTokenBlueprint,
  putFileToSignedUrl,
  type CreateTokenBlueprintPayload,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

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
 *   create 時に backend に X-Icon-* を送って iconUpload を返してもらう → GCS PUT → iconId 更新
 * - iconFile がない場合:
 *   通常の create のみ
 *
 * 重要:
 * - backend ログに「missing TOKEN_ICON_SIGNER_EMAIL env」が出ている場合、
 *   create レスポンスに iconUpload が乗らないので「画像アップロードはスキップ」して作成だけ成功させます。
 */
export async function createTokenBlueprintWithOptionalIcon(
  input: CreateTokenBlueprintInput,
): Promise<TokenBlueprint> {
  const iconFile = input.iconFile ?? null;

  // iconId は create 時点では通常不要（後から objectPath をセットする）
  const payload: CreateTokenBlueprintPayload = {
    name: input.name,
    symbol: input.symbol,
    brandId: input.brandId,
    companyId: input.companyId,
    description: input.description,
    assigneeId: input.assigneeId,
    createdBy: input.createdBy,
    iconId: null,
    contentFiles: input.contentFiles ?? [],
  };

  console.log("[tokenBlueprintCreateService] create start", {
    name: payload.name,
    symbol: payload.symbol,
    brandId: payload.brandId,
    hasIconFile: Boolean(iconFile),
    iconFile: iconFile
      ? { name: iconFile.name, type: iconFile.type, size: iconFile.size }
      : null,
  });

  // 1) まず create
  // - iconFile がある場合のみ「iconUpload を返してもらう」ためのヘッダ情報を渡す
  //   ※ repository 層で X-Icon-File-Name / X-Icon-Content-Type を付与する想定
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
  const objectPath = String(iconUpload?.objectPath ?? "").trim();
  const signedContentType = String(iconUpload?.contentType ?? "").trim();

  if (!uploadUrl || !objectPath) {
    console.warn(
      "[tokenBlueprintCreateService] icon upload skipped: iconUpload is missing on create response. " +
        "Backend env TOKEN_ICON_SIGNER_EMAIL 未設定などが原因の可能性があります。",
      { id: (tb as any)?.id, iconUpload },
    );
    return tb;
  }

  // 4) ブラウザ → 署名付きURLへ PUT
  // - Content-Type は署名に含まれるので一致必須
  console.log("[tokenBlueprintCreateService] icon PUT start", {
    id: (tb as any)?.id,
    objectPath,
    file: { name: iconFile.name, type: iconFile.type, size: iconFile.size },
    signedContentType,
  });

  await putFileToSignedUrl(uploadUrl, iconFile, signedContentType);

  console.log("[tokenBlueprintCreateService] icon PUT success", {
    id: (tb as any)?.id,
    objectPath,
  });

  // 5) token_blueprints.iconId に objectPath を反映（= アイコン紐付け確定）
  const updated = await updateTokenBlueprint(String((tb as any)?.id ?? ""), {
    iconId: objectPath,
  });

  console.log("[tokenBlueprintCreateService] icon attach success", {
    id: (updated as any)?.id,
    iconId: (updated as any)?.iconId,
    iconUrl: (updated as any)?.iconUrl,
  });

  return updated;
}
