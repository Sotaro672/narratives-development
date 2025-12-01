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
// ★ Card コンポーネント群を追加
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

// ProductBlueprintCard の型用
import type {
  ItemType,
  Fit,
} from "../../../../productBlueprint/src/domain/entity/catalog";

// ★ 印刷用サービスを import
import {
  createProductsForPrint,
  listProductsByProductionId,
  type PrintRow,
  type ProductSummaryForPrint,
} from "../../../../product/src/application/printService";

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
  // 印刷結果ダイアログ用 state
  // ==========================
  const [printing, setPrinting] = React.useState(false);
  const [printDialogOpen, setPrintDialogOpen] = React.useState(false);
  const [printedProducts, setPrintedProducts] = React.useState<
    ProductSummaryForPrint[]
  >([]);
  const [printError, setPrintError] = React.useState<string | null>(null);

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
  // ==========================
  const handlePrint = React.useCallback(async () => {
    if (!productionId) {
      window.alert("productionId が取得できませんでした。");
      return;
    }

    // 必要なら確認ダイアログ（残す）
    const ok = window.confirm(
      "印刷用の Product を発行します。同じ productionId を持つ productId 一覧を表示します。よろしいですか？",
    );
    if (!ok) return;

    try {
      setPrinting(true);
      setPrintError(null);

      // quantityRows(Create用) → PrintRow へ変換
      const rowsForPrint: PrintRow[] = quantityRows.map((row) => ({
        modelVariationId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      // 1) products 作成（印刷用）
      await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      // 2) 同じ productionId を持つ products 一覧を取得
      const list = await listProductsByProductionId(productionId);

      setPrintedProducts(list);
      setPrintDialogOpen(true);
    } catch (e) {
      console.error(e);
      setPrintError("印刷用 Product の発行または一覧取得に失敗しました。");
      window.alert("印刷用 Product の発行または一覧取得に失敗しました。");
    } finally {
      setPrinting(false);
    }
  }, [productionId, quantityRows]);

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

              {/* ===== 印刷カード（Product 発行 + 一覧ダイアログ） ===== */}
              <Card className="print-card">
                <CardHeader>
                  <CardTitle>商品IDタグ用 Product を発行する</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="print-card__content">
                    <Button
                      variant="solid"
                      size="lg"
                      onClick={handlePrint}
                      className="w-full max-w-xs"
                      disabled={printing}
                    >
                      {printing ? "発行中..." : "印刷用 Product を発行"}
                    </Button>
                  </div>
                </CardContent>
              </Card>
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
          ★ Product 一覧ダイアログ
         ========================== */}
      {printDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-lg rounded-lg bg-white p-4 shadow-lg">
            <h2 className="mb-2 text-lg font-semibold">
              発行された Product ID 一覧
            </h2>

            {printError && (
              <p className="mb-2 text-sm text-red-600">{printError}</p>
            )}

            {printedProducts.length === 0 ? (
              <p className="text-sm text-gray-600">
                該当する Product はありません。
              </p>
            ) : (
              <ul className="max-h-64 space-y-1 overflow-y-auto rounded border border-gray-200 bg-gray-50 p-2 text-sm font-mono">
                {printedProducts.map((p) => (
                  <li key={p.id}>
                    <span className="font-semibold">productId:</span> {p.id}
                    {p.modelId && (
                      <span className="ml-2 text-xs text-gray-500">
                        (modelId: {p.modelId})
                      </span>
                    )}
                  </li>
                ))}
              </ul>
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
