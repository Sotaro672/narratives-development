// frontend/console/list/src/presentation/hook/useListImageCard.tsx
// listImageCard のロジックを集約（inventory/listDetail 両対応）

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

  // inventory/useListCreate 用（input ref + handlers）
  imageInputRef?: React.RefObject<HTMLInputElement | null>;
  onSelectImages?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onDropImages?: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages?: (e: React.DragEvent<HTMLDivElement>) => void;

  // listDetail/useListDetail 用（files だけ渡す互換）
  onAddImages?: (files: FileList | null) => void;

  // remove/clear
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;
};

export type UseListImageCardResult = {
  // computed
  effectiveImageUrls: string[];
  hasImages: boolean;
  mainUrl: string;
  thumbIndices: number[];

  imageInputRef: React.RefObject<HTMLInputElement | null>;

  // handlers
  openPicker: () => void;
  handleInputChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  handleClear: () => void;
  handleRemoveAt: (idx: number) => void;
  handleSetMainIndex: (idx: number) => void;

  // drag/drop handlers
  onDropImages?: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages?: (e: React.DragEvent<HTMLDivElement>) => void;
};

export function useListImageCard(args: UseListImageCardArgs): UseListImageCardResult {
  const internalRef = React.useRef<HTMLInputElement | null>(null);
  const imageInputRef = (args.imageInputRef ?? internalRef) as React.RefObject<HTMLInputElement | null>;

  const effectiveImageUrls: string[] = React.useMemo(() => {
    const base = Array.isArray(args.imageUrls) ? args.imageUrls : [];
    return base.map((u) => s(u)).filter(Boolean);
  }, [args.imageUrls]);

  const hasImages = effectiveImageUrls.length > 0;

  const safeMainIndex = React.useMemo(() => {
    if (!hasImages) return 0;
    const n = effectiveImageUrls.length;
    const idx = Number.isFinite(args.mainImageIndex) ? args.mainImageIndex : 0;
    if (idx < 0) return 0;
    if (idx >= n) return 0;
    return idx;
  }, [hasImages, effectiveImageUrls.length, args.mainImageIndex]);

  const mainUrl = hasImages ? effectiveImageUrls[safeMainIndex] ?? "" : "";

  const thumbIndices: number[] = React.useMemo(() => {
    if (!hasImages) return [];
    return effectiveImageUrls
      .map((_, idx) => idx)
      .filter((idx) => idx !== safeMainIndex);
  }, [hasImages, effectiveImageUrls, safeMainIndex]);

  const openPicker = React.useCallback(() => {
    if (!args.isEdit) return;
    imageInputRef.current?.click();
  }, [args.isEdit, imageInputRef]);

  const handleInputChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (typeof args.onSelectImages === "function") {
        args.onSelectImages(e);
        return;
      }

      args.onAddImages?.(e.target.files);

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
      args.setMainImageIndex(idx);
    },
    [args.setMainImageIndex],
  );

  const handleClear = React.useCallback(() => {
    if (!args.isEdit) return;

    if (typeof args.onClearImages === "function") {
      args.onClearImages();
      args.setMainImageIndex(0);
      return;
    }

    if (typeof args.onRemoveImageAt === "function") {
      for (let i = effectiveImageUrls.length - 1; i >= 0; i--) {
        args.onRemoveImageAt(i);
      }
      args.setMainImageIndex(0);
    }
  }, [
    args.isEdit,
    args.onClearImages,
    args.onRemoveImageAt,
    args.setMainImageIndex,
    effectiveImageUrls.length,
  ]);

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