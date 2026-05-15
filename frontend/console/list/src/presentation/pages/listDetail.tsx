// frontend/console/list/src/presentation/pages/listDetail.tsx
// style 要素中心（状態/処理は hook に寄せる）

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Input } from "../../../../shell/src/shared/ui/input";

import PriceCard from "../../../../list/src/presentation/components/priceCard";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ListImageCard from "../components/listImageCard";
import { useListDetail } from "../hook/useListDetail";

export default function ListDetail() {
  const navigate = useNavigate();

  const vm = useListDetail();

  const isEdit = vm.isEdit;

  const headerTitle = isEdit ? vm.draftListingTitle || "出品詳細" : vm.listingTitle || "出品詳細";

  const onBackToListManagement = React.useCallback(() => {
    navigate("/list");
  }, [navigate]);

  const effectiveDecision = isEdit ? vm.draftDecision : vm.decisionNorm;
  const effectivePriceRows = isEdit ? vm.draftPriceRows : vm.priceRows;

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      vm.setDraftAssigneeId?.(id);
      vm.onSelectAssignee?.(id);
      vm.onChangeAssignee?.(id);
    },
    [vm],
  );

  return (
    <PageStyle
      layout="grid-2"
      title={headerTitle}
      onBack={onBackToListManagement}
      onEdit={!isEdit ? vm.onEdit : undefined}
      onCancel={isEdit ? vm.onCancel : undefined}
      onSave={isEdit ? vm.onSave : undefined}
      onCreate={undefined}
    >
      <div className="space-y-4">
        {vm.loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        )}
        {vm.error && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {vm.error}
          </div>
        )}

        {isEdit && vm.saveError && (
          <div className="text-sm text-red-600">
            保存に失敗しました: {vm.saveError}
          </div>
        )}
        {isEdit && vm.saving && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            保存中...
          </div>
        )}

        <ListImageCard
          isEdit={isEdit}
          saving={vm.saving}
          imageUrls={Array.isArray(vm.imageUrls) ? vm.imageUrls : []}
          mainImageIndex={vm.mainImageIndex}
          setMainImageIndex={vm.setMainImageIndex}
          onAddImages={(files) => vm.onAddImages?.(files)}
          onRemoveImageAt={(idx) => vm.onRemoveImageAt?.(idx)}
          onClearImages={vm.onClearImages}
        />

        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">タイトル</div>

            {!isEdit && (
              <div className="text-sm text-slate-800 break-words">
                {vm.listingTitle || "未設定"}
              </div>
            )}

            {isEdit && (
              <Input
                value={vm.draftListingTitle}
                placeholder="タイトルを入力"
                onChange={(e) => vm.setDraftListingTitle(e.target.value)}
                disabled={vm.saving}
              />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">説明</div>

            {!isEdit && (
              <div className="text-sm text-slate-800 whitespace-pre-wrap break-words">
                {vm.description || "未設定"}
              </div>
            )}

            {isEdit && (
              <textarea
                value={vm.draftDescription}
                placeholder="説明を入力"
                onChange={(e) => vm.setDraftDescription(e.target.value)}
                className="w-full min-h-[120px] rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none"
                disabled={vm.saving}
              />
            )}
          </CardContent>
        </Card>

        <PriceCard
          title="価格"
          rows={effectivePriceRows as any}
          mode={isEdit ? "edit" : "view"}
          currencySymbol="¥"
          onChangePrice={isEdit ? vm.onChangePrice : undefined}
        />

        {Array.isArray(effectivePriceRows) && effectivePriceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格情報がありません。
          </div>
        )}
      </div>

      <div className="space-y-4">
        <AdminCard
          title="担当者"
          mode={isEdit ? "edit" : "view"}
          assigneeName={vm.assigneeName}
          onSelectAssignee={isEdit ? handleSelectAssignee : undefined}
          onEditAssignee={isEdit ? vm.onEditAssignee : undefined}
          onClickAssignee={isEdit ? vm.onClickAssignee : undefined}
          createdByName={vm.createdByName}
          createdAt={vm.createdAt}
          updatedByName={vm.updatedByName}
          updatedAt={vm.updatedAt}
        />

        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択商品</div>
            <div className="text-sm text-slate-800 break-all">
              {vm.productBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {vm.productName || "未選択"}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択トークン</div>
            <div className="text-sm text-slate-800 break-all">
              {vm.tokenBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {vm.tokenName || "未選択"}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            {!isEdit && (
              <div className="flex gap-2">
                <div
                  className={[
                    "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                    effectiveDecision === "listing"
                      ? "bg-slate-900 text-white border-slate-900"
                      : "bg-white text-slate-700 border-slate-200",
                  ].join(" ")}
                >
                  出品
                </div>

                <div
                  className={[
                    "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                    effectiveDecision === "holding"
                      ? "bg-slate-900 text-white border-slate-900"
                      : "bg-white text-slate-700 border-slate-200",
                  ].join(" ")}
                >
                  保留
                </div>
              </div>
            )}

            {isEdit && (
              <div className="flex gap-2">
                <button
                  type="button"
                  className={[
                    "flex-1 h-9 rounded-md border text-sm flex items-center justify-center transition",
                    vm.draftDecision === "listing"
                      ? "bg-slate-900 text-white border-slate-900"
                      : "bg-white text-slate-700 border-slate-200",
                    vm.saving ? "opacity-60 cursor-not-allowed" : "cursor-pointer",
                  ].join(" ")}
                  onClick={() => vm.onToggleDecision("listing")}
                  disabled={vm.saving}
                >
                  出品
                </button>

                <button
                  type="button"
                  className={[
                    "flex-1 h-9 rounded-md border text-sm flex items-center justify-center transition",
                    vm.draftDecision === "holding"
                      ? "bg-slate-900 text-white border-slate-900"
                      : "bg-white text-slate-700 border-slate-200",
                    vm.saving ? "opacity-60 cursor-not-allowed" : "cursor-pointer",
                  ].join(" ")}
                  onClick={() => vm.onToggleDecision("holding")}
                  disabled={vm.saving}
                >
                  保留
                </button>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}