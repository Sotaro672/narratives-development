// frontend/console/production/src/application/productionManagementService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";

/** ソートキー */
export type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

/** 一覧表示用に totalQuantity などを付与した行型（内部用） */
export type ProductionRow = Production & {
  totalQuantity: number;
  assigneeName?: string;

  /** ★ productBlueprintName をログ用・表示用に保持（現状は productBlueprintId と同じ） */
  productBlueprintName?: string;
};

/** 画面表示用の行型 */
export type ProductionRowView = {
  id: string;
  productBlueprintId: string;
  productBlueprintName: string;
  assigneeId: string;
  assigneeName: string;
  status: ProductionStatus;
  totalQuantity: number;
  printedAtLabel: string;
  createdAtLabel: string;
};

const toTs = (iso?: string | null): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

const formatDate = (iso?: string | null): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

/** Production 一覧取得 */
export async function loadProductionRows(): Promise<ProductionRow[]> {
  const items = await listProductionsHTTP();

  console.log("[productionManagementService] fetched items:", items);

  const rows: ProductionRow[] = items.map((raw: any) => {
    const rawModels = Array.isArray(raw.models)
      ? raw.models
      : Array.isArray(raw.Models)
        ? raw.Models
        : [];

    const totalQuantity = rawModels.reduce(
      (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
      0,
    );

    const blueprintId =
      raw.productBlueprintId ?? raw.ProductBlueprintID ?? "";

    const row: ProductionRow = {
      ...(raw as Production),

      id: raw.id ?? raw.ID ?? "",
      productBlueprintId: blueprintId,

      /** ★ 本来は backend から productBlueprintName を返すべきだが、なければ ID を使う */
      productBlueprintName: raw.productBlueprintName ?? blueprintId,

      assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",
      assigneeName: raw.assigneeName ?? raw.AssigneeName ?? "",

      status: (raw.status ?? raw.Status ?? "") as ProductionStatus,
      printedAt: raw.printedAt ?? raw.PrintedAt ?? null,
      createdAt: raw.createdAt ?? raw.CreatedAt ?? null,
      updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? null,
      models: rawModels,

      totalQuantity,
    };

    return row;
  });

  console.log(
    "[productionManagementService] rows with totalQuantity (normalized):",
    rows,
    // ★ productBlueprintName を含めてログ出力
    rows.map((r) => ({
      id: r.id,
      productBlueprintId: r.productBlueprintId,
      productBlueprintName: r.productBlueprintName,
      assigneeId: r.assigneeId,
      assigneeName: r.assigneeName,
    })),
  );

  return rows;
}

/** rows → viewRows 変換（フィルタ + ソート + 表示用ラベル整形） */
export function buildRowsView(params: {
  baseRows: ProductionRow[];
  blueprintFilter: string[];
  assigneeFilter: string[];
  statusFilter: ProductionStatus[];
  sortKey: SortKey;
  sortDir: "asc" | "desc" | null;
}): ProductionRowView[] {
  const {
    baseRows,
    blueprintFilter,
    assigneeFilter,
    statusFilter,
    sortKey,
    sortDir,
  } = params;

  // ===== フィルタ適用 =====
  let data = baseRows.filter((p) => {
    if (
      blueprintFilter.length > 0 &&
      !blueprintFilter.includes(p.productBlueprintId)
    ) {
      return false;
    }
    if (assigneeFilter.length > 0 && !assigneeFilter.includes(p.assigneeId)) {
      return false;
    }
    if (statusFilter.length > 0 && !statusFilter.includes(p.status)) {
      return false;
    }
    return true;
  });

  // ===== ソート適用 =====
  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      if (sortKey === "totalQuantity") {
        const av = a.totalQuantity;
        const bv = b.totalQuantity;
        return sortDir === "asc" ? av - bv : bv - av;
      }
      const av = toTs(a[sortKey]);
      const bv = toTs(b[sortKey]);
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  // ===== 表示用変換 =====
  const view = data.map<ProductionRowView>((p) => ({
    id: p.id,
    productBlueprintId: p.productBlueprintId,
    productBlueprintName: p.productBlueprintName ?? "",
    assigneeId: p.assigneeId,
    assigneeName: p.assigneeName ?? "",
    status: p.status,
    totalQuantity: p.totalQuantity,
    printedAtLabel: formatDate(p.printedAt),
    createdAtLabel: formatDate(p.createdAt),
  }));

  console.log(
    "[productionManagementService] view rows:",
    view.map((v) => ({
      id: v.id,
      productBlueprintId: v.productBlueprintId,
      productBlueprintName: v.productBlueprintName,
      assigneeId: v.assigneeId,
      assigneeName: v.assigneeName,
    })),
  );

  return view;
}
