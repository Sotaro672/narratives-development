// frontend/console/mintRequest/src/application/mintRequestService.tsx

import type { InspectionBatchDTO } from "../infrastructure/api/mintRequestApi";

// repository は段階的移行に備えて * as import し、存在しない関数は runtime で noop にする
import * as repo from "../infrastructure/repository/mintRequestRepositoryHTTP";

/**
 * backend/internal/domain/productBlueprint.Patch に対応する DTO
 */
export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;

  // MintHandler (/mint/product_blueprints/{id}/patch) が付与するブランド名
  brandName?: string | null;

  itemType?: string | null; // Go 側 ItemType（"tops" / "bottoms" など）に対応
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: {
    type?: string | null;
  } | null;
  assigneeId?: string | null;
};

/**
 * backend/internal/domain/brand.Brand に対応する簡易 DTO
 * ListBrandByCompanyId（= /mint/brands）用
 */
export type BrandForMintDTO = {
  id: string;
  name: string;
};

/**
 * backend/internal/domain/tokenBlueprint.TokenBlueprint に対応する簡易 DTO
 * ListTokenBlueprintsByBrand（= /mint/token_blueprints?brandId=...）用
 *
 * - name: トークン名
 * - symbol: シンボル（例: LUMI）
 * - iconUrl: アイコン画像 URL（存在しない場合は undefined / 空文字）
 */
export type TokenBlueprintForMintDTO = {
  id: string;
  name: string;
  symbol: string;
  iconUrl?: string;
};

/**
 * mints テーブル（正）に対応する DTO
 * ※ 現在の運用では inspectionId = productionId を格納する前提
 */
export type MintDTO = {
  id: string;
  brandId: string;
  tokenBlueprintId: string;
  inspectionId: string; // = productionId
  products: string[]; // 正: array
  createdAt: string; // ISO string など（repo の実装に従う）
  createdBy: string;
  minted: boolean;
  mintedAt?: string | null;
  scheduledBurnDate?: string | null;

  // テーブル上は存在しても、ドメイン未定義ならここでは任意扱い
  onChainTxSignature?: string | null;
};

/**
 * MintUsecase 経由の /mint/inspections を叩き、
 * 指定された productionId に対応する InspectionBatchDTO を 1 件返す。
 *
 * ※ /mint/inspections は backend の MintUsecase
 *   （＝ GetModelVariationByID を含む処理）を経由する。
 */
export async function loadInspectionBatchFromMintAPI(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) return null;

  // /mint/inspections を実行（Repository 経由）
  const batches = await repo.fetchInspectionBatchesHTTP();

  // productionId で絞り込み
  const hit =
    batches.find((b) => (b as any).productionId === trimmed) ?? null;

  return hit;
}

/**
 * productBlueprintId から、MintUsecase.GetProductBlueprintPatchByID 経由で
 * ProductBlueprint Patch DTO を取得する。
 */
export async function loadProductBlueprintPatch(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const trimmed = productBlueprintId.trim();
  if (!trimmed) return null;

  return await repo.fetchProductBlueprintPatchHTTP(trimmed);
}

/**
 * current companyId に紐づく Brand 一覧を取得する。
 * backend/internal/application/usecase.MintUsecase.ListBrandsForCurrentCompany
 * （HTTP: GET /mint/brands）に対応。
 */
export async function loadBrandsForMint(): Promise<BrandForMintDTO[]> {
  const brands = await repo.fetchBrandsForMintHTTP();
  return brands ?? [];
}

/**
 * 指定した brandId に紐づく TokenBlueprint 一覧を取得する。
 * backend/internal/application/usecase.MintUsecase.ListTokenBlueprintsByBrand
 * （HTTP: GET /mint/token_blueprints?brandId=...）に対応。
 */
export async function loadTokenBlueprintsByBrand(
  brandId: string,
): Promise<TokenBlueprintForMintDTO[]> {
  const trimmed = brandId.trim();
  if (!trimmed) {
    return [];
  }

  const tokenBps = await repo.fetchTokenBlueprintsByBrandHTTP(trimmed);
  return tokenBps ?? [];
}

/**
 * ★ mints テーブルの有無で「申請済みモード」を判定するための取得関数（段階導入）
 *
 * - inspectionId (= productionId) で 1 件取得できることを期待
 * - repository 実装がまだ無い場合は null を返す（呼び出し側でフォールバック可能）
 *
 * 期待する repository 側の関数名（どちらか）:
 * - fetchMintByInspectionIdHTTP(inspectionId: string): Promise<MintDTO | null>
 */
export async function loadMintByInspectionIdFromMintAPI(
  inspectionId: string,
): Promise<MintDTO | null> {
  const trimmed = inspectionId.trim();
  if (!trimmed) return null;

  const anyRepo = repo as any;

  if (typeof anyRepo.fetchMintByInspectionIdHTTP === "function") {
    const mint = await anyRepo.fetchMintByInspectionIdHTTP(trimmed);
    return mint ?? null;
  }

  // repository 未実装なら noop（呼び出し側で旧 requestedBy/At 判定にフォールバック可能）
  return null;
}

/**
 * ★ 複数 inspectionId (= productionId) をまとめて引く版（段階導入）
 *
 * 期待する repository 側の関数名（どちらか）:
 * - fetchMintsByInspectionIdsHTTP(ids: string[]): Promise<Record<string, MintDTO>>
 *   ※ key は inspectionId
 */
export async function loadMintsByInspectionIdsFromMintAPI(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const anyRepo = repo as any;

  if (typeof anyRepo.fetchMintsByInspectionIdsHTTP === "function") {
    const m = await anyRepo.fetchMintsByInspectionIdsHTTP(ids);
    return (m ?? {}) as Record<string, MintDTO>;
  }

  return {};
}

/**
 * ミント申請詳細画面向けの TokenBlueprint を解決する。
 * 現状は個別 TokenBlueprint 詳細 API がないため undefined を返す。
 * 必要になれば backend の tokenBlueprint API を呼ぶ方式に置き換える。
 */
export function resolveBlueprintForMintRequest(requestId?: string) {
  return undefined;
}

/**
 * ★ ミント申請 POST 用サービス関数
 *
 * - productionId: 生産ID（inspectionBatch.productionId）
 * - tokenBlueprintId: 選択されたトークン設計ID
 * - scheduledBurnDate: 焼却予定日（"YYYY-MM-DD" 形式 / 任意）
 *
 * repository の postMintRequestHTTP に委譲する。
 */
export async function postMintRequest(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
) {
  const pid = productionId.trim();
  const tbid = tokenBlueprintId.trim();

  if (!pid || !tbid) {
    throw new Error("productionId / tokenBlueprintId is empty");
  }

  // eslint-disable-next-line no-console
  console.log("[MintRequestService] postMintRequest payload", {
    productionId: pid,
    tokenBlueprintId: tbid,
    scheduledBurnDate: scheduledBurnDate ?? null,
  });

  const res = await repo.postMintRequestHTTP(pid, tbid, scheduledBurnDate);

  // eslint-disable-next-line no-console
  console.log("[MintRequestService] postMintRequest response", res);

  return res;
}
