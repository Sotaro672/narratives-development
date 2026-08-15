// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintCreateService.tsx

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import type { CreateTokenBlueprintPayload } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import {
  attachTokenBlueprintIcon,
  createTokenBlueprint,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { fetchBrandsForCurrentCompany } from "../../brand/infrastructure/http/brandRepositoryHTTP";
import { uploadTokenBlueprintIconToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

export type TokenBlueprintBrandOption = {
  id: string;
  name: string;
};

/**
 * TokenBlueprintCardで選択可能なブランドを取得する。
 *
 * /brandsのresponseを正とし、Frontend側でfallbackや個別名前解決は行わない。
 */
export async function loadBrandsForCompany(): Promise<TokenBlueprintBrandOption[]> {
  return fetchBrandsForCurrentCompany();
}

/**
 * TokenBlueprint作成時のApplication入力。
 *
 * Backendで確定するcompanyId / createdBy / updatedByや、
 * Firebase Storage upload後に確定するicon情報は含めない。
 * contentFilesも作成後に専用処理で追加する。
 */
export type CreateTokenBlueprintInput = {
  name: string;
  symbol: string;
  brandId: string;
  description: string;
  assigneeId: string;
  iconFile?: File | null;
};

/**
 * TokenBlueprintを作成する。
 *
 * iconFileなし:
 * 1. TokenBlueprint本体を作成
 * 2. Backend BFF responseをそのまま返す
 *
 * iconFileあり:
 * 1. icon情報なしでTokenBlueprint本体を作成
 * 2. BFF responseのid / companyIdを使ってFirebase Storageへupload
 * 3. upload結果の確定icon情報をBackendへ保存
 * 4. 更新後のBFF responseをそのまま返す
 */
export async function createTokenBlueprintWithOptionalIcon(
  input: CreateTokenBlueprintInput,
): Promise<TokenBlueprint> {
  const payload: CreateTokenBlueprintPayload = {
    name: input.name,
    symbol: input.symbol,
    brandId: input.brandId,
    description: input.description,
    assigneeId: input.assigneeId,
    contentFiles: [],
  };

  const created = await createTokenBlueprint(payload);
  const iconFile = input.iconFile ?? null;

  if (!iconFile) {
    return created;
  }

  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({
    companyId: created.companyId,
    tokenBlueprintId: created.id,
    file: iconFile,
  });

  return attachTokenBlueprintIcon({
    tokenBlueprintId: created.id,
    iconUrl: uploaded.downloadUrl,
    iconObjectPath: uploaded.objectPath,
    iconFileName: uploaded.fileName,
    iconContentType: uploaded.contentType,
    iconSize: uploaded.size,
  });
}