// frontend/console/production/src/presentation/pages/productionDetail.tsx

import React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import ProductionQuantityCard from "../components/productionQuantityCard";

import { useProductionDetail } from "../hook/useProductionDetail";
import "../styles/production.css";

import LogCard from "../../../../log/src/presentation/LogCard";

// ProductBlueprintCard の型用
import type {
  ItemType,
  Fit,
} from "../../../../productBlueprint/src/domain/entity/catalog";

// ★ usePrintCard Hook（print_log + QR 情報取得）
// ✅ modelId を正にした版（QuantityRowBase: modelId）
import { usePrintCard } from "../../../../product/src/presentation/hook/usePrintCard";

// ★ 分離した印刷カードコンポーネント
import PrintCard from "../../../../product/src/presentation/component/printCard";

// ✅ Presentation 正: ProductionQuantityRowVM（キーは modelId）
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";

export default function ProductionDetail() {
  const {
    // モード関連
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
  // - 正は modelId
  // - usePrintCard も modelId を要求する（QuantityRowBase: modelId）
  // ==========================
  const rowsForPrint = React.useMemo(() => {
    const safe: ProductionQuantityRowVM[] = Array.isArray(quantityRows)
      ? quantityRows
      : [];

    return safe.map((r, index) => ({
      modelId: String(r.modelId ?? "").trim() || String(index),
      quantity: r.quantity ?? 0,

      // 以降は usePrintCard が参照しうる情報（無害に付与）
      modelNumber: r.modelNumber,
      size: r.size,
      color: r.color,
      rgb: r.rgb ?? null,
    }));
  }, [quantityRows]);

  const { onPrint, printing } = usePrintCard({
    productionId: productionId ?? null,
    hasProduction: !!production,
    rows: rowsForPrint,
  });

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

    const ok = window.confirm(
      "印刷後は生産数を更新できません。\n印刷後に追加生産が必要になった場合は生産計画を新規作成してください。",
    );
    if (!ok) return;

    await onPrint();
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

              {isViewMode && <PrintCard printing={printing} onClick={handlePrint} />}
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

          <LogCard title="更新履歴" logs={[]} emptyText="更新履歴はまだありません。" />
        </div>
      </PageStyle>
    </>
  );
}
