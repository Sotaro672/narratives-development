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
} from "../infrastructure/repository";

// ✅ MintRequestQueryService 呼び出し用（ID Token）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// ============================================================
// Types (ViewModel for MintRequestManagement)
// ============================================================

export type MintRequestRowStatus = "planning" | "requested" | "minted";

export type ViewRow = {
  id: string; // = productionId (= inspectionId 扱い)

  tokenName: string | null;
  productName: string | null;

  // ✅ 数量は Query 側が mintQuantity / productionQuantity を返すのでそれを優先
  mintQuantity: number;
  productionQuantity: number;

  status: MintRequestRowStatus; // = mint の有無・mintedAt で判定
  inspectionStatus: InspectionStatus; // = inspection.status

  createdByName: string | null;
  mintedAt: string | null;

  // ✅ detail 画面などで必要なら保持（現状 hook は参照しないので optional 扱い）
  tokenBlueprintId?: string | null;

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
// MintRequestQueryService (backend) access
// ============================================================

const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const API_BASE = sanitizeBase(sanitizeBase(RAW_ENV_BASE) || FALLBACK_BASE);

async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken(false);
}

async function requestJSON<T>(path: string): Promise<T> {
  const idToken = await getIdTokenOrThrow();
  const url = `${API_BASE}${path}`;
  log("request", { method: "GET", url });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
  });

  const txt = await res.text().catch(() => "");
  if (!res.ok) {
    log("response error", {
      url,
      status: res.status,
      statusText: res.statusText,
      bodyHead: txt?.slice(0, 300),
    });
    throw new Error(
      `MintRequestQueryService error: ${res.status} ${res.statusText}${
        txt ? `\n${txt}` : ""
      }`,
    );
  }

  try {
    return (txt ? JSON.parse(txt) : null) as T;
  } catch (_e) {
    log("response parse error", { url, bodyHead: txt?.slice(0, 300) });
    throw new Error("MintRequestQueryService response is not JSON");
  }
}

/**
 * MintRequestQueryService が返す “一覧用 DTO” を想定（フィールド名の揺れを吸収）
 *
 * ✅ 現在の backend (ProductionInspectionMintDTO) に合わせて:
 * - mintQuantity / productionQuantity を優先
 * - mintedAt / tokenBlueprintId は top-level or mint 配下の両方を吸収
 * - inspectionStatus / productName も top-level or inspection 配下を吸収
 */
type MintRequestManagementRowDTO = {
  id?: string;
  productionId?: string;
  inspectionId?: string;

  // list fields (preferred)
  tokenName?: string | null;
  productName?: string | null;
  mintQuantity?: number | null;
  productionQuantity?: number | null;
  inspectionStatus?: InspectionStatus | string | null;

  // legacy/alt fields (fallback)
  totalPassed?: number | null; // older name
  quantity?: number | null; // older name

  // requestedBy = mint.createdBy
  requestedBy?: string | null;
  requestedByName?: string | null;
  createdByName?: string | null; // 互換

  mintedAt?: string | null;

  // token blueprint (optional)
  tokenBlueprintId?: string | null;

  // raw sub docs (optional)
  mint?: {
    tokenBlueprintId?: string | null;
    tokenBlueprintID?: string | null; // casing fallback
    tokenBlueprint?: string | null; // older list DTO name
    tokenName?: string | null;
    mintedAt?: string | null;
    createdBy?: string | null;
  } | null;

  inspection?: {
    status?: InspectionStatus | string | null;
    productName?: string | null;
    totalPassed?: number | null;
    quantity?: number | null;
  } | null;
};

const asNonEmptyString = (v: any): string =>
  typeof v === "string" && v.trim() ? v.trim() : "";

const asStringOrNull = (v: any): string | null => {
  const s = typeof v === "string" ? v.trim() : "";
  return s ? s : null;
};

const asNumber0 = (v: any): number => {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
};

function normalizeInspectionStatus(v: any): InspectionStatus {
  const s = String(v ?? "").trim();
  if (s === "completed") return "completed";
  if (s === "inspecting") return "inspecting";
  return "notYet" as any; // domain 側の実体に合わせている前提
}

function normalizeProductionId(b: any): string {
  return String(b?.productionId ?? b?.inspectionId ?? b?.id ?? "").trim();
}

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

function uniqStrings(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const x of xs ?? []) {
    const s = String(x ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
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

/**
 * ✅ Query DTO (ProductionInspectionMintDTO など) から ViewRow に変換
 *
 * 重要:
 * - backend は mintQuantity / productionQuantity を返しているので、それを最優先で使う
 * - 以前の totalPassed / quantity を参照すると 0 に落ちる（今回の原因）
 */
function buildRowsFromQueryDTO(items: MintRequestManagementRowDTO[]): ViewRow[] {
  return (items ?? []).map((raw) => {
    const pid =
      asNonEmptyString(raw.productionId) ||
      asNonEmptyString(raw.inspectionId) ||
      asNonEmptyString(raw.id);

    const inspSt = normalizeInspectionStatus(
      raw.inspectionStatus ?? raw.inspection?.status,
    );

    const mintedAt =
      asStringOrNull(raw.mintedAt) ?? asStringOrNull(raw.mint?.mintedAt);

    // ✅ tokenName / productName は top-level 優先、なければ subdoc から拾う
    const tokenName =
      asStringOrNull(raw.tokenName) ?? asStringOrNull(raw.mint?.tokenName);

    const productName =
      asStringOrNull(raw.productName) ??
      asStringOrNull(raw.inspection?.productName);

    // ✅ 数量は mintQuantity / productionQuantity を最優先（←ここが今回の修正点）
    const mintQuantity = asNumber0(
      raw.mintQuantity ?? raw.totalPassed ?? raw.inspection?.totalPassed ?? 0,
    );

    const productionQuantity = asNumber0(
      raw.productionQuantity ?? raw.quantity ?? raw.inspection?.quantity ?? 0,
    );

    // ✅ tokenBlueprintId も維持（detail 画面や更新API用）
    const tokenBlueprintId =
      asStringOrNull(raw.tokenBlueprintId) ??
      asStringOrNull(raw.mint?.tokenBlueprintId) ??
      asStringOrNull(raw.mint?.tokenBlueprintID) ??
      asStringOrNull(raw.mint?.tokenBlueprint);

    // ✅ requestedBy / createdByName の吸収
    const requestedBy =
      asStringOrNull(raw.requestedBy) ?? asStringOrNull(raw.mint?.createdBy);

    const requesterName =
      asStringOrNull(raw.requestedByName) ??
      asStringOrNull(raw.createdByName) ??
      requestedBy;

    // ✅ planning/requested/minted 判定
    const hasRequestSignal =
      !!tokenBlueprintId || !!tokenName || !!requestedBy || !!mintedAt;

    const st: MintRequestRowStatus = !hasRequestSignal
      ? "planning"
      : mintedAt
        ? "minted"
        : "requested";

    return {
      id: pid,

      tokenName: tokenName ?? null,
      productName: productName ?? null,

      mintQuantity,
      productionQuantity,

      status: st,
      inspectionStatus: inspSt,

      createdByName: requesterName ?? null,
      mintedAt: mintedAt ?? null,

      tokenBlueprintId: tokenBlueprintId ?? null,

      statusLabel: inspectionStatusLabel(inspSt),
    };
  });
}

// ============================================================
// Service: MintRequestManagement (list screen)
// ============================================================

/**
 * MintRequestManagement 一覧用の行を組み立てて返す。
 *
 * ✅ 新フロー（推奨）:
 * - MintRequestQueryService（backend query）に 1 回リクエストして一覧を取得
 *
 * ✅ フォールバック:
 * - 旧フロー（inspections → productionIds → mints）で合成
 */
export async function loadMintRequestManagementRows(): Promise<ViewRow[]> {
  log("load start");

  // ------------------------------------------------------------
  // 0) ✅ まずは MintRequestQueryService を叩く（本命）
  // ------------------------------------------------------------
  const queryPaths = [
    "/mint/requests",
    "/mint/requests?view=management",
    "/mint/requests?view=list",
  ];

  for (const path of queryPaths) {
    try {
      const dto = await requestJSON<any>(path);

      // array 想定
      if (Array.isArray(dto)) {
        const rows = buildRowsFromQueryDTO(dto as MintRequestManagementRowDTO[]);
        log("MintRequestQueryService success", {
          path,
          rowsLen: rows.length,
          sample0: rows[0],
        });
        log("load end");
        return rows;
      }

      // object { items: [] } みたいな形にも対応
      if (dto && Array.isArray((dto as any).items)) {
        const rows = buildRowsFromQueryDTO(
          (dto as any).items as MintRequestManagementRowDTO[],
        );
        log("MintRequestQueryService success(items)", {
          path,
          rowsLen: rows.length,
          sample0: rows[0],
        });
        log("load end");
        return rows;
      }

      log("MintRequestQueryService unexpected shape", { path, dtoHead: dto });
    } catch (e: any) {
      log("MintRequestQueryService failed -> fallback next path", {
        path,
        err: e?.message ?? e,
      });
    }
  }

  // ------------------------------------------------------------
  // 1) フォールバック（旧フロー）
  // ------------------------------------------------------------
  log("fallback: legacy flow start");

  // まず inspections を取得（productionIds 抽出用）
  const initialBatches: InspectionBatchDTO[] = await fetchInspectionBatches();
  log(
    "fetchInspectionBatches(initial) result length=",
    (initialBatches ?? []).length,
    "sample[0]=",
    (initialBatches ?? [])[0],
  );

  const productionIds = uniqStrings(
    (initialBatches ?? []).map((b) => normalizeProductionId(b)),
  );

  log(
    "productionIds(uniq) length=",
    productionIds.length,
    "sample[0..4]=",
    productionIds.slice(0, 5),
  );

  // inspections を ListByProductionID で取得（可能なら）
  let batches: InspectionBatchDTO[] = initialBatches ?? [];
  try {
    const byProd: InspectionBatchDTO[] = await (fetchInspectionBatches as any)(
      productionIds,
    );
    if (Array.isArray(byProd) && byProd.length >= 0) {
      batches = byProd;
    }
    log(
      "fetchInspectionBatches(ListByProductionID) length=",
      (batches ?? []).length,
      "sample[0]=",
      (batches ?? [])[0],
    );
  } catch (e: any) {
    log(
      "fetchInspectionBatches(ListByProductionID) error=",
      e?.message ?? e,
      "fallback to initialBatches",
    );
    batches = initialBatches ?? [];
  }

  const productionIds2 = uniqStrings(
    (batches ?? []).map((b) => normalizeProductionId(b)),
  );
  log(
    "productionIds(from ListByProductionID inspections) length=",
    productionIds2.length,
    "sample[0..4]=",
    productionIds2.slice(0, 5),
  );

  // mints を map 取得
  let mintMap: Record<string, MintListRowDTO> = {};
  try {
    mintMap = await (listMintsByInspectionIDsHTTP as any)(productionIds2);
    const keys = Object.keys(mintMap ?? {});
    log(
      "listMintsByProductionID (via listMintsByInspectionIDsHTTP) keys=",
      keys.length,
      "sampleKey=",
      keys[0],
      "sampleVal=",
      keys[0] ? (mintMap as any)[keys[0]] : undefined,
    );
  } catch (e: any) {
    log("listMintsByProductionID error=", e?.message ?? e);
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

  log("fallback: legacy flow end");
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
  const ids = uniqStrings(inspectionIds ?? []);
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
