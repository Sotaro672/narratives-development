// frontend/console/list/src/presentation/components/listImageCard.tsx
// ✅ 商品画像カード（style 要素のみ / ロジックは hook に移譲）

import * as React from "react";

// ✅ rollup で "./listImageCard.css" が解決できないため、styles 配下へ統一
import "../styles/list.css";

import { Button } from "../../../../shell/src/shared/ui/button";
import { useListImageCard } from "../hook/useListImageCard";

function ImageIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" className="ivc__icon">
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
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" className="ivc__icon">
      <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
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

  // ✅ inventory/useListCreate 用（input ref + handlers）
  imageInputRef?: React.RefObject<HTMLInputElement | null>;
  onSelectImages?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onDropImages?: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages?: (e: React.DragEvent<HTMLDivElement>) => void;

  // ✅ listDetail/useListDetail 用（files だけ渡す互換）
  onAddImages?: (files: FileList | null) => void;

  // remove/clear
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;

  // anyVm fallback (optional)
  anyVm?: any;
};

export default function ListImageCard(props: ListImageCardProps) {
  const vm = useListImageCard({
    isEdit: props.isEdit,
    saving: props.saving,
    imageUrls: props.imageUrls,
    mainImageIndex: props.mainImageIndex,
    setMainImageIndex: props.setMainImageIndex,

    imageInputRef: props.imageInputRef,
    onSelectImages: props.onSelectImages,
    onDropImages: props.onDropImages,
    onDragOverImages: props.onDragOverImages,

    onAddImages: props.onAddImages,

    onRemoveImageAt: props.onRemoveImageAt,
    onClearImages: props.onClearImages,
    anyVm: props.anyVm,
  });

  return (
    <div className="ivc">
      {/* Header */}
      <div className="ivc__header">
        <div className="lic__header">
          <div className="lic__header-left">
            <span className="lic__icon-wrap">
              <ImageIcon />
            </span>
            <span className="ivc__title">商品画像</span>
          </div>

          {/* ✅ ボタンは廃止。残すのは「クリア」のみ（任意） */}
          {props.isEdit && vm.effectiveImageUrls.length > 0 && (
            <div className="lic__actions">
              <Button
                type="button"
                variant="ghost"
                className="h-8"
                onClick={vm.handleClear}
                disabled={Boolean(props.saving)}
              >
                クリア
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Body */}
      <div className="ivc__body">
        {/* ✅ hidden input（クリックで openPicker） */}
        <input
          ref={vm.imageInputRef as any}
          type="file"
          accept="image/*"
          multiple
          className="hidden"
          onChange={vm.handleInputChange}
        />

        {/* empty state */}
        {!vm.hasImages && (
          <div
            className={["lic__empty", props.isEdit ? "lic__empty--clickable" : ""].join(" ")}
            onClick={vm.openPicker}
            onDrop={vm.onDropImages}
            onDragOver={vm.onDragOverImages}
            role="button"
            tabIndex={0}
            title={props.isEdit ? "クリックで画像を追加" : undefined}
          >
            <div className="lic__empty-icon">
              <ImageIcon />
            </div>
            <div className="lic__empty-title">画像を追加</div>
            <div className="lic__empty-sub">
              {props.isEdit ? "クリックで選択（複数可） / ドロップでも追加できます" : "編集モードで追加できます"}
            </div>
          </div>
        )}

        {/* filled state */}
        {vm.hasImages && (
          <>
            {/* メイン（大） */}
            <div
              className="lic__main"
              onDrop={vm.onDropImages}
              onDragOver={vm.onDragOverImages}
              title={props.isEdit ? "クリックで画像追加（複数可）" : undefined}
            >
              <div
                className={["lic__main-media", props.isEdit ? "lic__main-media--clickable" : ""].join(
                  " ",
                )}
                onClick={vm.openPicker}
                role={props.isEdit ? "button" : undefined}
                tabIndex={props.isEdit ? 0 : undefined}
              >
                {vm.mainUrl && <img src={vm.mainUrl} alt="main" className="lic__img" />}
              </div>

              {/* remove main */}
              {props.isEdit && (
                <button
                  type="button"
                  className="lic__remove-btn"
                  onClick={(e) => {
                    e.stopPropagation();
                    vm.handleRemoveAt(props.mainImageIndex);
                  }}
                  aria-label="remove main image"
                  title="削除"
                  disabled={Boolean(props.saving)}
                >
                  <span className="lic__remove-x">×</span>
                </button>
              )}

              {/* footer */}
              <div className="lic__footer">
                <div className="lic__footer-left">
                  {vm.effectiveImageUrls.length} 枚
                  {props.isEdit ? "（×で削除 / クリックで追加）" : "（サムネでメイン切替）"}
                </div>
                {!props.isEdit && <div className="lic__footer-note">※ 画像変更は編集モードで行えます</div>}
              </div>
            </div>

            {/* サブ（小） + 追加タイル */}
            <div className="lic__grid">
              {vm.thumbIndices.map((idx: number) => {
                const url = vm.effectiveImageUrls[idx] ?? "";
                return (
                  <div
                    key={`${url}-${idx}`}
                    className="lic__thumb"
                    onClick={() => vm.handleSetMainIndex(idx)}
                    role="button"
                    tabIndex={0}
                    title="クリックでメインに設定"
                  >
                    <div className="lic__thumb-media">
                      {url && <img src={url} alt={`sub-${idx}`} className="lic__img" />}
                    </div>

                    {props.isEdit && (
                      <button
                        type="button"
                        className="lic__thumb-remove"
                        onClick={(e) => {
                          e.stopPropagation();
                          vm.handleRemoveAt(idx);
                        }}
                        aria-label="remove image"
                        title="削除"
                        disabled={Boolean(props.saving)}
                      >
                        <span className="lic__remove-x">×</span>
                      </button>
                    )}
                  </div>
                );
              })}

              {/* ✅ 追加タイル（押下でエクスプローラー） */}
              {props.isEdit && (
                <div
                  className="lic__add-tile"
                  onClick={vm.openPicker}
                  onDrop={vm.onDropImages}
                  onDragOver={vm.onDragOverImages}
                  role="button"
                  tabIndex={0}
                  title="クリックで画像を追加（複数可）"
                >
                  <div className="lic__empty-icon">
                    <PlusIcon />
                  </div>
                  <div className="lic__add-title">画像を追加</div>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
