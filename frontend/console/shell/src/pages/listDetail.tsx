// frontend/console/shell/src/pages/listDetail.tsx  
  
import * as React from "react";  
import { useNavigate } from "react-router-dom";  
import PageStyle from "../layout/PageStyle/PageStyle";  
  
import { Card, CardContent } from "../shared/ui/card";  
import { Input } from "../shared/ui/input";  
  
import PriceCard from "../features/list/presentation/components/priceCard";  
import AdminCard from "../features/admin/presentation/components/AdminCard";  
import ListImageCard from "../features/list/presentation/components/listImageCard";  
import ListStatusHeaderActions from "../features/list/presentation/components/ListStatusHeaderActions";  
import ListTargetProductCard from "../features/list/presentation/components/ListTargetProductCard";  
import { useListDetail } from "../features/list/presentation/hook/useListDetail";  
  
export default function ListDetail() {  
  const navigate = useNavigate();  
  const vm = useListDetail();  
  const isEdit = vm.isEdit;  
  
  const headerTitle =  
    vm.readableId || "出品詳細";  
  
  const onBackToListManagement = React.useCallback(() => {  
    navigate("/list");  
  }, [navigate]);  
  
  const effectivePriceRows = isEdit ? vm.draftPriceRows : vm.priceRows;  
  const effectiveAssigneeId = isEdit ? vm.draftAssigneeId : vm.assigneeId;  
  const effectiveAssigneeName = isEdit ? vm.draftAssigneeName : vm.assigneeName;  
  
  const effectiveStatus =  
    vm.status === "listing"  
      ? "listing"  
      : "suspended";  
  
  return (  
    <PageStyle  
      layout="grid-2"  
      title={headerTitle}  
      onBack={onBackToListManagement}  
      leadingActions={  
        !isEdit ? (  
          <ListStatusHeaderActions  
            status={effectiveStatus}  
            disabled  
          />  
        ) : undefined  
      }  
      onEdit={!isEdit && !vm.deleting ? vm.onEdit : undefined}  
      onDelete={isEdit && !vm.saving && !vm.deleting ? vm.onDelete : undefined}  
      onCancel={isEdit && !vm.deleting ? vm.onCancel : undefined}  
      onSave={isEdit && !vm.deleting ? vm.onSave : undefined}  
      isSaving={vm.saving}  
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
  
        {isEdit && vm.deleteError && (  
          <div className="text-sm text-red-600">  
            削除に失敗しました: {vm.deleteError}  
          </div>  
        )}  
  
        {isEdit && vm.deleting && (  
          <div className="text-xs text-[hsl(var(--muted-foreground))]">  
            削除中...  
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
          onAddImages={(files) => vm.onAddImages(files)}  
          onRemoveImageAt={(idx) => vm.onRemoveImageAt(idx)}  
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
                disabled={vm.saving || vm.deleting}  
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
                disabled={vm.saving || vm.deleting}  
              />  
            )}  
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
          <div className="text-xs text-[hsl(var(--muted-foreground))]">  
            価格情報がありません。  
          </div>  
        )}  
      </div>  
  
      <div className="space-y-4">  
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
      </div>  
    </PageStyle>  
  );  
}