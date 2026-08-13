// frontend/console/shell/src/pages/productionDetail.tsx

import React from "react";
import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import ProductionQuantityCard from "../features/production/presentation/components/productionQuantityCard";
import { useProductionDetail } from "../features/production/presentation/hook/useProductionDetail";
import LogCard from "../features/log/presentation/LogCard";
import { usePrintCard } from "../features/print/presentation/hook/usePrintCard";
import PrintCard from "../features/print/presentation/component/printCard";

import "../styles/production.css";

export default function ProductionDetail() {
  const {
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,
    adminMode,
    canEdit,
    onBack,
    onSave,
    onDelete,
    deleting,
    productionId,
    production,
    loading,
    error,
    quantityRows,
    setQuantityRows,
  } = useProductionDetail();

  const assigneeDisplay =
    production?.assigneeName ||
    production?.assigneeId ||
    "担当者が設定されていません";

  const createdAtLabel = production?.createdAt
    ? new Date(production.createdAt).toLocaleDateString("ja-JP")
    : "-";

  const isPrinted = production?.printed === true;
  const productBlueprintCategoryCode =
    production?.productBlueprintCategory?.code ?? "";

  const { onPrint, printing } = usePrintCard({
    productionId: productionId ?? null,
    hasProduction: Boolean(production),
  });

  const handleSave = React.useCallback(() => {
    void onSave();
  }, [onSave]);

  const handleDelete = React.useCallback(async () => {
    if (isPrinted) {
      window.alert("印刷済みの生産は削除できません。");
      return;
    }

    const ok = window.confirm(
      "この生産情報を削除します。\nこの操作は取り消せません。",
    );

    if (!ok) return;

    await onDelete();
  }, [isPrinted, onDelete]);

  const handlePrint = React.useCallback(async () => {
    if (!productionId) {
      window.alert("productionId が取得できませんでした。");
      return;
    }

    if (isPrinted) {
      await onPrint();
      return;
    }

    const ok = window.confirm(
      "印刷後は生産数を更新できません。\n印刷後に追加生産が必要になった場合は生産計画を新規作成してください。",
    );

    if (!ok) return;

    await onPrint();
  }, [productionId, onPrint, isPrinted]);

  return (
    <PageStyle
      layout="grid-2"
      title="生産詳細"
      onBack={onBack}
      onEdit={isViewMode && canEdit ? switchToEdit : undefined}
      onDelete={isEditMode && canEdit && !deleting ? handleDelete : undefined}
      onCancel={isEditMode ? switchToView : undefined}
      onSave={isEditMode ? handleSave : undefined}
    >
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
            <ProductBlueprintCard
              mode="view"
              productName={production.productName}
              brandName={production.brandName}
              productBlueprintCategory={production.productBlueprintCategory}
            />

            <ProductionQuantityCard
              title="モデル別 生産数一覧"
              rows={quantityRows}
              productBlueprintCategory={productBlueprintCategoryCode}
              mode={isEditMode ? "edit" : "view"}
              onChangeRows={isEditMode ? setQuantityRows : undefined}
            />

            {isViewMode && (
              <PrintCard
                printing={printing}
                onClick={handlePrint}
                printed={isPrinted}
              />
            )}
          </>
        )}
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assigneeDisplay}
          assigneeCandidates={[]}
          loadingMembers={false}
          createdByName={production?.createdByName || "-"}
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
  );
}