// frontend/console/inventory/src/application/inventoryDetailService.tsx

import type { InventoryRow } from "../presentation/components/inventoryCard";
import {
  fetchInventoryDetailDTO,
  fetchInventoryIDsByProductAndTokenDTO,
  type InventoryDetailDTO,
  type ProductBlueprintPatchDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

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

  /** 単一Mint前提だった名残り。方針Aでは原則空でOK（必要なら modelIds[] にする） */
  modelId: string;

  // ProductBlueprintCard に流し込むため（GetPatchByID の結果）
  productBlueprintPatch: ProductBlueprintPatchDTO;

  // InventoryCard rows（複数 inventory の集計結果）
  rows: InventoryRow[];
  totalStock: number;

  /** max(updatedAt) */
  updatedAt?: string;
};

// ============================================================
// Mapper (DTO row -> InventoryRow)
// ============================================================

function mapDtoToRows(dto: InventoryDetailDTO): InventoryRow[] {
  const rows = Array.isArray(dto.rows) ? dto.rows : [];

  return rows.map((r) => ({
    token: r.token ?? undefined,
    modelNumber: String(r.modelNumber ?? ""),
    size: String(r.size ?? ""),
    color: String(r.color ?? ""),
    rgb: (r.rgb ?? null) as any,
    stock: Number(r.stock ?? 0),
  }));
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
  // patch は最初に取れたものを優先
  const patch =
    dtos.find(
      (d) =>
        d?.productBlueprintPatch &&
        Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};

  // updatedAt は max
  let maxUpdated: string | undefined = undefined;
  for (const d of dtos) {
    const t = d?.updatedAt ? String(d.updatedAt) : "";
    if (!t) continue;
    if (!maxUpdated || t > maxUpdated) maxUpdated = t;
  }

  // rows を結合 → 同一キーで合算
  // キー: token + modelNumber + size + color + rgb
  const agg = new Map<
    string,
    {
      token?: string;
      modelNumber: string;
      size: string;
      color: string;
      rgb: any;
      stock: number;
    }
  >();

  for (const d of dtos) {
    for (const r of mapDtoToRows(d)) {
      const token = String(r.token ?? "").trim() || "-";
      const modelNumber = String(r.modelNumber ?? "").trim() || "-";
      const size = String(r.size ?? "").trim() || "-";
      const color = String(r.color ?? "").trim() || "-";
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

  // totalStock は rows 合計（DTOの totalStock を足すより確実）
  const totalStock = mergedRows.reduce((sum, r) => sum + Number(r.stock ?? 0), 0);

  // 表示安定のため sort
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
// - 方針A: pbId + tbId -> inventoryIds -> details -> merge
// ============================================================

export async function queryInventoryDetailByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryDetailViewModel> {
  const pbId = String(productBlueprintId ?? "").trim();
  const tbId = String(tokenBlueprintId ?? "").trim();

  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  console.log("[inventory/queryInventoryDetailByProductAndToken] start", { pbId, tbId });

  // ① inventoryIds 解決
  const idsDto = await fetchInventoryIDsByProductAndTokenDTO(pbId, tbId);
  const inventoryIds = Array.isArray(idsDto?.inventoryIds)
    ? idsDto.inventoryIds.map((x: unknown) => String(x ?? "").trim()).filter(Boolean)
    : [];

  console.log("[inventory/queryInventoryDetailByProductAndToken] ids resolved", {
    pbId,
    tbId,
    count: inventoryIds.length,
    sample: inventoryIds.slice(0, 10),
    raw: idsDto,
  });

  if (inventoryIds.length === 0) {
    // 404にしたいなら repository 側で 404 を投げてもOK。ここではUI表示を考えて空で返すのも手。
    throw new Error(
      "inventoryIds is empty (no inventory for productBlueprintId + tokenBlueprintId)",
    );
  }

  // ② 各 inventoryId の詳細を並列取得
  const results: PromiseSettledResult<InventoryDetailDTO>[] =
    await Promise.allSettled(
      inventoryIds.map(async (id: string): Promise<InventoryDetailDTO> => {
        const dto = await fetchInventoryDetailDTO(id);
        return dto;
      }),
    );

  const ok: InventoryDetailDTO[] = [];
  const failed: Array<{ id: string; reason: string }> = [];

  results.forEach((r: PromiseSettledResult<InventoryDetailDTO>, idx: number) => {
    const id = inventoryIds[idx];
    if (r.status === "fulfilled") ok.push(r.value);
    else failed.push({ id, reason: String((r.reason as any)?.message ?? r.reason) });
  });

  console.log("[inventory/queryInventoryDetailByProductAndToken] detail fetched", {
    pbId,
    tbId,
    ok: ok.length,
    failed: failed.length,
    failedSample: failed.slice(0, 5),
  });

  if (ok.length === 0) {
    throw new Error(
      `failed to fetch any inventory detail: ${failed.map((x) => x.id).join(", ")}`,
    );
  }

  // ③ マージしてViewModel化
  const vm = mergeDetailDTOs(pbId, tbId, inventoryIds, ok);

  console.log("[inventory/queryInventoryDetailByProductAndToken] merged viewModel", {
    inventoryKey: vm.inventoryKey,
    inventoryIds: vm.inventoryIds.length,
    totalStock: vm.totalStock,
    rowsCount: vm.rows.length,
    rowsSample: vm.rows.slice(0, 5),
    productBlueprintPatch: vm.productBlueprintPatch,
    updatedAt: vm.updatedAt,
  });

  return vm;
}

/**
 * 互換: 旧ルート（/inventory/detail/:inventoryId）を残す場合のみ使う
 * ※ 方針Aへ移行後は、基本は queryInventoryDetailByProductAndToken を呼ぶ
 */
export async function queryInventoryDetail(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  console.log("[inventory/queryInventoryDetail] start", { inventoryId: id });

  const dto = await fetchInventoryDetailDTO(id);

  console.log("[inventory/queryInventoryDetail] dto received", {
    inventoryId: id,
    tokenBlueprintId: dto.tokenBlueprintId,
    productBlueprintId: dto.productBlueprintId,
    modelId: dto.modelId,
    totalStock: dto.totalStock,
    rowsCount: Array.isArray(dto.rows) ? dto.rows.length : 0,
  });

  // 単一DTOを merge と同じ形へ寄せる（見た目・ロジック共通化）
  const pbId = String(dto.productBlueprintId ?? "").trim();
  const tbId = String(dto.tokenBlueprintId ?? "").trim();

  const vm = mergeDetailDTOs(pbId || "-", tbId || "-", [id], [dto]);

  console.log("[inventory/queryInventoryDetail] mapped viewModel", {
    inventoryKey: vm.inventoryKey,
    totalStock: vm.totalStock,
    rowsCount: vm.rows.length,
    rowsSample: vm.rows.slice(0, 5),
  });

  return vm;
}
