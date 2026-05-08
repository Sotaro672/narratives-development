// frontend/console/inventory/src/presentation/hook/listCreate/useListingImages.ts
import * as React from "react";

import { dedupeFiles, type ImageInputRef } from "../../../application/listCreate/listCreateService";

export function useListingImages(): {
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef;
  onSelectImages: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onDropImages: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages: (e: React.DragEvent<HTMLDivElement>) => void;
  removeImageAt: (idx: number) => void;
  clearImages: () => void;
} {
  const [images, setImages] = React.useState<File[]>([]);
  const [mainImageIndex, setMainImageIndex] = React.useState<number>(0);

  const imageInputRef = React.useRef<HTMLInputElement | null>(null);

  const onSelectImages = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(e.target.files ?? []).filter(Boolean) as File[];
      if (files.length === 0) return;

      const next = dedupeFiles(images, files);
      setImages(next);

      // eslint-disable-next-line no-console
      console.log("[inventory/listImage] selected", {
        addedCount: files.length,
        totalCount: next.length,
        names: next.slice(0, 6).map((f) => f.name),
      });

      e.currentTarget.value = "";
    },
    [images],
  );

  const onDropImages = React.useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      const files = Array.from(e.dataTransfer.files ?? [])
        .filter(Boolean)
        .filter((f) => String(f.type || "").startsWith("image/")) as File[];

      if (files.length === 0) return;

      const next = dedupeFiles(images, files);
      setImages(next);

      // eslint-disable-next-line no-console
      console.log("[inventory/listImage] dropped", {
        addedCount: files.length,
        totalCount: next.length,
        names: next.slice(0, 6).map((f) => f.name),
      });
    },
    [images],
  );

  const onDragOverImages = React.useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const removeImageAt = React.useCallback((idx: number) => {
    setImages((prev) => {
      const next = prev.filter((_, i) => i !== idx);

      // eslint-disable-next-line no-console
      console.log("[inventory/listImage] removed", {
        removedIndex: idx,
        totalCount: next.length,
        names: next.slice(0, 6).map((f) => f.name),
      });

      return next;
    });

    setMainImageIndex((prevMain) => {
      if (idx === prevMain) return 0;
      if (idx < prevMain) return Math.max(0, prevMain - 1);
      return prevMain;
    });
  }, []);

  const clearImages = React.useCallback(() => {
    setImages([]);

    // eslint-disable-next-line no-console
    console.log("[inventory/listImage] cleared", { totalCount: 0 });

    setMainImageIndex(0);
  }, []);

  const [imagePreviewUrls, setImagePreviewUrls] = React.useState<string[]>([]);
  React.useEffect(() => {
    if (images.length === 0) {
      setImagePreviewUrls([]);
      return;
    }

    const urls = images.map((f) => URL.createObjectURL(f));
    setImagePreviewUrls(urls);

    return () => {
      urls.forEach((u) => {
        try {
          URL.revokeObjectURL(u);
        } catch {
          // noop
        }
      });
    };
  }, [images]);

  React.useEffect(() => {
    if (images.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }
    if (mainImageIndex < 0 || mainImageIndex > images.length - 1) {
      setMainImageIndex(0);
    }
  }, [images.length, mainImageIndex]);

  return {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef: imageInputRef as unknown as ImageInputRef,
    onSelectImages,
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,
  };
}
