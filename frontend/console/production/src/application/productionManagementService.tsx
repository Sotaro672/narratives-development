// frontend/console/production/src/application/productionManagementService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";

/** ソートキー */
export type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

/**
 * 一覧表示用に totalQuantity などを付与した行型（内部用）
 * - Backend DTO に合わせて camelCase を主とする
 */
export type ProductionRow = Omit<
  Production,
  "assigneeId" | "assigneeName" | "productName" | "brandName"
> & {
  totalQuantity: number;

  /** backend から受け取る productName（なければ fallback で ID を使う） */
  productName: string;

  /** backend から受け取る brandName（なければ空文字） */
  brandName: string;

  /** UI では常に string として扱う */
  assigneeId: string;
  assigneeName: string;
};

/** 画面表示用の行型 */
export type ProductionRowView = {
  id: string;
  productBlueprintId: string;
  productName: string;
  assigneeId: string;
  assigneeName: string;
  status: ProductionStatus;
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

const pad2 = (n: number): string => String(n).padStart(2, "0");

/**
 * yyyy/mm/dd hh:mm
 * - backend は time.Time を JSON で返すので、ここでは ISO 文字列として受ける想定
 */
const formatDateTime = (iso?: string | null): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";

  const y = d.getFullYear();
  const m = pad2(d.getMonth() + 1);
  const day = pad2(d.getDate());
  const hh = pad2(d.getHours());
  const mm = pad2(d.getMinutes());

  return `${y}/${m}/${day} ${hh}:${mm}`;
};

const asString = (v: any): string => (typeof v === "string" ? v : "");
const asNonEmptyString = (v: any): string =>
  typeof v === "string" && v.trim() ? v.trim() : "";

/** Production 一覧取得 */
export async function loadProductionRows(): Promise<ProductionRow[]> {
  const items = await listProductionsHTTP();

  const rows: ProductionRow[] = items.map((raw: any) => {
    // ✅ DTO 優先（camelCase）、互換のため PascalCase も fallback
    const rawModels = Array.isArray(raw.models)
      ? raw.models
      : Array.isArray(raw.Models)
        ? raw.Models
        : [];

    // DTO には totalQuantity が来る想定だが、互換のため再計算 fallback も残す
    const computedTotalQuantity = rawModels.reduce(
      (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
      0,
    );

    const totalQuantity =
      typeof raw.totalQuantity === "number"
        ? raw.totalQuantity
        : typeof raw.TotalQuantity === "number"
          ? raw.TotalQuantity
          : computedTotalQuantity;

    const blueprintId = asNonEmptyString(
      raw.productBlueprintId ?? raw.ProductBlueprintID ?? "",
    );

    const productName =
      asNonEmptyString(raw.productName ?? raw.ProductName) || blueprintId;

    const assigneeId = asString(raw.assigneeId ?? raw.AssigneeID ?? "");
    const assigneeName = asString(raw.assigneeName ?? raw.AssigneeName ?? "");
    const brandName = asString(raw.brandName ?? raw.BrandName ?? "");

    const row: ProductionRow = {
      ...(raw as Production),

      id: asNonEmptyString(raw.id ?? raw.ID ?? ""),
      productBlueprintId: blueprintId,

      productName,
      brandName,

      assigneeId,
      assigneeName,

      status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

      // time は backend の time.Time が ISO で来る前提（string / null）
      printedAt: (raw.printedAt ?? raw.PrintedAt ?? null) as any,
      createdAt: (raw.createdAt ?? raw.CreatedAt ?? null) as any,
      updatedAt: (raw.updatedAt ?? raw.UpdatedAt ?? null) as any,

      models: rawModels as any,

      totalQuantity,
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
    status: p.status,
    totalQuantity: p.totalQuantity,
    printedAtLabel: formatDateTime((p as any).printedAt ?? null),
    createdAtLabel: formatDateTime((p as any).createdAt ?? null),
    brandName: p.brandName,
  }));

  return view;
}
