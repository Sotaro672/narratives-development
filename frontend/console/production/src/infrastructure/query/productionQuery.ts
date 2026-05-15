// frontend/console/production/src/infrastructure/query/productionQuery.ts

// ✅ API base は shell shared を single source of truth にする
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";

// ✅ auth headers は shell shared を single source of truth にする
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

// ProductBlueprint 一覧取得は既存の HTTP Repository を利用
import {
  listProductBlueprintsHTTP,
} from "../../../../productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP";

// ==============================
// 型定義（backend/internal/domain/production/repository_port.go に合わせる）
// ==============================

export type ModelQuantity = {
  modelId: string;
  quantity: number;
};

// CreateProductionInput（contract only）
export type CreateProductionInput = {
  productBlueprintId: string;
  assigneeId: string;
  models: ModelQuantity[];

  printed?: boolean | null;
  printedAt?: string | null;

  createdBy?: string | null;
  createdAt?: string | null;
};

// Filter（contract only）
export type ProductionFilter = {
  id?: string;
  productBlueprintId?: string;
  assigneeId?: string;
  modelId?: string;
  printed?: boolean | null;
};

// backend の Production モデルに対応する最小限の一覧用型（Printed に統一）
export type ProductionSummary = {
  id: string;
  productBlueprintId: string;
  assigneeId: string | null;

  printed?: boolean | null;
  printedAt?: string | null;

  createdAt?: string | null;
  updatedAt?: string | null;

  // 他にも models 等が返ってきていても OK
  [key: string]: unknown;
};

// 商品設計 + 紐づく Production の一覧行
export type ProductBlueprintWithProductionsRow = {
  productBlueprintId: string;
  productName: string;
  brandId: string;
  productions: ProductionSummary[];
};

// Production 一覧 API からのレスポンス想定（一覧用途）
type ProductionListResponse = ProductionSummary;

// ==============================
// Production 一覧取得（生データ）
// ==============================
export async function listProductionsHTTP(): Promise<ProductionListResponse[]> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/productions`, {
    method: "GET",
    headers: {
      ...headers,
    },
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `生産計画一覧の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductionListResponse[];
}

// ==============================
// productBlueprintId ごとに Production を紐付けた一覧を返すクエリ
// ==============================
export async function fetchProductBlueprintsWithProductions(): Promise<
  ProductBlueprintWithProductionsRow[]
> {
  // 1. 商品設計一覧を取得
  const blueprints = await listProductBlueprintsHTTP();

  // 2. Production 一覧を取得
  const productions = await listProductionsHTTP();

  // 3. productBlueprintId ごとに紐付ける
  const rows: ProductBlueprintWithProductionsRow[] = blueprints.map((pb) => {
    const relatedProductions = productions
      .filter((p) => p.productBlueprintId === pb.id)
      .map<ProductionSummary>((p) => ({
        ...p,
        id: String(p.id ?? ""),
        productBlueprintId: String(p.productBlueprintId ?? ""),
        assigneeId: (p.assigneeId ?? null) as string | null,

        printed:
          typeof (p as any).printed === "boolean"
            ? ((p as any).printed as boolean)
            : (p as any).Printed === true
              ? true
              : (p as any).Printed === false
                ? false
                : (p as any).printed ?? null,

        printedAt: ((p as any).printedAt ?? (p as any).PrintedAt ?? null) as
          | string
          | null,

        createdAt: ((p as any).createdAt ?? (p as any).CreatedAt ?? null) as
          | string
          | null,
        updatedAt: ((p as any).updatedAt ?? (p as any).UpdatedAt ?? null) as
          | string
          | null,
      }));

    return {
      productBlueprintId: pb.id,
      productName: pb.productName,
      brandId: pb.brandId, // ← brandName ではなく brandId
      productions: relatedProductions,
    };
  });

  return rows;
}

// ==============================
// 特定の productBlueprintId に紐づく Production だけを取得
// ==============================
export async function listProductionsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ProductionSummary[]> {
  const all = await listProductionsHTTP();

  return all
    .filter((p) => p.productBlueprintId === productBlueprintId)
    .map<ProductionSummary>((p) => ({
      ...p,
      id: String(p.id ?? ""),
      productBlueprintId: String(p.productBlueprintId ?? ""),
      assigneeId: (p.assigneeId ?? null) as string | null,

      printed:
        typeof (p as any).printed === "boolean"
          ? ((p as any).printed as boolean)
          : (p as any).Printed === true
            ? true
            : (p as any).Printed === false
              ? false
              : (p as any).printed ?? null,

      printedAt: ((p as any).printedAt ?? (p as any).PrintedAt ?? null) as
        | string
        | null,

      createdAt: ((p as any).createdAt ?? (p as any).CreatedAt ?? null) as
        | string
        | null,
      updatedAt: ((p as any).updatedAt ?? (p as any).UpdatedAt ?? null) as
        | string
        | null,
    }));
}