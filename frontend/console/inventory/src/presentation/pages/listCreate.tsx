// frontend/console/inventory/src/presentation/pages/listCreate.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Select } from "../../../../shell/src/shared/ui/select";

import PriceCard from "../../../../list/presentation/components/priceCard";
import ListImageCard from "../../../../list/presentation/components/listImageCard";
import { useListCreate } from "../hook/useListCreate";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function InventoryListCreate() {
  const {
    onBack,
    onCreate,

    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
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

    decision,
    setDecision,
  } = useListCreate();

  const missingModelIdCount = React.useMemo(() => {
    return (priceRows ?? []).filter((r: any) => !s(r?.modelId)).length;
  }, [priceRows]);

  const headerTitle = React.useMemo(() => {
    const safeProductName = s(productName);
    const safeTokenName = s(tokenName);

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

  const assigneeOptions = React.useMemo(() => {
    return (assigneeCandidates ?? []).map((c) => ({
      value: c.name,
      label: c.name,
    }));
  }, [assigneeCandidates]);

  const handleChangeAssignee = React.useCallback(
    (selectedName: string) => {
      const matched = (assigneeCandidates ?? []).find(
        (c) => c.name === selectedName,
      );
      if (!matched) return;

      handleSelectAssignee(matched.id);
    },
    [assigneeCandidates, handleSelectAssignee],
  );

  return (
    <PageStyle
      layout="grid-2"
      title={headerTitle}
      onBack={onBack}
      onList={onCreate}
    >
      {/* 左カラム */}
      <div className="space-y-4">
        <ListImageCard
          isEdit={true}
          saving={false}
          imageUrls={Array.isArray(imagePreviewUrls) ? imagePreviewUrls : []}
          mainImageIndex={mainImageIndex}
          setMainImageIndex={setMainImageIndex}
          onAddImages={onAddImages}
          onRemoveImageAt={onRemoveImageAt}
          onClearImages={onClearImages}
        />

        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">タイトル</div>
            <input
              value={listingTitle}
              onChange={(e) => setListingTitle(e.target.value)}
              placeholder="例: Narratives シャツ1（赤 / S・M）"
              className="w-full h-10 px-3 rounded-md border border-slate-200 bg-white text-sm outline-none focus:ring-2 focus:ring-slate-200"
            />
          </CardContent>
        </Card>

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

        <PriceCard
          title="価格"
          rows={priceRows as any}
          mode="edit"
          currencySymbol="¥"
          onChangePrice={(idx: number, price: number | null) =>
            onChangePrice(idx, price)
          }
        />

        {priceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格行データは未取得です（DTO/別APIから rows を供給する実装が必要です）。
          </div>
        )}

        {missingModelIdCount > 0 && (
          <div className="text-xs text-red-600">
            modelId が未設定の価格行があります: {missingModelIdCount} 件
          </div>
        )}
      </div>

      {/* 右カラム */}
      <div className="space-y-4">
        {loadingDTO && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        )}

        {dtoError && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {dtoError}
          </div>
        )}

        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">担当者</div>

            {loadingMembers ? (
              <div className="text-xs text-slate-400">
                担当者を読み込み中です…
              </div>
            ) : assigneeOptions.length > 0 ? (
              <Select
                options={assigneeOptions}
                value={assigneeName || ""}
                onChange={handleChangeAssignee}
                placeholder="担当者を選択してください"
              />
            ) : (
              <div className="text-xs text-slate-400">
                担当者候補がありません。
              </div>
            )}
          </CardContent>
        </Card>

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