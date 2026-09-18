//frontend\console\shell\src\pages\listCreate.tsx
import * as React from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
} from "../shared/ui/card";
import {
  Select,
} from "../shared/ui/select";
import Textarea from "../shared/ui/textarea";

import PriceCard from "../features/list/presentation/components/priceCard";
import ListImageCard from "../features/list/presentation/components/listImageCard";
import ListProgressModal from "../features/list/presentation/components/listProgressModal";
import ListStatusHeaderActions from "../features/list/presentation/components/ListStatusHeaderActions";
import InventoryListCard from "../features/inventory/presentation/components/InventoryListCard";

import {
  useListCreate,
} from "../features/inventory/presentation/hook/useListCreate";

import "../styles/list.css";

export default function InventoryListCreate() {
  const {
    onBack,
    onCreate,
    saving,
    progress,
    progressOpen,
    onCloseProgress,
    loadingDTO,
    dtoError,
    productName,
    tokenName,
    priceRows,
    onChangePrice,
    listingTitle,
    setListingTitle,
    description,
    setDescription,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    onAddImages,
    onRemoveImageAt,
    onClearImages,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
    listItems,
    listLoading,
    listError,
    onOpenList,
    status,
    setStatus,
  } = useListCreate();

  const missingModelIdCount = React.useMemo(
    () => priceRows.filter((row) => !row.modelId).length,
    [priceRows],
  );

  const headerTitle = React.useMemo(() => {
    const safeProductName = productName.trim();
    const safeTokenName = tokenName.trim();

    if (safeProductName && safeTokenName) {
      return `出品作成：${safeProductName} / ${safeTokenName}`;
    }

    if (safeProductName) {
      return `出品作成：${safeProductName}`;
    }

    if (safeTokenName) {
      return `出品作成：${safeTokenName}`;
    }

    return "出品作成";
  }, [productName, tokenName]);

  const assigneeOptions = React.useMemo(
    () =>
      assigneeCandidates.map((candidate) => ({
        value: candidate.name,
        label: candidate.name,
      })),
    [assigneeCandidates],
  );

  const handleChangeAssignee = React.useCallback(
    (selectedName: string) => {
      const matched = assigneeCandidates.find(
        (candidate) => candidate.name === selectedName,
      );

      if (!matched) return;

      handleSelectAssignee(matched.id);
    },
    [assigneeCandidates, handleSelectAssignee],
  );

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={headerTitle}
        onBack={onBack}
        leadingActions={
          <ListStatusHeaderActions
            status={status}
            onChange={setStatus}
          />
        }
        onCreate={onCreate}
        isSaving={saving}
      >
        {/* 左カラム */}
        <div className="list-create__column">
          <ListImageCard
            isEdit={true}
            saving={saving}
            imageUrls={imagePreviewUrls}
            mainImageIndex={mainImageIndex}
            setMainImageIndex={setMainImageIndex}
            onAddImages={onAddImages}
            onRemoveImageAt={onRemoveImageAt}
            onClearImages={onClearImages}
          />

          <Card>
            <CardContent className="list-create__card-content list-create__card-content--stack">
              <div className="list-create__label">
                タイトル
              </div>

              <input
                value={listingTitle}
                onChange={(event) =>
                  setListingTitle(
                    event.target.value,
                  )
                }
                placeholder="例: Narratives シャツ1（赤 / S・M）"
                className="list-create__title-input"
                disabled={saving}
              />
            </CardContent>
          </Card>

          <Card>
            <CardContent className="list-create__card-content list-create__card-content--stack">
              <div className="list-create__label">
                説明
              </div>

              <Textarea
                value={description}
                onChange={(event) =>
                  setDescription(
                    event.target.value,
                  )
                }
                placeholder="商品の状態、サイズ感、注意事項などを入力してください。"
                rows={5}
                disabled={saving}
              />
            </CardContent>
          </Card>

          <PriceCard
            title="価格"
            rows={priceRows}
            mode="edit"
            currencySymbol="¥"
            onChangePrice={onChangePrice}
          />

          {priceRows.length === 0 && (
            <div className="list-create__message list-create__message--muted">
              価格行データは未取得です。
            </div>
          )}

          {missingModelIdCount > 0 && (
            <div className="list-create__message list-create__message--error">
              modelId が未設定の価格行があります: {missingModelIdCount} 件
            </div>
          )}
        </div>

        {/* 右カラム */}
        <div className="list-create__column">
          {loadingDTO && (
            <div className="list-create__status list-create__status--muted">
              読み込み中...
            </div>
          )}

          {dtoError && (
            <div className="list-create__status list-create__status--error">
              読み込みに失敗しました: {dtoError}
            </div>
          )}

          <Card>
            <CardContent className="list-create__card-content">
              <div className="list-create__label list-create__label--spaced">
                担当者
              </div>

              {loadingMembers ? (
                <div className="list-create__assignee-message">
                  担当者を読み込み中です…
                </div>
              ) : assigneeOptions.length > 0 ? (
                <Select
                  options={assigneeOptions}
                  value={assigneeName}
                  onChange={handleChangeAssignee}
                  placeholder="担当者を選択してください"
                  disabled={saving}
                />
              ) : (
                <div className="list-create__assignee-message">
                  担当者候補がありません。
                </div>
              )}
            </CardContent>
          </Card>

          <InventoryListCard
            items={listItems}
            loading={listLoading}
            error={listError}
            onOpenList={onOpenList}
          />
        </div>
      </PageStyle>

      <ListProgressModal
        open={progressOpen}
        progress={progress}
        onClose={
          progress.isBlockingNavigation
            ? undefined
            : onCloseProgress
        }
      />
    </>
  );
}