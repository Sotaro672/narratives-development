// frontend/console/shell/src/features/list/presentation/hook/useListImageCard.tsx

import * as React from "react";

export type UseListImageCardArgs = {
  isEdit: boolean;
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: (idx: number) => void;
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;
};

export type UseListImageCardResult = {
  effectiveImageUrls: string[];
  hasImages: boolean;
  mainUrl: string;
  thumbIndices: number[];
  handleClear: () => void;
  handleRemoveAt: (idx: number) => void;
  handleSetMainIndex: (idx: number) => void;
};

export function useListImageCard(
  args: UseListImageCardArgs,
): UseListImageCardResult {
  const effectiveImageUrls = React.useMemo(() => {
    const base = Array.isArray(args.imageUrls)
      ? args.imageUrls
      : [];

    return base
      .map((url) => String(url ?? "").trim())
      .filter(Boolean);
  }, [args.imageUrls]);

  const hasImages = effectiveImageUrls.length > 0;

  const safeMainIndex = React.useMemo(() => {
    if (!hasImages) {
      return 0;
    }

    const count = effectiveImageUrls.length;
    const index = Number.isFinite(args.mainImageIndex)
      ? args.mainImageIndex
      : 0;

    if (index < 0 || index >= count) {
      return 0;
    }

    return index;
  }, [
    hasImages,
    effectiveImageUrls.length,
    args.mainImageIndex,
  ]);

  const mainUrl = hasImages
    ? effectiveImageUrls[safeMainIndex] ?? ""
    : "";

  const thumbIndices = React.useMemo(() => {
    if (!hasImages) {
      return [];
    }

    return effectiveImageUrls
      .map((_, index) => index)
      .filter((index) => index !== safeMainIndex);
  }, [
    hasImages,
    effectiveImageUrls,
    safeMainIndex,
  ]);

  const handleRemoveAt = React.useCallback(
    (idx: number) => {
      if (!args.isEdit) {
        return;
      }

      args.onRemoveImageAt?.(idx);
    },
    [
      args.isEdit,
      args.onRemoveImageAt,
    ],
  );

  const handleSetMainIndex = React.useCallback(
    (idx: number) => {
      args.setMainImageIndex(idx);
    },
    [args.setMainImageIndex],
  );

  const handleClear = React.useCallback(() => {
    if (!args.isEdit) {
      return;
    }

    args.onClearImages?.();
    args.setMainImageIndex(0);
  }, [
    args.isEdit,
    args.onClearImages,
    args.setMainImageIndex,
  ]);

  return {
    effectiveImageUrls,
    hasImages,
    mainUrl,
    thumbIndices,
    handleClear,
    handleRemoveAt,
    handleSetMainIndex,
  };
}