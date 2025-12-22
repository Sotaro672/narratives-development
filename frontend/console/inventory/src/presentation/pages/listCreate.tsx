// frontend/console/inventory/src/presentation/pages/listCreate.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

// ✅ PriceCard（list app 側に作ったコンポーネントを流用）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ NEW: 既存の商品画像カードを list app から流用
import ListImageCard from "../../../../list/src/presentation/components/listImageCard";

// ✅ logic は hook 側へ（UI state も hook に寄せた）
import { useListCreate } from "../hook/useListCreate";

// local trim helper (UI-only)
function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function InventoryListCreate() {
  const {
    title,
    onBack,
    onCreate,

    // dto state
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    // price
    priceRows,
    onChangePrice,

    // listing (moved to hook)
    listingTitle,
    setListingTitle,
    description,
    setDescription,

    // images (moved to hook)
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    onSelectImages,
    removeImageAt,
    clearImages,

    // assignee
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,

    // decision
    decision,
    setDecision,
  } = useListCreate();

  // ✅ modelId 付与チェック（UIで検知できるように）
  const missingModelIdCount = React.useMemo(() => {
    return (priceRows ?? []).filter((r: any) => !s(r?.modelId)).length;
  }, [priceRows]);

  // ✅ ListImageCard が要求する onAddImages(FileList|null) へアダプト
  // - hook 側の onSelectImages(ChangeEvent<HTMLInputElement>) を流用するため、最小限の疑似イベントを渡す
  const onAddImages = React.useCallback(
    (files: FileList | null) => {
      if (!files || files.length === 0) return;

      const fakeEvent = {
        target: { files },
        currentTarget: { value: "" },
      } as any;

      onSelectImages(fakeEvent);
    },
    [onSelectImages],
  );

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack} onCreate={onCreate}>
      {/* =========================
          左カラム
          - 商品画像（ListImageCard を流用）
          - タイトル
          - 説明
          - PriceCard
          ========================= */}
      <div className="space-y-4">
        {/* ✅ 商品画像カード（list app の既存コンポーネントを流用） */}
        <ListImageCard
          isEdit={true}
          saving={false}
          imageUrls={Array.isArray(imagePreviewUrls) ? imagePreviewUrls : []}
          mainImageIndex={Number.isFinite(Number(mainImageIndex)) ? mainImageIndex : 0}
          setMainImageIndex={(idx) => setMainImageIndex(idx)}
          onAddImages={onAddImages}
          onRemoveImageAt={(idx) => removeImageAt(idx)}
          onClearImages={() => clearImages()}
        />

        {/* ✅ タイトル入力カード（商品画像の下に配置） */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">タイトル</div>
            <input
              value={listingTitle}
              onChange={(e) => setListingTitle(e.target.value)}
              placeholder="例: Solid State シャツ1（赤 / S・M）"
              className="w-full h-10 px-3 rounded-md border border-slate-200 bg-white text-sm outline-none focus:ring-2 focus:ring-slate-200"
            />
          </CardContent>
        </Card>

        {/* ✅ 説明入力カード */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">説明</div>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="商品の状態、サイズ感、注意事項などを入力してください。"
              rows={5}
              className="w-full px-3 py-2 rounded-md border border-slate-200 bg-white text-sm outline-none focus:ring-2 focus:ring-slate-200"
            />
          </CardContent>
        </Card>

        {/* ✅ PriceCard */}
        <PriceCard
          title="価格"
          // ✅ rows は UI 用に size/color/stock を含んでOK（POSTは service 側で modelId+price のみに射影する）
          rows={priceRows as any}
          mode="edit"
          currencySymbol="¥"
          onChangePrice={(idx: number, price: number | null) => onChangePrice(idx, price)}
        />

        {priceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格行データは未取得です（DTO/別APIから rows を供給する実装が必要です）。
          </div>
        )}

        {/* ✅ modelId が欠けていると POST で落ちるので、UI 上でも見えるように */}
        {priceRows.length > 0 && missingModelIdCount > 0 && (
          <div className="text-xs text-red-600">
            価格行に modelId が付与されていない行があります（{missingModelIdCount} 件）。
            DTO の priceRows に modelId が含まれているか確認してください。
          </div>
        )}
      </div>

      {/* =========================
          右カラム
          ========================= */}
      <div className="space-y-4">
        {/* DTO 読み込み状態（style elements only） */}
        {loadingDTO && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">読み込み中...</div>
        )}
        {dtoError && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {dtoError}
          </div>
        )}

        {/* ✅ 担当者 */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">担当者</div>

            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="w-full justify-between"
                >
                  <span>{assigneeName || "未設定"}</span>
                  <span className="text-[11px] text-slate-400" />
                </Button>
              </PopoverTrigger>

              <PopoverContent className="p-2 space-y-1">
                {loadingMembers && (
                  <p className="text-xs text-slate-400">担当者を読み込み中です…</p>
                )}

                {!loadingMembers && assigneeCandidates.length > 0 && (
                  <div className="space-y-1">
                    {assigneeCandidates.map((c) => (
                      <button
                        key={c.id}
                        type="button"
                        className="block w-full text-left px-2 py-1 rounded hover:bg-slate-100 text-sm"
                        onClick={() => handleSelectAssignee(c.id)}
                      >
                        {c.name}
                      </button>
                    ))}
                  </div>
                )}

                {!loadingMembers && assigneeCandidates.length === 0 && (
                  <p className="text-xs text-slate-400">担当者候補がありません。</p>
                )}
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        {/* ✅ 選択商品カード */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択商品</div>
            <div className="text-sm text-slate-800 break-all">
              {productBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {productName || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 選択トークンカード */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択トークン</div>
            <div className="text-sm text-slate-800 break-all">
              {tokenBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {tokenName || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 出品｜保留 */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            <div className="flex gap-2">
              <Button
                type="button"
                variant={decision === "list" ? "default" : "outline"}
                size="sm"
                className="flex-1"
                onClick={() => setDecision("list")}
              >
                出品
              </Button>

              <Button
                type="button"
                variant={decision === "hold" ? "default" : "outline"}
                size="sm"
                className="flex-1"
                onClick={() => setDecision("hold")}
              >
                保留
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
