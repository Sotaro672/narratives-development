// frontend/console/shell/src/pages/productionDetail.tsx

import React from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import ProductBlueprintCard from "../features/productBlueprint/presentation/components/productBlueprintForm";
import { toProductBlueprintCategoryPathKey } from "../features/productBlueprint/domain/productBlueprintCategory";
import ProductionQuantityCard from "../features/production/presentation/components/productionQuantityCard";
import { useProductionDetail } from "../features/production/presentation/hook/useProductionDetail";
import { usePrintCard } from "../features/print/presentation/hook/usePrintCard";
import { ErrorMessage } from "../shared/ui/error";

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
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    creator,
    createdAt,
    updater,
    updatedAt,
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
    reloadProduction,
  } = useProductionDetail();

  const isPrinted = production?.printed === true;

  const productBlueprintCategoryCode =
    production?.productBlueprintCategoryPath
      ? toProductBlueprintCategoryPathKey(
          production.productBlueprintCategoryPath,
        )
      : "";

  const {
    onPrint,
    onQrOutput,
    onCsvOutput,
    printing,
    qrOutputting,
    csvOutputting,
  } = usePrintCard({
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

    const ok = window.confirm(
      "印刷後は生産数を更新できません。\n印刷後に追加生産が必要になった場合は生産計画を新規作成してください。",
    );

    if (!ok) return;

    const logs = await onPrint();

    if (logs.length === 0) return;

    await reloadProduction();
  }, [productionId, onPrint, reloadProduction]);

  const handleQrOutput = React.useCallback(async () => {
    if (!productionId) {
      window.alert("productionId が取得できませんでした。");
      return;
    }

    await onQrOutput();
  }, [productionId, onQrOutput]);

  const handleCsvOutput = React.useCallback(async () => {
    if (!productionId) {
      window.alert("productionId が取得できませんでした。");
      return;
    }

    await onCsvOutput();
  }, [productionId, onCsvOutput]);

  return (
    <PageStyle
      layout="grid-2"
      title="生産詳細"
      onBack={onBack}
      statusButtonLabel="印刷"
      statusButtonBusyLabel="発行中..."
      onStatusButtonClick={
        isViewMode && production && !isPrinted
          ? handlePrint
          : undefined
      }
      isStatusButtonLoading={printing}
      statusButtonDisabled={loading || !productionId}
      onQrOutput={
        isViewMode && isPrinted
          ? handleQrOutput
          : undefined
      }
      isQrOutputting={qrOutputting}
      qrOutputDisabled={loading || !productionId}
      onCsvOutput={
        isViewMode && isPrinted
          ? handleCsvOutput
          : undefined
      }
      isCsvOutputting={csvOutputting}
      csvOutputDisabled={loading || !productionId}
      onEdit={
        isViewMode && canEdit
          ? switchToEdit
          : undefined
      }
      onDelete={
        isEditMode && canEdit && !deleting
          ? handleDelete
          : undefined
      }
      onCancel={
        isEditMode
          ? switchToView
          : undefined
      }
      onSave={
        isEditMode
          ? handleSave
          : undefined
      }
    >
      <div className="page-column">
        {loading && (
          <div className="production-detail__state">
            生産情報を読み込み中です…
          </div>
        )}

        {!loading && error && (
          <ErrorMessage>
            {error}
          </ErrorMessage>
        )}

        {!loading && !error && !production && (
          <div className="production-detail__state">
            対象の生産情報が見つかりません。
          </div>
        )}

        {!loading && !error && production && (
          <>
            <ProductBlueprintCard
              mode="view"
              productName={production.productName}
              brandName={production.brandName}
              productBlueprintCategoryPath={
                production.productBlueprintCategoryPath
              }
            />

            <ProductionQuantityCard
              title="モデル別 生産数一覧"
              rows={quantityRows}
              productBlueprintCategory={productBlueprintCategoryCode}
              mode={isEditMode ? "edit" : "view"}
              onChangeRows={
                isEditMode
                  ? setQuantityRows
                  : undefined
              }
            />
          </>
        )}
      </div>

      <div className="page-column">
        <AdminCard
          title="管理情報"
          assigneeId={assigneeId || undefined}
          assigneeName={assigneeName}
          assigneeCandidates={assigneeCandidates}
          loadingMembers={loadingMembers}
          createdByName={creator}
          createdAt={createdAt}
          updatedByName={updater}
          updatedAt={updatedAt}
          mode={adminMode}
          onSelectAssignee={onSelectAssignee}
          onEditAssignee={onEditAssignee}
          onClickAssignee={onClickAssignee}
        />
      </div>
    </PageStyle>
  );
}