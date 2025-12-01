// frontend/console/product/src/infrastructure/api/printApi.ts

import {
  createProductHTTP,
  createPrintLogsHTTP,
  fetchPrintLogsByProductionId,
  fetchProductsByProductionId,
} from "../repository/productRepositoryHTTP";

// 印刷用の行型（ProductionDetail 画面側から渡す）
// modelId のみ（modelVariationId は事前に解決済みとする）
export type PrintRow = {
  modelId?: string;
  quantity: number | null | undefined;
};

/** products 一覧の簡易型（ダイアログ表示用 + QR ラベル用） */
export type ProductSummaryForPrint = {
  id: string;
  modelId: string;
  productionId: string;
  // ★ 追加: QR ラベルで使う modelNumber（backend から渡される想定）
  modelNumber?: string;
};

/** print_log 一覧の型（QR ペイロードを含めてフロントで扱う想定） */
export type PrintLogForPrint = {
  id: string;
  productionId: string;
  productIds: string[];
  printedBy: string;
  printedAt: string; // ISO 文字列
  // ★ バックエンド側で BuildProductQRValue によって生成された QR ペイロード（URL など）
  qrPayloads: string[];
};

/* ---------------------------------------------------------
 * print_log 取得（API ラッパ）
 *   GET /products/print-logs?productionId={id}
 *   → PrintLogForPrint[] を返す
 * --------------------------------------------------------- */
export async function listPrintLogsByProductionId(
  productionId: string,
): Promise<PrintLogForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  // eslint-disable-next-line no-console
  console.log("[printApi] listPrintLogsByProductionId called:", { productionId: id });

  const raw = await fetchPrintLogsByProductionId(id);

  // eslint-disable-next-line no-console
  console.log("[printApi] fetchPrintLogsByProductionId raw:", raw);

  if (!raw) {
    // eslint-disable-next-line no-console
    console.warn(
      "[printApi] listPrintLogsByProductionId: raw is null/undefined, return []",
    );
    return [];
  }

  if (!Array.isArray(raw)) {
    // eslint-disable-next-line no-console
    console.warn(
      "[printApi] listPrintLogsByProductionId: raw is not array. Treat as empty.",
      raw,
    );
    return [];
  }

  return (raw as any[])
    .map((log: any) => {
      const productIds: string[] = Array.isArray(log.productIds)
        ? log.productIds
            .map((v: unknown) => String(v))
            .filter((v: string) => !!v)
        : [];

      const qrPayloads: string[] = Array.isArray(log.qrPayloads)
        ? log.qrPayloads
            .map((v: unknown) => String(v))
            .filter((v: string) => !!v)
        : [];

      const mapped: PrintLogForPrint = {
        id: log.id ?? log.ID ?? "",
        productionId: log.productionId ?? log.ProductionID ?? "",
        productIds,
        // backend 側では printedBy は廃止済みなので、あれば拾う・なければ空文字
        printedBy: log.printedBy ?? log.PrintedBy ?? "",
        printedAt:
          log.printedAt ??
          log.PrintedAt ??
          "", // ISO 文字列 or 空文字（フロント側で必要ならパース）
        qrPayloads,
      };

      // eslint-disable-next-line no-console
      console.log("[printApi] listPrintLogsByProductionId mapped log:", mapped);

      return mapped;
    })
    .filter((log) => log.id && log.productionId === id);
}

/* ---------------------------------------------------------
 * 印刷用: 各 modelId の quantity 分だけ Product を作成
 *
 * 1. /products に対して Product を必要数作成
 * 2. /products/print-logs に対して print_log 作成リクエストを送信
 * 3. /products/print-logs?productionId=... で print_log を取得
 * 4. 取得した print_log 一覧（QR ペイロード付き）を返す
 *
 * → usePrintCard.tsx からは、この戻り値を受け取って
 *   「どの productId に対してどの QR ペイロードか」を利用できる
 * --------------------------------------------------------- */
export async function createProductsForPrint(params: {
  productionId: string;
  rows: PrintRow[];
}): Promise<PrintLogForPrint[]> {
  const { productionId, rows } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  // eslint-disable-next-line no-console
  console.log("[printApi] createProductsForPrint called:", { productionId: id, rows });

  // 印刷タイミングの時刻（全 Product で共通とする）
  const printedAtISO = new Date().toISOString();
  // eslint-disable-next-line no-console
  console.log("[printApi] createProductsForPrint printedAtISO:", printedAtISO);

  const tasks: Promise<void>[] = [];

  rows.forEach((row) => {
    const q = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity as number)))
      : 0;

    const rawModelId = row.modelId ?? "";
    const modelId = rawModelId.trim();

    // ID が空 or quantity 0 以下はスキップ
    if (!modelId || q <= 0) {
      // eslint-disable-next-line no-console
      console.log("[printApi] createProductsForPrint skip row:", { row, q, modelId });
      return;
    }

    // eslint-disable-next-line no-console
    console.log(
      "[printApi] createProductsForPrint enqueue createProductHTTP:",
      { modelId, q },
    );

    for (let i = 0; i < q; i += 1) {
      tasks.push(
        createProductHTTP({
          modelId,
          productionId: id,
          printedAt: printedAtISO,
        }),
      );
    }
  });

  // eslint-disable-next-line no-console
  console.log(
    "[printApi] createProductsForPrint total createProductHTTP tasks:",
    tasks.length,
  );

  // 1. Product を全件作成
  await Promise.all(tasks);

  // eslint-disable-next-line no-console
  console.log("[printApi] createProductsForPrint: all createProductHTTP done");

  // 2. print_log 作成リクエストをバックエンドへ送信
  const createLogsResult = await createPrintLogsHTTP(id);
  // eslint-disable-next-line no-console
  console.log("[printApi] createPrintLogsHTTP result:", createLogsResult);

  // 3. print_log を取得（ここで BuildProductQRValue 済みの qrPayloads を受け取る想定）
  const logs = await listPrintLogsByProductionId(id);

  // eslint-disable-next-line no-console
  console.log("[printApi] createProductsForPrint logs from backend:", logs);

  // 4. 呼び出し元（usePrintCard.tsx など）へ返却
  return logs;
}

/* ---------------------------------------------------------
 * 指定 productionId を持つ products 一覧を取得
 *   GET /products?productionId={id} を想定
 * --------------------------------------------------------- */
export async function listProductsByProductionId(
  productionId: string,
): Promise<ProductSummaryForPrint[]> {
  const id = productionId.trim();
  if (!id) return [];

  // eslint-disable-next-line no-console
  console.log("[printApi] listProductsByProductionId called:", { productionId: id });

  const raw = await fetchProductsByProductionId(id);

  // eslint-disable-next-line no-console
  console.log("[printApi] fetchProductsByProductionId raw:", raw);

  if (!raw) {
    // eslint-disable-next-line no-console
    console.warn(
      "[printApi] listProductsByProductionId: raw is null/undefined, return []",
    );
    return [];
  }

  if (!Array.isArray(raw)) {
    // eslint-disable-next-line no-console
    console.warn(
      "[printApi] listProductsByProductionId: raw is not array. Treat as empty.",
      raw,
    );
    return [];
  }

  const mapped = (raw as any[])
    .map((p) => ({
      id: p.id ?? p.ID ?? "",
      modelId: p.modelId ?? p.ModelID ?? "",
      productionId: p.productionId ?? p.ProductionID ?? "",
      // ★ 追加: backend が返した modelNumber を拾う
      modelNumber: p.modelNumber ?? p.ModelNumber ?? "",
    }))
    .filter((p) => p.id && p.productionId === id);

  // eslint-disable-next-line no-console
  console.log("[printApi] listProductsByProductionId mapped:", mapped);

  return mapped;
}
