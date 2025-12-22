// frontend/console/list/src/presentation/hook/useListImageCard.tsx
// ✅ listImageCard のロジックを集約（inventory/listDetail 両対応）

import * as React from "react";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export type UseListImageCardArgs = {
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

export type UseListImageCardResult = {
  // computed
  effectiveImageUrls: string[];
  hasImages: boolean;
  mainUrl: string;
  thumbIndices: number[];

  // ✅ component が参照しているので必ず返す
  imageInputRef: React.RefObject<HTMLInputElement | null>;

  // handlers
  openPicker: () => void;
  handleInputChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  handleClear: () => void;

  // ✅ component が参照しているので必ず返す
  handleRemoveAt: (idx: number) => void;
  handleSetMainIndex: (idx: number) => void;

  // drag/drop handlers (component がそのまま渡せるように)
  onDropImages?: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages?: (e: React.DragEvent<HTMLDivElement>) => void;
};

export function useListImageCard(args: UseListImageCardArgs): UseListImageCardResult {
  const anyVm = args.anyVm as any;

  // ✅ ref は必ず返す（inventory から渡されない場合でも動くように internal ref を持つ）
  const internalRef = React.useRef<HTMLInputElement | null>(null);
  const imageInputRef = (args.imageInputRef ?? internalRef) as React.RefObject<HTMLInputElement | null>;

  const effectiveImageUrls: string[] = React.useMemo(() => {
    const arr = Array.isArray(args.imageUrls) ? args.imageUrls : [];
    return arr.map((u) => s(u)).filter(Boolean);
  }, [args.imageUrls]);

  const hasImages = effectiveImageUrls.length > 0;

  const mainUrl = hasImages ? effectiveImageUrls[args.mainImageIndex] ?? "" : "";

  const thumbIndices: number[] = React.useMemo(() => {
    if (!hasImages) return [];
    return effectiveImageUrls
      .map((_, idx) => idx)
      .filter((idx) => idx !== args.mainImageIndex);
  }, [hasImages, effectiveImageUrls, args.mainImageIndex]);

  const openPicker = React.useCallback(() => {
    if (!args.isEdit) return;
    imageInputRef.current?.click();
  }, [args.isEdit, imageInputRef]);

  const handleInputChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      // ✅ inventory 側は event handler をそのまま渡す
      if (typeof args.onSelectImages === "function") {
        args.onSelectImages(e);
        return;
      }

      // ✅ listDetail 側は files を渡す互換
      args.onAddImages?.(e.target.files);

      // 同じファイルを再選択できるように（安全側）
      try {
        e.currentTarget.value = "";
      } catch {
        // noop
      }
    },
    [args.onSelectImages, args.onAddImages],
  );

  const handleRemoveAt = React.useCallback(
    (idx: number) => {
      if (!args.isEdit) return;
      args.onRemoveImageAt?.(idx);
    },
    [args.isEdit, args.onRemoveImageAt],
  );

  const handleSetMainIndex = React.useCallback(
    (idx: number) => {
      // view/edit どちらでも main 切替は許可（既存挙動に合わせる）
      args.setMainImageIndex(idx);
    },
    [args.setMainImageIndex],
  );

  const handleClear = React.useCallback(() => {
    if (!args.isEdit) return;

    // 1) hook が提供する clear があればそれを優先
    if (typeof args.onClearImages === "function") {
      args.onClearImages();
      args.setMainImageIndex(0);
      return;
    }

    // 2) anyVm fallback
    if (typeof anyVm?.onClearImages === "function") {
      anyVm.onClearImages();
      args.setMainImageIndex(0);
      return;
    }

    // 3) removeAt で全削除（末尾から）
    if (typeof args.onRemoveImageAt === "function") {
      for (let i = effectiveImageUrls.length - 1; i >= 0; i--) {
        args.onRemoveImageAt(i);
      }
      args.setMainImageIndex(0);
    }
  }, [args.isEdit, args.onClearImages, args.onRemoveImageAt, args.setMainImageIndex, anyVm, effectiveImageUrls.length]);

  return {
    effectiveImageUrls,
    hasImages,
    mainUrl,
    thumbIndices,

    imageInputRef,

    openPicker,
    handleInputChange,
    handleClear,

    handleRemoveAt,
    handleSetMainIndex,

    onDropImages: args.onDropImages,
    onDragOverImages: args.onDragOverImages,
  };
}
