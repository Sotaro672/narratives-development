// frontend/console/list/src/presentation/pages/listDetail.tsx
// ✅ style 要素中心（状態/処理は hook に寄せる）

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Input } from "../../../../shell/src/shared/ui/input";

// ✅ PriceCard（list app 側のコンポーネント）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ AdminCard（担当者編集 + 作成/更新情報表示）
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

// ✅ NEW: 商品画像カード（分離）
import ListImageCard from "../components/listImageCard";

// ✅ hook（同一 app 内なので相対でOK）
import { useListDetail } from "../hook/useListDetail";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function ListDetail() {
  const navigate = useNavigate();

  const vm = useListDetail();
  const anyVm = vm as any;

  const isEdit = vm.isEdit;

  const headerTitle =
    (isEdit ? s(vm.draftListingTitle) : s(vm.listingTitle)) || "出品詳細";

  // ✅ 戻るは -1 ではなく、一覧（listManagement.tsx）へ絶対遷移
  const onBackToListManagement = React.useCallback(() => {
    navigate("/list");
  }, [navigate]);

  const effectiveDecision = isEdit ? vm.draftDecision : vm.decisionNorm;

  const effectivePriceRows = isEdit ? vm.draftPriceRows : vm.priceRows;

  // ✅ AdminCard: 担当者選択の通知（hook 側に存在しうる関数を吸収）
  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      // 代表的な命名を吸収（存在するものだけ呼ぶ）
      if (typeof anyVm.setDraftAssigneeId === "function") {
        anyVm.setDraftAssigneeId(id);
      }
      if (typeof anyVm.onSelectAssignee === "function") {
        anyVm.onSelectAssignee(id);
      }
      if (typeof anyVm.onChangeAssignee === "function") {
        anyVm.onChangeAssignee(id);
      }
    },
    [anyVm],
  );

  return (
    <PageStyle
      layout="grid-2"
      title={headerTitle}
      onBack={onBackToListManagement}
      // ✅ 編集開始は PageHeader の編集ボタンだけ
      onEdit={!isEdit ? vm.onEdit : undefined}
      onCancel={isEdit ? vm.onCancel : undefined}
      onSave={isEdit ? vm.onSave : undefined}
      onCreate={undefined}
    >
      {/* =========================
          左カラム
          - 商品画像（edit 対応: UI-only）
          - タイトル（edit 対応）
          - 説明（edit 対応）
          - 価格（PriceCard: edit 対応）
          ========================= */}
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

        {/* ✅ 商品画像カード（分離） */}
        <ListImageCard
          isEdit={isEdit}
          saving={vm.saving}
          imageUrls={(vm as any).imageUrls ?? []}
          mainImageIndex={(vm as any).mainImageIndex ?? 0}
          setMainImageIndex={(idx) => vm.setMainImageIndex(idx)}
          onAddImages={(files) => vm.onAddImages(files)}
          onRemoveImageAt={(idx) => vm.onRemoveImageAt(idx)}
          onClearImages={typeof anyVm.onClearImages === "function" ? anyVm.onClearImages : undefined}
          anyVm={anyVm}
        />

        {/* ✅ タイトル */}
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

        {/* ✅ 説明 */}
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

        {/* ✅ 価格 */}
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

      {/* =========================
          右カラム
          - 担当者（AdminCard: edit 対応）
          - 選択商品
          - 選択トークン
          - 出品｜保留（edit 対応）
          ========================= */}
      <div className="space-y-4">
        {/* ✅ 担当者（edit 時に編集可能） */}
        <AdminCard
          title="担当者"
          mode={isEdit ? "edit" : "view"}
          assigneeName={vm.assigneeName}
          onSelectAssignee={isEdit ? handleSelectAssignee : undefined}
          onEditAssignee={isEdit ? anyVm.onEditAssignee : undefined}
          onClickAssignee={isEdit ? anyVm.onClickAssignee : undefined}
          createdByName={vm.createdByName}
          createdAt={vm.createdAt}
          updatedByName={vm.updatedByName}
          updatedAt={vm.updatedAt}
        />

        {/* 選択商品 */}
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

        {/* 選択トークン */}
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

        {/* ✅ 出品｜保留（編集開始は PageHeader のみ） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            {/* view */}
            {!isEdit && (
              <>
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
              </>
            )}

            {/* edit */}
            {isEdit && (
              <>
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
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
