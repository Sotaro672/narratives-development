// frontend/console/production/src/application/printService.tsx

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE as BACKEND_API_BASE } from "../infrastructure/http/productionRepositoryHTTP";

// 印刷用の行型（ProductionDetail 画面側から渡す）
// modelId / modelVariationId のどちらかが入っていれば OK にする
export type PrintRow = {
  modelId?: string;
  modelVariationId?: string;
  quantity: number | null | undefined;
};

/** products 一覧の簡易型（ダイアログ表示用） */
export type ProductSummaryForPrint = {
  id: string;
  modelId: string;
  productionId: string;
};

/* ---------------------------------------------------------
 * 共通: Firebase ID トークン取得
 * --------------------------------------------------------- */
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

/* ---------------------------------------------------------
 * Product 作成API（印刷用）: 1件分
 * --------------------------------------------------------- */
async function createProductHTTP(payload: {
  modelId: string;
  productionId: string;
  printedAt: string; // ISO 文字列
  printedBy?: string | null;
}): Promise<void> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(`${BACKEND_API_BASE}/products`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      modelId: payload.modelId,
      productionId: payload.productionId,
      printedAt: payload.printedAt,
      printedBy: payload.printedBy ?? null,
      // inspectionResult / connectedToken などは
      // バックエンド側でデフォルト値(notYet/null)を設定する想定
    }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Product create failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }
}

/* ---------------------------------------------------------
 * 印刷用: 各 modelId の quantity 分だけ Product を作成
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<void> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  // 印刷タイミングの時刻（全Productで共通とする）
  const printedAtISO = new Date().toISOString();

  // printedBy は Firebase ユーザーIDなどを使用
  const user = auth.currentUser;
  const printedBy = user?.uid ?? null;

  const tasks: Promise<void>[] = [];

  rows.forEach((row) => {
    const q = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity as number)))
      : 0;

    // modelId or modelVariationId → 実際に送る modelId に正規化
    const rawModelId = row.modelId ?? row.modelVariationId ?? "";
    const modelId = rawModelId.trim();

    // ID が空 or quantity 0 以下はスキップ
    if (!modelId || q <= 0) {
      return;
    }

    for (let i = 0; i < q; i += 1) {
      tasks.push(
        createProductHTTP({
          modelId,
          productionId: id,
          printedAt: printedAtISO,
          printedBy,
        }),
      );
    }
  });

  await Promise.all(tasks);
}

/* ---------------------------------------------------------
 * 指定 productionId を持つ products 一覧を取得
 *   GET /products?productionId={id} を想定
 *   （※ バックエンド側でこのエンドポイントが必要）
 * --------------------------------------------------------- */
export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(id);

  const res = await fetch(
    `${BACKEND_API_BASE}/products?productionId=${safeId}`,
    {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    },
  );

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `List products failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = (await res.json()) as any[];

  return raw
    .map((p) => ({
      id: p.id ?? p.ID ?? "",
      modelId: p.modelId ?? p.ModelID ?? "",
      productionId: p.productionId ?? p.ProductionID ?? "",
    }))
    .filter((p) => p.id && p.productionId === id);
}
