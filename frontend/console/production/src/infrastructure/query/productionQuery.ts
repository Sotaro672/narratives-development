// frontend/console/production/src/infrastructure/query/productionQuery.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ProductBlueprint 一覧取得は既存の HTTP Repository を利用
import {
  listProductBlueprintsHTTP,
} from "../../../../productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP";

// Production 用の API_BASE は HTTP Repository と共通利用
import { API_BASE as PRODUCTION_API_BASE } from "../http/productionRepositoryHTTP";

// ==============================
// 共通: Firebase 認証トークン取得
// ==============================
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

// ==============================
// 型定義
// ==============================

// backend の Production モデルに対応する最小限の一覧用型
export type ProductionStatus = "draft" | "planned" | "in_progress";

export type ProductionSummary = {
  id: string;
  productBlueprintId: string;
  assigneeId: string | null;
  status: ProductionStatus;
  createdAt?: string | null;
  updatedAt?: string | null;
};

// 商品設計 + 紐づく Production の一覧行
export type ProductBlueprintWithProductionsRow = {
  productBlueprintId: string;
  productName: string;
  brandId: string;
  productions: ProductionSummary[];
};

// Production 一覧 API からのレスポンス想定
// （必要に応じてフィールドを追加してください）
type ProductionListResponse = {
  id: string;
  productBlueprintId: string;
  assigneeId: string | null;
  status: ProductionStatus;
  createdAt?: string | null;
  updatedAt?: string | null;

  // 他にも models, printedAt などが返ってきていても OK
  // ここでは使わないので省略
  [key: string]: unknown;
};

// ==============================
// Production 一覧取得（生データ）
// ==============================
export async function listProductionsHTTP(): Promise<ProductionListResponse[]> {
  const idToken = await getIdTokenOrThrow();

  const res = await fetch(`${PRODUCTION_API_BASE}/productions`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
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
export async function fetchProductBlueprintsWithProductions(): Promise<ProductBlueprintWithProductionsRow[]> {
  // 1. 商品設計一覧を取得
  const blueprints = await listProductBlueprintsHTTP();

  // 2. Production 一覧を取得
  const productions = await listProductionsHTTP();

  // 3. productBlueprintId ごとに紐付ける
  const rows: ProductBlueprintWithProductionsRow[] = blueprints.map((pb) => {
    const relatedProductions = productions
      .filter((p) => p.productBlueprintId === pb.id)
      .map<ProductionSummary>((p) => ({
        id: p.id,
        productBlueprintId: p.productBlueprintId,
        assigneeId: (p.assigneeId ?? null) as string | null,
        status: p.status,
        createdAt: (p.createdAt ?? null) as string | null,
        updatedAt: (p.updatedAt ?? null) as string | null,
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
      id: p.id,
      productBlueprintId: p.productBlueprintId,
      assigneeId: (p.assigneeId ?? null) as string | null,
      status: p.status,
      createdAt: (p.createdAt ?? null) as string | null,
      updatedAt: (p.updatedAt ?? null) as string | null,
    }));
}
