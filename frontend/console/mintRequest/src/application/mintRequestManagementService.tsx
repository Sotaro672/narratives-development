// frontend/console/mintRequest/src/application/mintRequestManagementService.tsx
import type { InspectionStatus } from "../domain/entity/inspections";
import {
  fetchInspectionBatches,
  type InspectionBatchDTO,
  type MintListRowDTO,
  type MintDTO,
} from "../infrastructure/api/mintRequestApi";

// ✅ 一覧用 MintListRow を repository から取得する（DTO とは別ルート）
import {
  listMintsByInspectionIDsHTTP,
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
// Debug logger
// ============================================================

const log = (...args: any[]) => {
  // eslint-disable-next-line no-console
  console.log("[mintRequest/mintRequestManagementService]", ...args);
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
  if ((mint as any)?.mintedAt) return "minted";
  return "requested";
}

function normalizeProductionId(b: any): string {
  return String(b?.productionId ?? "").trim();
}

function buildRows(
  batches: InspectionBatchDTO[],
  mintMap: Record<string, MintListRowDTO>,
): ViewRow[] {
  return (batches ?? []).map((b) => {
    const pid = normalizeProductionId(b);
    const mint: MintListRowDTO | null = pid ? (mintMap?.[pid] ?? null) : null;

    const st = deriveMintStatusFromListRow(mint);
    const inspSt = (b.status ?? "inspecting") as InspectionStatus;

    return {
      id: pid,

      tokenName: (mint as any)?.tokenName ?? null,
      productName: (b as any).productName ?? null,

      mintQuantity: (b as any).totalPassed ?? 0,
      productionQuantity:
        (b as any).quantity ?? ((b as any).inspections?.length ?? 0),

      status: st,
      inspectionStatus: inspSt,

      createdByName: (mint as any)?.createdByName ?? null,
      mintedAt: ((mint as any)?.mintedAt ?? null) as string | null,

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
  log("load start");

  const batches: InspectionBatchDTO[] = await fetchInspectionBatches();
  log(
    "fetchInspectionBatches result length=",
    (batches ?? []).length,
    "sample[0]=",
    (batches ?? [])[0],
  );

  const productionIds = (batches ?? [])
    .map((b) => normalizeProductionId(b))
    .filter((s) => !!s);

  log(
    "productionIds length=",
    productionIds.length,
    "sample[0..4]=",
    productionIds.slice(0, 5),
  );

  // ✅ 一覧は list row を取得（view=list 相当）
  let mintMap: Record<string, MintListRowDTO> = {};
  try {
    mintMap = await listMintsByInspectionIDsHTTP(productionIds);
    const keys = Object.keys(mintMap ?? {});
    log(
      "listMintsByInspectionIDsHTTP keys=",
      keys.length,
      "sampleKey=",
      keys[0],
      "sampleVal=",
      keys[0] ? (mintMap as any)[keys[0]] : undefined,
    );
  } catch (e: any) {
    log("listMintsByInspectionIDsHTTP error=", e?.message ?? e);
    mintMap = {};
  }

  const rows = buildRows(batches ?? [], mintMap ?? {});
  log(
    "buildRows rows(length)=",
    rows.length,
    "rowsWithTokenName=",
    rows.filter((r) => !!r.tokenName).length,
    "rows sample[0..4]=",
    rows.slice(0, 5),
  );

  log(
    "rows with empty tokenName:",
    rows.filter((r) => !r.tokenName).slice(0, 10),
  );

  log("load end");
  return rows;
}

// ============================================================
// Service: MintDTO fetch (full DTO)
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

  const keys = Object.keys(m ?? {});
  log(
    "loadMintsDTOMapByInspectionIds keys=",
    keys.length,
    "sampleKey=",
    keys[0],
    "sampleVal=",
    keys[0] ? (m as any)[keys[0]] : undefined,
  );

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

  log("loadMintDTOByInspectionId iid=", iid, "result=", m ?? null);

  return (m ?? null) as MintDTO | null;
}
