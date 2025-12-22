// frontend/console/list/src/presentation/pages/listDetail.tsx
// ✅ style 要素中心（状態/処理は hook に寄せる）

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Input } from "../../../../shell/src/shared/ui/input";
import { Button } from "../../../../shell/src/shared/ui/button";

// ✅ PriceCard（list app 側のコンポーネント）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ AdminCard（担当者編集 + 作成/更新情報表示）
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

// ✅ hook（同一 app 内なので相対でOK）
import { useListDetail } from "../hook/useListDetail";

function ImageIcon() {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 24 24"
      fill="none"
      className="text-slate-400"
    >
      <path
        d="M21 19V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M8.5 10.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M21 16l-5.5-5.5a2 2 0 0 0-2.8 0L5 18"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      className="text-slate-500"
    >
      <path
        d="M12 5v14M5 12h14"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

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

  // ✅ 型を固定して noImplicitAny を回避
  const effectiveImageUrls: string[] = React.useMemo(() => {
    const arr = Array.isArray(vm.imageUrls) ? vm.imageUrls : [];
    return arr.map((u) => s(u)).filter(Boolean);
  }, [vm.imageUrls]);

  const hasImages = effectiveImageUrls.length > 0;

  const mainUrl = hasImages ? effectiveImageUrls[vm.mainImageIndex] : "";

  const thumbIndices: number[] = React.useMemo(() => {
    if (!hasImages) return [];
    return effectiveImageUrls
      .map((_: string, idx: number) => idx)
      .filter((idx: number) => idx !== vm.mainImageIndex);
  }, [hasImages, effectiveImageUrls, vm.mainImageIndex]);

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

        {/* ✅ 商品画像カード（複数枚アップロード UI に更新） */}
        <Card>
          <CardContent className="p-4 space-y-3">
            <div className="text-sm font-medium flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-slate-50 border border-slate-200">
                  <ImageIcon />
                </span>
                商品画像
              </div>

              {isEdit && (
                <div className="flex items-center gap-2">
                  <label className="cursor-pointer">
                    <input
                      type="file"
                      accept="image/*"
                      multiple
                      className="hidden"
                      onChange={(e) => vm.onAddImages(e.target.files)}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      className="h-8"
                      disabled={vm.saving}
                    >
                      画像を追加
                    </Button>
                  </label>

                  {effectiveImageUrls.length > 0 && (
                    <Button
                      type="button"
                      variant="ghost"
                      className="h-8"
                      onClick={() => {
                        // hook に「全削除」が無い場合は、存在する分だけ呼べるようにする
                        if (typeof anyVm.onClearImages === "function") {
                          anyVm.onClearImages();
                          return;
                        }
                        // fallback: 末尾から削除（indexがずれるのを避ける）
                        if (typeof vm.onRemoveImageAt === "function") {
                          for (let i = effectiveImageUrls.length - 1; i >= 0; i--) {
                            vm.onRemoveImageAt(i);
                          }
                          vm.setMainImageIndex(0);
                        }
                      }}
                      disabled={vm.saving}
                    >
                      クリア
                    </Button>
                  )}
                </div>
              )}
            </div>

            {/* empty state */}
            {!hasImages && (
              <div
                className={[
                  "rounded-xl border border-dashed border-slate-300 bg-slate-50/30 w-full aspect-[16/9]",
                  "flex flex-col items-center justify-center gap-3 select-none",
                  isEdit ? "cursor-pointer" : "",
                ].join(" ")}
                onClick={() => {
                  // edit時は「追加」ボタンがあるので、カードクリックは何もしない（誤タップ防止）
                }}
              >
                <div className="w-12 h-12 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                  <ImageIcon />
                </div>
                <div className="text-sm text-slate-700">画像は未設定です</div>
                <div className="text-xs text-[hsl(var(--muted-foreground))]">
                  {isEdit
                    ? "右上の「画像を追加」から複数画像を追加できます。"
                    : "画像を追加する場合は編集モードに切り替えてください。"}
                </div>
              </div>
            )}

            {/* filled state */}
            {hasImages && (
              <>
                {/* メイン（大） */}
                <div className="relative rounded-xl overflow-hidden border border-slate-200 bg-white">
                  <div className="w-full aspect-[16/9] bg-slate-50">
                    {mainUrl && (
                      <img
                        src={mainUrl}
                        alt="main"
                        className="w-full h-full object-cover"
                      />
                    )}
                  </div>

                  {/* remove main */}
                  {isEdit && (
                    <button
                      type="button"
                      className="absolute top-3 right-3 w-8 h-8 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                      onClick={() => vm.onRemoveImageAt(vm.mainImageIndex)}
                      aria-label="remove main image"
                      title="削除"
                      disabled={vm.saving}
                    >
                      <span className="text-slate-600 leading-none">×</span>
                    </button>
                  )}

                  {/* footer */}
                  <div className="px-3 py-2 border-t border-slate-200 flex items-center justify-between">
                    <div className="text-xs text-[hsl(var(--muted-foreground))]">
                      {effectiveImageUrls.length} 枚
                      {isEdit
                        ? "（サムネの×で削除できます）"
                        : "（サムネをクリックしてメイン切替できます）"}
                    </div>
                    {!isEdit && (
                      <div className="text-[11px] text-slate-400">
                        ※ 画像変更は編集モードで行えます
                      </div>
                    )}
                  </div>
                </div>

                {/* サブ（小） + 追加タイル（edit時のみ表示） */}
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  {thumbIndices.map((idx: number) => {
                    const url = effectiveImageUrls[idx] ?? "";
                    return (
                      <div
                        key={`${url}-${idx}`}
                        className="relative rounded-xl overflow-hidden border border-slate-200 bg-white cursor-pointer"
                        onClick={() => vm.setMainImageIndex(idx)}
                        role="button"
                        tabIndex={0}
                        title="クリックでメインに設定"
                      >
                        <div className="w-full aspect-square bg-slate-50">
                          {url && (
                            <img
                              src={url}
                              alt={`sub-${idx}`}
                              className="w-full h-full object-cover"
                            />
                          )}
                        </div>

                        {isEdit && (
                          <button
                            type="button"
                            className="absolute top-2 right-2 w-7 h-7 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                            onClick={(e) => {
                              e.stopPropagation();
                              vm.onRemoveImageAt(idx);
                            }}
                            aria-label="remove image"
                            title="削除"
                            disabled={vm.saving}
                          >
                            <span className="text-slate-600 leading-none">×</span>
                          </button>
                        )}
                      </div>
                    );
                  })}

                  {/* 追加タイル（edit時のみ） */}
                  {isEdit && (
                    <label
                      className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 cursor-pointer flex flex-col items-center justify-center gap-2 aspect-square"
                      title="画像を追加（複数可）"
                    >
                      <input
                        type="file"
                        accept="image/*"
                        multiple
                        className="hidden"
                        onChange={(e) => vm.onAddImages(e.target.files)}
                      />
                      <div className="w-10 h-10 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                        <PlusIcon />
                      </div>
                      <div className="text-xs text-slate-700">画像を追加</div>
                    </label>
                  )}
                </div>
              </>
            )}
          </CardContent>
        </Card>

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
