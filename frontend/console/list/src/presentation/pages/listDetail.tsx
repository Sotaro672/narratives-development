// frontend/list/src/pages/listDetail.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";

// ✅ PriceCard（list app 側のコンポーネント）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ hook
import { useListDetail } from "../../../../list/src/presentation/hook/useListDetail";

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

// local trim helper (UI-only)
function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function ListDetail() {
  // ✅ listDetail hook の返却型差分に強くする（UI 側で必要な情報を any で吸収）
  const vm = useListDetail() as any;

  // -----------------------------
  // ViewModel normalization (best-effort)
  // -----------------------------
  const loading = !!vm?.loading;
  const error = s(vm?.error || vm?.dtoError);

  const listingTitle =
    s(vm?.listingTitle) || s(vm?.title) || s(vm?.list?.title) || "";
  const description =
    s(vm?.description) || s(vm?.list?.description) || s(vm?.detail?.description) || "";

  const assigneeName =
    s(vm?.assigneeName) || s(vm?.admin?.assigneeName) || "未設定";

  const productBrandName =
    s(vm?.productBrandName) || s(vm?.product?.brandName) || s(vm?.productBlueprint?.brandName);
  const productName =
    s(vm?.productName) || s(vm?.product?.name) || s(vm?.productBlueprint?.productName);

  const tokenBrandName =
    s(vm?.tokenBrandName) || s(vm?.token?.brandName) || s(vm?.tokenBlueprint?.brandName);
  const tokenName =
    s(vm?.tokenName) || s(vm?.token?.name) || s(vm?.tokenBlueprint?.tokenName);

  // decision/status (view)
  const decision =
    s(vm?.decision) ||
    s(vm?.status) ||
    s(vm?.list?.status) ||
    "";

  // price rows (view)
  const priceRows = (vm?.priceRows || vm?.prices || vm?.list?.priceRows || vm?.list?.prices || []) as any[];

  // images (view)
  // - create 画面の images: File[]
  // - detail 画面は URL 配列 or image objects を想定（best-effort で拾う）
  const imageUrls: string[] = React.useMemo(() => {
    const urls =
      (vm?.imagePreviewUrls as string[]) ||
      (vm?.imageUrls as string[]) ||
      (vm?.images as any[])?.map((x: any) => s(x?.url || x?.src || x?.publicUrl || x?.downloadUrl)).filter(Boolean) ||
      (vm?.list?.images as any[])?.map((x: any) => s(x?.url || x?.src || x?.publicUrl || x?.downloadUrl)).filter(Boolean) ||
      [];
    return urls.filter((u) => !!s(u));
  }, [vm]);

  const hasImages = imageUrls.length > 0;

  // メイン画像は detail 側ではローカル state で切替（view only）
  const [mainImageIndex, setMainImageIndex] = React.useState(0);

  React.useEffect(() => {
    // 画像が減った時のガード
    if (!hasImages) {
      setMainImageIndex(0);
      return;
    }
    if (mainImageIndex >= imageUrls.length) {
      setMainImageIndex(0);
    }
  }, [hasImages, imageUrls.length, mainImageIndex]);

  const mainUrl = hasImages ? imageUrls[mainImageIndex] : "";

  const thumbIndices = React.useMemo(() => {
    if (!hasImages) return [];
    return imageUrls.map((_, idx) => idx).filter((idx) => idx !== mainImageIndex);
  }, [hasImages, imageUrls, mainImageIndex]);

  return (
    <PageStyle
      layout="grid-2"
      title={s(vm?.pageTitle) || "出品詳細"}
      onBack={vm?.onBack}
      onSave={undefined}
    >
      {/* =========================
          左カラム（create 画面を模倣 / view-only）
          - 商品画像
          - タイトル
          - 説明
          - 価格（PriceCard view）
          ========================= */}
      <div className="space-y-4">
        {/* 状態表示（任意） */}
        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">読み込み中...</div>
        )}
        {error && (
          <div className="text-sm text-red-600">読み込みに失敗しました: {error}</div>
        )}

        {/* ✅ 商品画像カード（view-only） */}
        <Card>
          <CardContent className="p-4 space-y-3">
            <div className="text-sm font-medium flex items-center gap-2">
              <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-slate-50 border border-slate-200">
                <ImageIcon />
              </span>
              商品画像
            </div>

            {!hasImages && (
              <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 w-full aspect-[16/9] flex flex-col items-center justify-center gap-3 select-none">
                <div className="w-12 h-12 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                  <ImageIcon />
                </div>
                <div className="text-sm text-slate-700">画像は未設定です</div>
                <div className="text-xs text-[hsl(var(--muted-foreground))]">
                  画像を追加する場合は「画像」機能（別画面/別操作）から追加してください。
                </div>
              </div>
            )}

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

                  <div className="px-3 py-2 border-t border-slate-200 flex items-center justify-between">
                    <div className="text-xs text-[hsl(var(--muted-foreground))]">
                      {imageUrls.length} 枚（クリックでサブ画像をメインにできます）
                    </div>
                  </div>
                </div>

                {/* サブ（小） */}
                {thumbIndices.length > 0 && (
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {thumbIndices.map((idx) => {
                      const url = imageUrls[idx];
                      return (
                        <div
                          key={`${url}-${idx}`}
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
                        </div>
                      );
                    })}
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>

        {/* ✅ タイトル（view-only） */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">タイトル</div>
            <div className="text-sm text-slate-800 break-words">
              {listingTitle || "未設定"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 説明（view-only） */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">説明</div>
            <div className="text-sm text-slate-800 whitespace-pre-wrap break-words">
              {description || "未設定"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ PriceCard（view mode） */}
        <PriceCard
          title="価格"
          rows={priceRows as any}
          mode="view"
          currencySymbol="¥"
          // view なので onChangePrice は渡さない
        />

        {Array.isArray(priceRows) && priceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格情報がありません。
          </div>
        )}
      </div>

      {/* =========================
          右カラム（create 画面を模倣 / view-only）
          - 担当者
          - 選択商品
          - 選択トークン
          - 出品｜保留（表示のみ）
          ========================= */}
      <div className="space-y-4">
        {/* ✅ 担当者（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">担当者</div>
            <div className="text-sm text-slate-800 break-all">{assigneeName}</div>

            {/* 管理情報（あれば） */}
            {(vm?.admin?.createdByName || vm?.admin?.createdAt) && (
              <div className="mt-3 text-xs text-[hsl(var(--muted-foreground))] space-y-1">
                {vm?.admin?.createdByName && (
                  <div>作成者: {s(vm.admin.createdByName)}</div>
                )}
                {vm?.admin?.createdAt && (
                  <div>作成日時: {s(vm.admin.createdAt)}</div>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        {/* ✅ 選択商品（view-only） */}
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

        {/* ✅ 選択トークン（view-only） */}
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

        {/* ✅ 出品｜保留（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            <div className="flex gap-2">
              <div
                className={[
                  "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                  s(decision).toLowerCase() === "list"
                    ? "bg-slate-900 text-white border-slate-900"
                    : "bg-white text-slate-700 border-slate-200",
                ].join(" ")}
              >
                出品
              </div>

              <div
                className={[
                  "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                  s(decision).toLowerCase() === "hold"
                    ? "bg-slate-900 text-white border-slate-900"
                    : "bg-white text-slate-700 border-slate-200",
                ].join(" ")}
              >
                保留
              </div>
            </div>

            {decision && (
              <div className="mt-2 text-xs text-[hsl(var(--muted-foreground))]">
                現在: {decision}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
