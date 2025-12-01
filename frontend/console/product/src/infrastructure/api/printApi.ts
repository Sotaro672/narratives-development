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

/** products 一覧の簡易型（ダイアログ表示用） */
export type ProductSummaryForPrint = {
  id: string;
  modelId: string;
  productionId: string;
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

  const raw = await fetchPrintLogsByProductionId(id);

  return raw
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

      return {
        id: log.id ?? log.ID ?? "",
        productionId: log.productionId ?? log.ProductionID ?? "",
        productIds,
        printedBy: log.printedBy ?? log.PrintedBy ?? "",
        printedAt:
          log.printedAt ??
          log.PrintedAt ??
          "", // ISO 文字列 or 空文字（フロント側で必要ならパース）
        qrPayloads,
      } as PrintLogForPrint;
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

  // 印刷タイミングの時刻（全 Product で共通とする）
  const printedAtISO = new Date().toISOString();

  const tasks: Promise<void>[] = [];

  rows.forEach((row) => {
    const q = Number.isFinite(Number(row.quantity))
      ? Math.max(0, Math.floor(Number(row.quantity as number)))
      : 0;

    const rawModelId = row.modelId ?? "";
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
        }),
      );
    }
  });

  // 1. Product を全件作成
  await Promise.all(tasks);

  // 2. print_log 作成リクエストをバックエンドへ送信
  await createPrintLogsHTTP(id);

  // 3. print_log を取得（ここで BuildProductQRValue 済みの qrPayloads を受け取る想定）
  const logs = await listPrintLogsByProductionId(id);

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

  const raw = await fetchProductsByProductionId(id);

  return (raw as any[])
    .map((p) => ({
      id: p.id ?? p.ID ?? "",
      modelId: p.modelId ?? p.ModelID ?? "",
      productionId: p.productionId ?? p.ProductionID ?? "",
    }))
    .filter((p) => p.id && p.productionId === id);
}
