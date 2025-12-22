// frontend/console/list/src/presentation/components/listImageCard.tsx
// ✅ 商品画像カード（listDetail から分離）

import * as React from "react";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

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

export type ListImageCardProps = {
  isEdit: boolean;
  saving?: boolean;

  // urls + selection
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: (idx: number) => void;

  // handlers
  onAddImages?: (files: FileList | null) => void;
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;

  // anyVm fallback (optional)
  anyVm?: any;
};

export default function ListImageCard(props: ListImageCardProps) {
  const anyVm = props.anyVm as any;

  const effectiveImageUrls: string[] = React.useMemo(() => {
    const arr = Array.isArray(props.imageUrls) ? props.imageUrls : [];
    return arr.map((u) => s(u)).filter(Boolean);
  }, [props.imageUrls]);

  const hasImages = effectiveImageUrls.length > 0;

  const mainUrl = hasImages ? effectiveImageUrls[props.mainImageIndex] : "";

  const thumbIndices: number[] = React.useMemo(() => {
    if (!hasImages) return [];
    return effectiveImageUrls
      .map((_: string, idx: number) => idx)
      .filter((idx: number) => idx !== props.mainImageIndex);
  }, [hasImages, effectiveImageUrls, props.mainImageIndex]);

  return (
    <Card>
      <CardContent className="p-4 space-y-3">
        <div className="text-sm font-medium flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-slate-50 border border-slate-200">
              <ImageIcon />
            </span>
            商品画像
          </div>

          {props.isEdit && (
            <div className="flex items-center gap-2">
              <label className="cursor-pointer">
                <input
                  type="file"
                  accept="image/*"
                  multiple
                  className="hidden"
                  onChange={(e) => props.onAddImages?.(e.target.files)}
                />
                <Button
                  type="button"
                  variant="outline"
                  className="h-8"
                  disabled={Boolean(props.saving)}
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
                    if (typeof props.onClearImages === "function") {
                      props.onClearImages();
                      return;
                    }
                    if (typeof anyVm?.onClearImages === "function") {
                      anyVm.onClearImages();
                      return;
                    }

                    // fallback: 末尾から削除（indexがずれるのを避ける）
                    if (typeof props.onRemoveImageAt === "function") {
                      for (let i = effectiveImageUrls.length - 1; i >= 0; i--) {
                        props.onRemoveImageAt(i);
                      }
                      props.setMainImageIndex(0);
                    }
                  }}
                  disabled={Boolean(props.saving)}
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
              props.isEdit ? "cursor-pointer" : "",
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
              {props.isEdit
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
              {props.isEdit && (
                <button
                  type="button"
                  className="absolute top-3 right-3 w-8 h-8 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                  onClick={() => props.onRemoveImageAt?.(props.mainImageIndex)}
                  aria-label="remove main image"
                  title="削除"
                  disabled={Boolean(props.saving)}
                >
                  <span className="text-slate-600 leading-none">×</span>
                </button>
              )}

              {/* footer */}
              <div className="px-3 py-2 border-t border-slate-200 flex items-center justify-between">
                <div className="text-xs text-[hsl(var(--muted-foreground))]">
                  {effectiveImageUrls.length} 枚
                  {props.isEdit
                    ? "（サムネの×で削除できます）"
                    : "（サムネをクリックしてメイン切替できます）"}
                </div>
                {!props.isEdit && (
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
                    onClick={() => props.setMainImageIndex(idx)}
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

                    {props.isEdit && (
                      <button
                        type="button"
                        className="absolute top-2 right-2 w-7 h-7 rounded-full bg-white/90 border border-slate-200 flex items-center justify-center hover:bg-white"
                        onClick={(e) => {
                          e.stopPropagation();
                          props.onRemoveImageAt?.(idx);
                        }}
                        aria-label="remove image"
                        title="削除"
                        disabled={Boolean(props.saving)}
                      >
                        <span className="text-slate-600 leading-none">×</span>
                      </button>
                    )}
                  </div>
                );
              })}

              {/* 追加タイル（edit時のみ） */}
              {props.isEdit && (
                <label
                  className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 cursor-pointer flex flex-col items-center justify-center gap-2 aspect-square"
                  title="画像を追加（複数可）"
                >
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    className="hidden"
                    onChange={(e) => props.onAddImages?.(e.target.files)}
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
  );
}
