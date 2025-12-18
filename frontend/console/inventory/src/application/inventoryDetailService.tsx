// frontend/console/inventory/src/application/inventoryDetailService.tsx

import type { InventoryRow } from "../presentation/components/inventoryCard";
import {
  API_BASE,
  fetchInventoryIDsByProductAndTokenDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// Firebase Auth から ID トークンを取得（detail fetch に必要）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// ============================================================
// DTOs (local, to avoid missing exports)
// ============================================================

export type TokenBlueprintSummaryDTO = {
  id: string;
  name?: string;
  symbol?: string;
};

export type ProductBlueprintSummaryDTO = {
  id: string;
  name?: string;
};

export type ProductBlueprintPatchDTO = {
  productName?: string | null;
  brandId?: string | null;
  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: any;
  assigneeId?: string | null;
};

export type InventoryDetailRowDTO = {
  tokenBlueprintId?: string;
  token?: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | null;
  stock: number;
};

export type InventoryDetailDTO = {
  inventoryId: string;

  // ✅ NEW: backend DTO が返す場合がある（方針A）
  inventoryIds?: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;
  modelId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

  tokenBlueprint?: TokenBlueprintSummaryDTO;
  productBlueprint?: ProductBlueprintSummaryDTO;

  rows: InventoryDetailRowDTO[];
  totalStock: number;

  updatedAt?: string;
};

// ============================================================
// ViewModel (Screen-friendly shape)
// ============================================================

export type InventoryDetailViewModel = {
  /** 画面用の一意キー（pbId + tbId） */
  inventoryKey: string;

  /** 方針A: 詳細が対象とする inventoryId の集合 */
  inventoryIds: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;

  /** 方針Aでは原則空 */
  modelId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

  rows: InventoryRow[];
  totalStock: number;

  /** max(updatedAt) */
  updatedAt?: string;
};

// ============================================================
// helpers
// ============================================================

function asString(v: unknown): string {
  return String(v ?? "").trim();
}

function uniqStrings(xs: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const x of xs) {
    const s = asString(x);
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}

function toOptionalString(v: any): string | undefined {
  const s = asString(v);
  return s ? s : undefined;
}

function toRgbNumberOrNull(v: any): number | null | undefined {
  if (v === undefined) return undefined;
  if (v === null) return null;

  if (typeof v === "number" && Number.isFinite(v)) return v;

  const s = asString(v);
  if (!s) return null;

  const normalized = s.replace(/^#/, "").replace(/^0x/i, "");
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    const n = Number.parseInt(normalized, 16);
    return Number.isFinite(n) ? n : null;
  }

  const d = Number.parseInt(s, 10);
  return Number.isFinite(d) ? d : null;
}

async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("Not authenticated");
  const token = await user.getIdToken();
  if (!token) throw new Error("Failed to acquire ID token");
  return token;
}

function mapProductBlueprintPatch(raw: any): ProductBlueprintPatchDTO {
  const patchRaw = (raw ?? {}) as any;

  return {
    productName:
      patchRaw.productName !== undefined ? (patchRaw.productName as any) : undefined,
    brandId: patchRaw.brandId !== undefined ? (patchRaw.brandId as any) : undefined,
    itemType: patchRaw.itemType !== undefined ? String(patchRaw.itemType) : undefined,
    fit: patchRaw.fit !== undefined ? (patchRaw.fit as any) : undefined,
    material: patchRaw.material !== undefined ? (patchRaw.material as any) : undefined,
    weight:
      patchRaw.weight !== undefined && patchRaw.weight !== null
        ? Number(patchRaw.weight)
        : undefined,
    qualityAssurance: Array.isArray(patchRaw.qualityAssurance)
      ? patchRaw.qualityAssurance.map((x: any) => String(x))
      : undefined,
    productIdTag:
      patchRaw.productIdTag !== undefined ? patchRaw.productIdTag : undefined,
    assigneeId:
      patchRaw.assigneeId !== undefined ? (patchRaw.assigneeId as any) : undefined,
  };
}

function pickPatch(dtos: InventoryDetailDTO[]): ProductBlueprintPatchDTO {
  const found =
    dtos.find(
      (d) =>
        d?.productBlueprintPatch && Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};
  return found as any;
}

function pickTokenNameFromDTO(dto: any): string {
  return (
    asString(dto?.tokenBlueprint?.name) ||
    asString(dto?.TokenBlueprint?.name) ||
    asString(dto?.tokenBlueprintName) ||
    ""
  );
}

function pickUpdatedAtMax(dtos: InventoryDetailDTO[]): string | undefined {
  let maxUpdated: string | undefined = undefined;
  for (const d of dtos as any[]) {
    const t = d?.updatedAt ? String(d.updatedAt) : "";
    if (!t) continue;
    if (!maxUpdated || t > maxUpdated) maxUpdated = t;
  }
  return maxUpdated;
}

// ============================================================
// HTTP (detail)
// - /inventory/{inventoryId}
// ============================================================

async function fetchInventoryDetailDTO(inventoryId: string): Promise<InventoryDetailDTO> {
  const id = asString(inventoryId);
  if (!id) throw new Error("inventoryId is empty");

  const token = await getIdTokenOrThrow();
  const url = `${API_BASE}/inventory/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`Failed to fetch inventory detail: ${res.status} ${res.statusText} ${text}`);
  }

  const data = await res.json();

  const rows: InventoryDetailRowDTO[] = Array.isArray(data?.rows)
    ? data.rows.map((r: any) => ({
        tokenBlueprintId: toOptionalString(
          r?.tokenBlueprintId ?? r?.TokenBlueprintID ?? r?.token_blueprint_id,
        ),
        token: toOptionalString(r?.token ?? r?.Token),
        modelNumber: asString(r?.modelNumber ?? r?.ModelNumber),
        size: asString(r?.size ?? r?.Size),
        color: asString(r?.color ?? r?.Color),
        rgb: toRgbNumberOrNull(r?.rgb ?? r?.RGB),
        stock: Number(r?.stock ?? r?.Stock ?? 0),
      }))
    : [];

  return {
    inventoryId: asString(data?.inventoryId ?? data?.id ?? id),
    inventoryIds: Array.isArray(data?.inventoryIds)
      ? data.inventoryIds.map((x: any) => asString(x)).filter(Boolean)
      : undefined,

    tokenBlueprintId: asString(data?.tokenBlueprintId ?? data?.TokenBlueprintID),
    productBlueprintId: asString(data?.productBlueprintId ?? data?.ProductBlueprintID),
    modelId: asString(data?.modelId ?? data?.ModelID),

    productBlueprintPatch: mapProductBlueprintPatch(data?.productBlueprintPatch),

    tokenBlueprint: data?.tokenBlueprint
      ? {
          id: asString(data.tokenBlueprint.id),
          name: data.tokenBlueprint.name ? asString(data.tokenBlueprint.name) : undefined,
          symbol: data.tokenBlueprint.symbol
            ? asString(data.tokenBlueprint.symbol)
            : undefined,
        }
      : undefined,

    productBlueprint: data?.productBlueprint
      ? {
          id: asString(data.productBlueprint.id),
          name: data.productBlueprint.name
            ? asString(data.productBlueprint.name)
            : undefined,
        }
      : undefined,

    rows,
    totalStock: Number(data?.totalStock ?? 0),
    updatedAt: data?.updatedAt ? String(data.updatedAt) : undefined,
  };
}

// ============================================================
// Mapper (DTO row -> InventoryRow)
// ============================================================

function mapDtoToRows(
  dto: InventoryDetailDTO,
  opts?: { expectedTokenBlueprintId?: string; fallbackTokenName?: string },
): InventoryRow[] {
  const expectedTbId = asString(opts?.expectedTokenBlueprintId);
  const fallbackToken = asString(opts?.fallbackTokenName);

  const rowsRaw: any[] = Array.isArray((dto as any)?.rows) ? ((dto as any).rows as any[]) : [];
  const out: InventoryRow[] = [];

  for (const r of rowsRaw) {
    const rowTbId = asString(r?.tokenBlueprintId ?? r?.TokenBlueprintID ?? r?.token_blueprint_id);

    if (expectedTbId && rowTbId && rowTbId !== expectedTbId) continue;

    const token = asString(r?.token ?? r?.Token) || fallbackToken || "-";

    out.push({
      token,
      modelNumber: asString(r?.modelNumber ?? r?.ModelNumber ?? ""),
      size: asString(r?.size ?? r?.Size ?? ""),
      color: asString(r?.color ?? r?.Color ?? ""),
      rgb: (r?.rgb ?? r?.RGB ?? null) as any,
      stock: Number(r?.stock ?? r?.Stock ?? 0),
    });
  }

  return out;
}

// ============================================================
// Aggregator (multiple DTOs -> one ViewModel)
// ============================================================

function mergeDetailDTOs(
  pbId: string,
  tbId: string,
  inventoryIds: string[],
  dtos: InventoryDetailDTO[],
): InventoryDetailViewModel {
  const patch = pickPatch(dtos);
  const maxUpdated = pickUpdatedAtMax(dtos);

  // rows を結合 → 同一キーで合算（表示安定）
  const agg = new Map<
    string,
    { token?: string; modelNumber: string; size: string; color: string; rgb: any; stock: number }
  >();

  for (const d of dtos as any[]) {
    const fallbackTokenName = pickTokenNameFromDTO(d);

    for (const r of mapDtoToRows(d as any, { expectedTokenBlueprintId: tbId, fallbackTokenName })) {
      const token = asString(r.token) || "-";
      const modelNumber = asString(r.modelNumber) || "-";
      const size = asString(r.size) || "-";
      const color = asString(r.color) || "-";
      const rgbKey = r.rgb == null ? "nil" : String(r.rgb);

      const key = `${token}__${modelNumber}__${size}__${color}__${rgbKey}`;

      const cur = agg.get(key);
      if (!cur) {
        agg.set(key, {
          token,
          modelNumber,
          size,
          color,
          rgb: r.rgb ?? null,
          stock: Number(r.stock ?? 0),
        });
      } else {
        cur.stock += Number(r.stock ?? 0);
      }
    }
  }

  const mergedRows: InventoryRow[] = Array.from(agg.values()).map((v) => ({
    token: v.token,
    modelNumber: v.modelNumber,
    size: v.size,
    color: v.color,
    rgb: v.rgb,
    stock: v.stock,
  }));

  const totalStock = mergedRows.reduce((sum, r) => sum + Number(r.stock ?? 0), 0);

  mergedRows.sort((a, b) => {
    const as = (x: any) => String(x ?? "");
    if (as(a.token) !== as(b.token)) return as(a.token).localeCompare(as(b.token));
    if (as(a.modelNumber) !== as(b.modelNumber))
      return as(a.modelNumber).localeCompare(as(b.modelNumber));
    if (as(a.size) !== as(b.size)) return as(a.size).localeCompare(as(b.size));
    if (as(a.color) !== as(b.color)) return as(a.color).localeCompare(as(b.color));
    return as(a.rgb).localeCompare(as(b.rgb));
  });

  return {
    inventoryKey: `${pbId}__${tbId}`,
    inventoryIds,

    tokenBlueprintId: tbId,
    productBlueprintId: pbId,
    modelId: "",

    productBlueprintPatch: patch,
    rows: mergedRows,
    totalStock,

    updatedAt: maxUpdated,
  };
}

// ============================================================
// Query Request (Application Layer)
// - ✅ 方針Aのみ: pbId + tbId -> inventoryIds -> details -> merge
// ============================================================

export async function queryInventoryDetailByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryDetailViewModel> {
  const pbId = asString(productBlueprintId);
  const tbId = asString(tokenBlueprintId);

  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  // ① inventoryIds 解決
  const idsDto = await fetchInventoryIDsByProductAndTokenDTO(pbId, tbId);
  const idsFromResolver = Array.isArray((idsDto as any)?.inventoryIds)
    ? (idsDto as any).inventoryIds.map((x: unknown) => asString(x)).filter(Boolean)
    : [];

  if (idsFromResolver.length === 0) {
    throw new Error("inventoryIds is empty (no inventory for productBlueprintId + tokenBlueprintId)");
  }

  // ② 各 inventoryId の詳細を並列取得
  const results: PromiseSettledResult<InventoryDetailDTO>[] = await Promise.allSettled(
    idsFromResolver.map(async (id: string): Promise<InventoryDetailDTO> => {
      return await fetchInventoryDetailDTO(id);
    }),
  );

  const ok: InventoryDetailDTO[] = [];
  const failed: Array<{ id: string; reason: string }> = [];

  results.forEach((r, idx) => {
    const id = idsFromResolver[idx];
    if (r.status === "fulfilled") ok.push(r.value);
    else failed.push({ id, reason: String((r.reason as any)?.message ?? r.reason) });
  });

  if (ok.length === 0) {
    throw new Error(`failed to fetch any inventory detail: ${failed.map((x) => x.id).join(", ")}`);
  }

  // ✅ backend DTO に inventoryIds が入ってくる場合は union
  const idsFromDTO: string[] = [];
  for (const d of ok as any[]) {
    const xs = Array.isArray(d?.inventoryIds) ? d.inventoryIds : [];
    for (const x of xs) idsFromDTO.push(asString(x));
  }

  const inventoryIds = uniqStrings([...idsFromResolver, ...idsFromDTO]);

  // ③ マージしてViewModel化
  return mergeDetailDTOs(pbId, tbId, inventoryIds, ok);
}
