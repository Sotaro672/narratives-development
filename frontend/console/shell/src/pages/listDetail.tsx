// frontend/console/shell/src/pages/listDetail.tsx

import PageStyle from "../layout/PageStyle/PageStyle";

import { Card, CardContent } from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import { Input } from "../shared/ui/input";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";
import Textarea from "../shared/ui/textarea";

import PriceCard from "../features/list/presentation/components/priceCard";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import ListImageCard from "../features/list/presentation/components/listImageCard";
import ListProgressModal from "../features/list/presentation/components/listProgressModal";
import ListStatusHeaderActions from "../features/list/presentation/components/ListStatusHeaderActions";
import ListTargetProductCard from "../features/list/presentation/components/ListTargetProductCard";
import ListSalesSummaryCard from "../features/list/presentation/components/ListSalesSummaryCard";
import { useListDetail } from "../features/list/presentation/hook/useListDetail";

import "../styles/list.css";

export default function ListDetail() {
  const vm = useListDetail();
  const isEdit = vm.isEdit;

  const headerTitle = vm.readableId || "出品詳細";

  const effectivePriceRows = isEdit
    ? vm.draftPriceRows
    : vm.priceRows;

  const effectiveAssigneeId = isEdit
    ? vm.draftAssigneeId
    : vm.assigneeId;

  const effectiveAssigneeName = isEdit
    ? vm.draftAssigneeName
    : vm.assigneeName;

  const effectiveStatus =
    vm.status === "listing"
      ? "listing"
      : "suspended";

  const statusActionDisabled =
    vm.loading ||
    vm.saving ||
    vm.deleting ||
    vm.progress.isBlockingNavigation ||
    !vm.status;

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={headerTitle}
        onBack={vm.onBack}
        leadingActions={
          !isEdit ? (
            <ListStatusHeaderActions
              status={effectiveStatus}
              onChange={vm.onChangeStatus}
              disabled={statusActionDisabled}
            />
          ) : undefined
        }
        onEdit={
          !isEdit && !vm.deleting
            ? vm.onEdit
            : undefined
        }
        onDelete={
          isEdit &&
          !vm.saving &&
          !vm.deleting
            ? vm.onDelete
            : undefined
        }
        onCancel={
          isEdit && !vm.deleting
            ? vm.onCancel
            : undefined
        }
        onSave={
          isEdit && !vm.deleting
            ? vm.onSave
            : undefined
        }
        isSaving={vm.saving}
        onCreate={undefined}
      >
        <div className="page-column">
          {vm.loading && (
            <Text as="div" size="sm" tone="muted">
              読み込み中...
            </Text>
          )}

          {vm.error && (
            <ErrorMessage>
              読み込みに失敗しました: {vm.error}
            </ErrorMessage>
          )}

          {isEdit && vm.deleteError && (
            <ErrorMessage>
              削除に失敗しました: {vm.deleteError}
            </ErrorMessage>
          )}

          {isEdit && vm.deleting && (
            <Text as="div" size="xs" tone="muted">
              削除中...
            </Text>
          )}

          {vm.saveError && (
            <ErrorMessage>
              保存に失敗しました: {vm.saveError}
            </ErrorMessage>
          )}

          <ListImageCard
            isEdit={isEdit}
            saving={vm.saving}
            imageUrls={Array.isArray(vm.imageUrls) ? vm.imageUrls : []}
            mainImageIndex={vm.mainImageIndex}
            setMainImageIndex={vm.setMainImageIndex}
            onAddImages={(files) => vm.onAddImages(files)}
            onRemoveImageAt={(idx) => vm.onRemoveImageAt(idx)}
            onClearImages={vm.onClearImages}
          />

          <Card>
            <CardContent className="list-detail__card-content">
              <Stack gap="sm">
                <Text as="div" size="sm" weight="medium">
                  タイトル
                </Text>

                {!isEdit && (
                  <Text as="div" size="sm" wrap="anywhere">
                    {vm.listingTitle || "未設定"}
                  </Text>
                )}

                {isEdit && (
                  <Input
                    value={vm.draftListingTitle}
                    placeholder="タイトルを入力"
                    onChange={(e) => vm.setDraftListingTitle(e.target.value)}
                    disabled={vm.saving || vm.deleting}
                  />
                )}
              </Stack>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="list-detail__card-content">
              <Stack gap="sm">
                <Text as="div" size="sm" weight="medium">
                  説明
                </Text>

                {!isEdit && (
                  <Text
                    as="div"
                    size="sm"
                    wrap="pre-wrap"
                    className="text--wrap-anywhere"
                  >
                    {vm.description || "未設定"}
                  </Text>
                )}

                {isEdit && (
                  <Textarea
                    value={vm.draftDescription}
                    placeholder="説明を入力"
                    onChange={(e) => vm.setDraftDescription(e.target.value)}
                    className="list-detail__description-input"
                    disabled={vm.saving || vm.deleting}
                  />
                )}
              </Stack>
            </CardContent>
          </Card>

          <PriceCard
            title="価格"
            rows={effectivePriceRows}
            mode={isEdit ? "edit" : "view"}
            currencySymbol="¥"
            onChangePrice={isEdit ? vm.onChangePrice : undefined}
          />

          {Array.isArray(effectivePriceRows) && effectivePriceRows.length === 0 && (
            <Text as="div" size="xs" tone="muted">
              価格情報がありません。
            </Text>
          )}
        </div>

        <div className="page-column">
          <AdminCard
            title="担当者"
            mode={isEdit ? "edit" : "view"}
            assigneeId={effectiveAssigneeId || undefined}
            assigneeName={effectiveAssigneeName}
            assigneeCandidates={vm.assigneeCandidates}
            loadingMembers={vm.loadingMembers}
            onSelectAssignee={isEdit ? vm.onSelectAssignee : undefined}
            createdByName={vm.createdByName}
            createdAt={vm.createdAt}
            updatedByName={vm.updatedByName}
            updatedAt={vm.updatedAt}
          />

          <ListTargetProductCard
            productName={vm.productName}
            tokenName={vm.tokenName}
          />

          <ListSalesSummaryCard
            totalOrderCount={vm.totalOrderCount}
            totalSalesAmount={vm.totalSalesAmount}
          />
        </div>
      </PageStyle>

      <ListProgressModal
        open={vm.progressOpen}
        progress={vm.progress}
        onClose={
          vm.progress.isBlockingNavigation
            ? undefined
            : vm.onCloseProgress
        }
      />
    </>
  );
}