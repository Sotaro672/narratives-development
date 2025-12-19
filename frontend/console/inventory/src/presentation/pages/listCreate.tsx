// frontend/console/inventory/src/presentation/pages/listCreate.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

// ✅ NEW: PriceCard（list app 側に作ったコンポーネントを流用）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ logic は hook 側へ
import { useListCreate } from "../hook/useListCreate";

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

    // assignee
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,

    // decision
    decision,
    setDecision,
  } = useListCreate();

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onCreate={onCreate} // ✅ PageHeader に「作成」ボタンを表示
    >
      {/* =========================
          左カラム：PriceCard
          ========================= */}
      <div className="space-y-4">
        <PriceCard
          title="価格"
          rows={priceRows}
          mode="edit"
          currencySymbol="¥"
          onChangePrice={(idx, price) => onChangePrice(idx, price)}
        />

        {/* 補足（必要なら削除OK） */}
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
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        )}
        {dtoError && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {dtoError}
          </div>
        )}

        {/* ✅ 担当者（title: 担当者） */}
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
                  <p className="text-xs text-slate-400">
                    担当者を読み込み中です…
                  </p>
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
                  <p className="text-xs text-slate-400">
                    担当者候補がありません。
                  </p>
                )}
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        {/* ✅ 選択商品カード：productName / brandName（DTO） */}
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

        {/* ✅ 選択トークンカード：tokenName / brandName（DTO） */}
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

        {/* ✅ 出品｜保留 選択カード */}
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
