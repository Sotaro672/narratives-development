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

// ✅ logic は hook 側へ（UI state も hook に寄せた）
import { useListCreate } from "../hook/useListCreate";

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
    imageInputRef,
    openImagePicker,
    onSelectImages,
    onDropImages,
    onDragOverImages,
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

  const hasImages = images.length > 0;
  const mainUrl = hasImages ? imagePreviewUrls[mainImageIndex] : "";

  // thumbnails = all except main (order preserved)
  const thumbIndices = React.useMemo(() => {
    if (!hasImages) return [];
    return images
      .map((_, idx) => idx)
      .filter((idx) => idx !== mainImageIndex);
  }, [hasImages, images, mainImageIndex]);

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onCreate={onCreate}
    >
      {/* =========================
          左カラム
          - 商品画像（メイン大 + サブ小 + 追加タイル）
          - タイトル
          - 説明
          - PriceCard
          ========================= */}
      <div className="space-y-4">
        {/* ✅ 商品画像カード（期待値: メイン大 / 2枚目以降小 + 追加） */}
        <Card>
          <CardContent className="p-4 space-y-3">
            <div className="text-sm font-medium flex items-center gap-2">
              <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-slate-50 border border-slate-200">
                <ImageIcon />
              </span>
              商品画像
            </div>

            {/* hidden input */}
            <input
              ref={imageInputRef}
              type="file"
              accept="image/*"
              multiple
              className="hidden"
              onChange={onSelectImages}
            />

            {/* empty state (2枚目画像のイメージ) */}
            {!hasImages && (
              <div
                className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 w-full aspect-[16/9] flex flex-col items-center justify-center gap-3 cursor-pointer select-none"
                onClick={openImagePicker}
                onDrop={onDropImages}
                onDragOver={onDragOverImages}
                role="button"
                tabIndex={0}
              >
                <div className="w-12 h-12 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                  <ImageIcon />
                </div>
                <div className="text-sm text-slate-700">画像をドロップ</div>
                <div className="text-xs text-[hsl(var(--muted-foreground))]">
                  またはクリックして選択
                </div>
              </div>
            )}

            {/* filled state (1枚目画像のイメージ) */}
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
                  <button
                    type="button"
                    className="absolute top-3 right-3 w-8 h-8 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                    onClick={() => removeImageAt(mainImageIndex)}
                    aria-label="remove main image"
                    title="削除"
                  >
                    <span className="text-slate-600 leading-none">×</span>
                  </button>

                  {/* footer */}
                  <div className="px-3 py-2 border-t border-slate-200 flex items-center justify-between">
                    <div className="text-xs text-[hsl(var(--muted-foreground))]">
                      {images.length} 枚選択中（クリックでサブ画像をメインにできます）
                    </div>
                    <div className="flex items-center gap-2">
                      <Button type="button" variant="outline" size="sm" onClick={openImagePicker}>
                        画像を追加
                      </Button>
                      {images.length > 0 && (
                        <Button type="button" variant="ghost" size="sm" onClick={clearImages}>
                          クリア
                        </Button>
                      )}
                    </div>
                  </div>
                </div>

                {/* サブ（小） + 追加タイル */}
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  {thumbIndices.map((idx) => {
                    const url = imagePreviewUrls[idx];
                    const f = images[idx];
                    return (
                      <div
                        key={`${f.name}-${f.size}-${f.lastModified}-${idx}`}
                        className="relative rounded-xl overflow-hidden border border-slate-200 bg-white cursor-pointer"
                        onClick={() => setMainImageIndex(idx)}
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

                        <button
                          type="button"
                          className="absolute top-2 right-2 w-7 h-7 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                          onClick={(e) => {
                            e.stopPropagation();
                            removeImageAt(idx);
                          }}
                          aria-label="remove image"
                          title="削除"
                        >
                          <span className="text-slate-600 leading-none">×</span>
                        </button>

                        <div className="px-2 py-2 border-t border-slate-200">
                          <div className="text-xs truncate">{f.name}</div>
                        </div>
                      </div>
                    );
                  })}

                  {/* 追加タイル（アップロードと並列表示） */}
                  <div
                    className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 cursor-pointer flex flex-col items-center justify-center gap-2 aspect-square"
                    onClick={openImagePicker}
                    onDrop={onDropImages}
                    onDragOver={onDragOverImages}
                    role="button"
                    tabIndex={0}
                    title="画像を追加"
                  >
                    <div className="w-10 h-10 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                      <PlusIcon />
                    </div>
                    <div className="text-xs text-slate-700">画像を追加</div>
                  </div>
                </div>
              </>
            )}

            <div className="text-xs text-[hsl(var(--muted-foreground))]">
              ※アップロード処理は後で実装（現状は選択UIのみ）。
            </div>
          </CardContent>
        </Card>

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
            <div className="text-xs text-[hsl(var(--muted-foreground))]">
              出品一覧で表示されるタイトルです。
            </div>
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
            <div className="text-xs text-[hsl(var(--muted-foreground))]">
              購入者に伝えたいポイントを記入してください。
            </div>
          </CardContent>
        </Card>

        {/* ✅ PriceCard */}
        <PriceCard
          title="価格"
          rows={priceRows}
          mode="edit"
          currencySymbol="¥"
          onChangePrice={(idx, price) => onChangePrice(idx, price)}
        />

        {priceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格行データは未取得です（DTO/別APIから rows を供給する実装が必要です）。
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
          <div className="text-sm text-red-600">読み込みに失敗しました: {dtoError}</div>
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
            <div className="text-sm text-slate-800 break-all">{productBrandName || "未選択"}</div>
            <div className="text-sm text-slate-800 break-all">{productName || "未選択"}</div>
          </CardContent>
        </Card>

        {/* ✅ 選択トークンカード */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択トークン</div>
            <div className="text-sm text-slate-800 break-all">{tokenBrandName || "未選択"}</div>
            <div className="text-sm text-slate-800 break-all">{tokenName || "未選択"}</div>
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
