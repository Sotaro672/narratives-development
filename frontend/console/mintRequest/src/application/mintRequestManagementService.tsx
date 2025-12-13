// frontend/console/mintRequest/src/application/mintRequestManagementService.tsx
import type { InspectionStatus } from "../domain/entity/inspections";
import {
  fetchInspectionBatches,
  fetchMintsMapByInspectionIds,
  type InspectionBatchDTO,
  type MintListRowDTO,
  type MintDTO,
} from "../infrastructure/api/mintRequestApi";

// MintDTO を repository から直接引くため（MintListRowDTO とは別ルート）
import {
  fetchMintsByInspectionIdsHTTP,
  fetchMintByInspectionIdHTTP,
} from "../infrastructure/repository/mintRequestRepositoryHTTP";

// ============================================================
// Types (ViewModel for MintRequestManagement)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type ViewRow = {
  id: string; // = productionId (= inspectionId 扱い)

  tokenName: string | null;
  productName: string | null;

  mintQuantity: number; // = inspection.totalPassed
  productionQuantity: number; // = inspection.quantity

  status: MintRequestRowStatus; // = mint の有無・mintedAt で判定
  inspectionStatus: InspectionStatus; // = inspection.status

  createdByName: string | null;
  mintedAt: string | null;

  statusLabel: string; // 画面表示用（ここでは検査ステータス）
};

// ============================================================
// Pure helpers
// ============================================================

const inspectionStatusLabel = (
  s: InspectionStatus | null | undefined,
): string => {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
};

function deriveMintStatusFromListRow(
  mint: MintListRowDTO | null,
): MintRequestRowStatus {
  if (!mint) return "planning";
  if (mint.mintedAt) return "minted";
  return "requested";
}

function normalizeProductionId(b: any): string {
  return String(b?.productionId ?? "").trim();
}

function buildRows(
  batches: InspectionBatchDTO[],
  mintMap: Record<string, MintListRowDTO>,
): ViewRow[] {
  return batches.map((b) => {
    const pid = normalizeProductionId(b);
    const mint: MintListRowDTO | null = pid ? (mintMap[pid] ?? null) : null;

    const st = deriveMintStatusFromListRow(mint);
    const inspSt = (b.status ?? "inspecting") as InspectionStatus;

    return {
      id: pid,

      tokenName: mint?.tokenName ?? null,
      productName: (b as any).productName ?? null,

      mintQuantity: (b as any).totalPassed ?? 0,
      productionQuantity:
        (b as any).quantity ?? ((b as any).inspections?.length ?? 0),

      status: st,
      inspectionStatus: inspSt,

      createdByName: (mint?.createdByName as any) ?? null,
      mintedAt: (mint?.mintedAt as any) ?? null,

      statusLabel: inspectionStatusLabel(inspSt),
    };
  });
}

// ============================================================
// Service: MintRequestManagement (list screen)
// ============================================================

/**
 * MintRequestManagement 一覧用の行を組み立てて返す。
 * - inspections を取得
 * - productionIds を抽出
 * - mints(list) をまとめて取得（inspectionId -> MintListRowDTO）
 * - 1行に合成して ViewRow[] で返す
 */
export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  const batches: InspectionBatchDTO[] = await fetchInspectionBatches();

  const productionIds = (batches ?? [])
    .map((b) => normalizeProductionId(b))
    .filter((s) => !!s);

  const mintMap = await fetchMintsMapByInspectionIds(productionIds);

  return buildRows(batches ?? [], mintMap ?? {});
}

// ============================================================
// Service: MintDTO fetch (full DTO)  ※要望により追加
// ============================================================

/**
 * MintDTO を inspectionIds (= productionIds) でまとめて取得する。
 * - 詳細画面や、将来的な “mint存在判定以外の情報” が必要になった場合のため
 */
export async function loadMintsDTOMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  // repository 直呼び（/mint/mints?inspectionIds=... のレスポンスを MintDTO map として扱う）
  const m = await fetchMintsByInspectionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}

/**
 * MintDTO を inspectionId で1件取得（バックエンドが用意されている場合）
 */
export async function loadMintDTOByInspectionId(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) return null;

  const m = await fetchMintByInspectionIdHTTP(iid);
  return (m ?? null) as MintDTO | null;
}
