// frontend/console/production/src/application/productionManagementService.tsx

import type { Production } from "../../../shell/src/shared/types/production";

// ✅ date label util (single source of truth)
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";

// ✅ production list fetcher (was missing)
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";

/** ソートキー */
export type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

/**
 * 一覧表示用に totalQuantity などを付与した行型（内部用）
 * - Backend DTO (PascalCase) を正とする
 */
export type ProductionRow = Omit<
  Production,
  "assigneeId" | "assigneeName" | "productName" | "brandName"
> & {
  totalQuantity: number;
  productName: string;
  brandName: string;

  /** UI では常に string として扱う */
  assigneeId: string;
  assigneeName: string;

  /** printed:boolean */
  printed: boolean;

  /** 表示用ラベル（backend が返す / ない場合は util で作る） */
  printedAtLabel: string;
  createdAtLabel: string;

  /** timestamps（backend が返す ISO string） */
  printedAt?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type ProductionRowView = {
  id: string;
  productBlueprintId: string;
  productName: string;
  assigneeId: string;
  assigneeName: string;
  printed: boolean;
  totalQuantity: number;
  printedAtLabel: string;
  createdAtLabel: string;
  brandName: string;
};

const toTs = (iso?: string | null): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

/** Production 一覧取得（backend DTO をそのまま利用） */
export async function loadProductionRows(): Promise<ProductionRow[]> {
  const items = await listProductionsHTTP();

  const rows: ProductionRow[] = (Array.isArray(items) ? items : []).map((raw: any) => {
    // ✅ backend DTO を正として直接参照
    const id: string = String(raw.ID ?? "");
    const productBlueprintId: string = String(raw.ProductBlueprintID ?? "");
    const assigneeId: string = String(raw.AssigneeID ?? "");
    const printed: boolean = Boolean(raw.Printed);

    const models = Array.isArray(raw.Models) ? raw.Models : [];

    const totalQuantity: number =
      typeof raw.totalQuantity === "number" ? raw.totalQuantity : 0;

    const productName: string = String(raw.productName ?? "");
    const brandName: string = String(raw.brandName ?? "");
    const assigneeName: string = String(raw.assigneeName ?? "");

    const printedAt: string | null =
      raw.PrintedAt != null ? String(raw.PrintedAt) : null;
    const createdAt: string | null =
      raw.CreatedAt != null ? String(raw.CreatedAt) : null;
    const updatedAt: string | null =
      raw.UpdatedAt != null ? String(raw.UpdatedAt) : null;

    // ✅ backend の label を優先しつつ、無い場合は util で生成（表示形式固定）
    const printedAtLabel: string =
      typeof raw.printedAtLabel === "string" && raw.printedAtLabel
        ? raw.printedAtLabel
        : safeDateTimeLabelJa(printedAt, "-");

    const createdAtLabel: string =
      typeof raw.createdAtLabel === "string" && raw.createdAtLabel
        ? raw.createdAtLabel
        : safeDateTimeLabelJa(createdAt, "-");

    const row: ProductionRow = {
      ...(raw as Production),

      id,
      productBlueprintId,

      assigneeId,
      assigneeName,

      printed,

      models: models as any,

      printedAt,
      createdAt,
      updatedAt,

      totalQuantity,
      productName,
      brandName,

      printedAtLabel,
      createdAtLabel,
    };

    return row;
  });

  return rows;
}

/** rows → viewRows 変換（フィルタ + ソート + 表示用ラベル整形） */
export function buildRowsView(params: {
  baseRows: ProductionRow[];
  blueprintFilter: string[];
  assigneeFilter: string[];
  printedFilter: boolean[]; // [true] / [false] / [true,false] / []
  sortKey: SortKey;
  sortDir: "asc" | "desc" | null;
}): ProductionRowView[] {
  const {
    baseRows,
    blueprintFilter,
    assigneeFilter,
    printedFilter,
    sortKey,
    sortDir,
  } = params;

  // ===== フィルタ適用 =====
  let data = (Array.isArray(baseRows) ? baseRows : []).filter((p) => {
    if (blueprintFilter.length > 0 && !blueprintFilter.includes(p.productBlueprintId)) {
      return false;
    }
    if (assigneeFilter.length > 0 && !assigneeFilter.includes(p.assigneeId)) {
      return false;
    }
    if (printedFilter.length > 0 && !printedFilter.includes(p.printed)) {
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
      const av = toTs((a as any)[sortKey] as any);
      const bv = toTs((b as any)[sortKey] as any);
      return sortDir === "asc" ? av - bv : bv - av;
    });
  }

  // ===== 表示用変換 =====
  const view = data.map<ProductionRowView>((p) => ({
    id: p.id,
    productBlueprintId: p.productBlueprintId,
    productName: p.productName,
    assigneeId: p.assigneeId,
    assigneeName: p.assigneeName,
    printed: p.printed,
    totalQuantity: p.totalQuantity,
    printedAtLabel: p.printedAtLabel,
    createdAtLabel: p.createdAtLabel,
    brandName: p.brandName,
  }));

  return view;
}