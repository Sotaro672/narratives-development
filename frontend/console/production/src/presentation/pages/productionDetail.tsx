// frontend/console/production/src/presentation/pages/productionDetail.tsx

import React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import ProductionQuantityCard from "../components/productionQuantityCard";

import { useProductionDetail } from "../hook/useProductionDetail";
import "../styles/production.css";

import LogCard from "../../../../log/src/presentation/LogCard";
import { Button } from "../../../../shell/src/shared/ui/button";

// ProductBlueprintCard の型用
import type {
  ItemType,
  Fit,
} from "../../../../productBlueprint/src/domain/entity/catalog";

// ★ usePrintCard Hook（print_log + QR 情報取得）
import { usePrintCard } from "../../../../product/src/presentation/hook/usePrintCard";

// ★ 分離した印刷カードコンポーネント
import PrintCard from "../../../../product/src/presentation/component/printCard";

export default function ProductionDetail() {
  const {
    // モード関連
    mode,
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,

    // AdminCard 用モード
    adminMode,

    // 戻る
    onBack,
    onSave,

    // データ関連
    productionId,
    production,
    loading,
    error,
    creator,
    quantityRows,
    setQuantityRows,
    productBlueprint,
    pbLoading,
    pbError,
  } = useProductionDetail();

  const assigneeDisplay =
    production?.assigneeName ||
    production?.assigneeId ||
    "担当者が設定されていません";

  const createdAtLabel = production?.createdAt
    ? new Date(production.createdAt).toLocaleDateString("ja-JP")
    : "-";

  // ==========================
  // usePrintCard: 印刷 + print_log 取得
  // ==========================
  const {
    onPrint,
    printLogs,
    printing,
    error: printError,
  } = usePrintCard({
    productionId: productionId ?? null,
    hasProduction: !!production,
    rows: quantityRows,
  });

  // ==========================
  // 印刷結果ダイアログ用 state
  // ==========================
  const [printDialogOpen, setPrintDialogOpen] = React.useState(false);

  // ==========================
  // ヘッダー操作
  // ==========================
  const handleEnterEdit = React.useCallback(() => {
    switchToEdit();
  }, [switchToEdit]);

  const handleCancelEdit = React.useCallback(() => {
    switchToView();
  }, [switchToView]);

  const handleSave = React.useCallback(() => {
    void onSave();
  }, [onSave]);

  const handleDelete = React.useCallback(() => {
    // TODO: 削除処理
  }, []);

  // ==========================
  // ★ 印刷ボタン押下時処理
  //   - usePrintCard.onPrint を呼び出し
  //   - 戻り値として Hook 内部に保持された printLogs を
  //     ダイアログで表示する
  // ==========================
  const handlePrint = React.useCallback(async () => {
    if (!productionId) {
      window.alert("productionId が取得できませんでした。");
      return;
    }

    const ok = window.confirm(
      "印刷",
    );
    if (!ok) return;

    // usePrintCard 内で:
    //   1. Product を作成
    //   2. print_log を作成
    //   3. print_log 一覧（QR ペイロード付き）を取得して保持
    await onPrint();

    // Hook が保持している printLogs をダイアログで確認する
    setPrintDialogOpen(true);
  }, [productionId, onPrint]);

  // ==========================
  // 戻る
  // ==========================
  const handleBack = React.useCallback(() => {
    onBack();
  }, [onBack]);

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="生産詳細"
        onBack={handleBack}
        onEdit={isViewMode ? handleEnterEdit : undefined}
        onDelete={isEditMode ? handleDelete : undefined}
        onCancel={isEditMode ? handleCancelEdit : undefined}
        onSave={isEditMode ? handleSave : undefined}
      >
        {/* ========== 左カラム ========== */}
        <div className="space-y-4">
          {loading && (
            <div className="flex h-full items-center justify-center text-gray-500">
              生産情報を読み込み中です…
            </div>
          )}

          {!loading && error && (
            <div className="flex h-full items-center justify-center text-red-500">
              {error}
            </div>
          )}

          {!loading && !error && !production && (
            <div className="flex h-full items-center justify-center text-gray-500">
              対象の生産情報が見つかりません。
            </div>
          )}

          {!loading && !error && production && (
            <>
              {pbLoading && (
                <div className="p-4 text-gray-500">商品設計を読み込み中…</div>
              )}

              {!pbLoading && pbError && (
                <div className="p-4 text-red-500">{pbError}</div>
              )}

              {!pbLoading && !pbError && productBlueprint && (
                <ProductBlueprintCard
                  mode="view"
                  productName={productBlueprint.productName}
                  brand={production.brandName ?? ""}
                  brandId={productBlueprint.brandId}
                  itemType={productBlueprint.itemType as ItemType}
                  fit={productBlueprint.fit as Fit}
                  materials={productBlueprint.material}
                  weight={productBlueprint.weight}
                  washTags={productBlueprint.qualityAssurance}
                  productIdTag={productBlueprint.productIdTag}
                />
              )}

              <ProductionQuantityCard
                title="モデル別 生産数一覧"
                rows={quantityRows}
                mode={isEditMode ? "edit" : "view"}
                onChangeRows={isEditMode ? setQuantityRows : undefined}
              />
              {isViewMode && (
                <PrintCard printing={printing} onClick={handlePrint} />
              )}
            </>
          )}
        </div>

        {/* ========== 右カラム ========== */}
        <div className="space-y-4">
          <AdminCard
            title="管理情報"
            assigneeName={assigneeDisplay}
            assigneeCandidates={[]}
            loadingMembers={false}
            createdByName={creator}
            createdAt={createdAtLabel}
            mode={adminMode}
            onSelectAssignee={() => {}}
          />

          <LogCard
            title="更新履歴"
            logs={[]}
            emptyText="更新履歴はまだありません。"
          />
        </div>
      </PageStyle>

      {/* ==========================
          ★ print_log 一覧 + QR ペイロード ダイアログ
         ========================== */}
      {printDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-2xl rounded-lg bg-white p-4 shadow-lg">
            <h2 className="mb-2 text-lg font-semibold">
              発行された print_log 一覧
            </h2>

            {printError && (
              <p className="mb-2 text-sm text-red-600">{printError}</p>
            )}

            {printLogs.length === 0 ? (
              <p className="text-sm text-gray-600">
                該当する print_log はありません。
              </p>
            ) : (
              <div className="max-h-80 space-y-3 overflow-y-auto rounded border border-gray-200 bg-gray-50 p-3 text-sm">
                {printLogs.map((log) => (
                  <div
                    key={log.id}
                    className="rounded border border-gray-200 bg-white p-2"
                  >
                    <div className="mb-1 flex items-center justify-between">
                      <span className="font-semibold">
                        print_log ID: {log.id}
                      </span>
                      <span className="text-xs text-gray-500">
                        printedAt: {log.printedAt}
                      </span>
                    </div>
                    <div className="mb-1 text-xs text-gray-500">
                      printedBy: {log.printedBy}
                    </div>

                    {log.productIds.length === 0 ? (
                      <div className="text-xs text-gray-500">
                        productId は記録されていません。
                      </div>
                    ) : (
                      <ul className="space-y-1 text-xs font-mono">
                        {log.productIds.map((pid, idx) => (
                          <li key={`${log.id}-${pid}-${idx}`}>
                            <span className="font-semibold">productId:</span>{" "}
                            {pid}
                            {log.qrPayloads[idx] && (
                              <span className="ml-2 text-[11px] text-blue-600 underline">
                                {/* QR ペイロードは URL を想定。 */}
                                <a
                                  href={log.qrPayloads[idx]}
                                  target="_blank"
                                  rel="noreferrer"
                                >
                                  QR ペイロードを開く
                                </a>
                              </span>
                            )}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
            )}

            <div className="mt-4 flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setPrintDialogOpen(false)}
              >
                閉じる
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
