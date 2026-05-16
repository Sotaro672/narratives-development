// frontend/console/inventory/src/presentation/hook/listCreate/useListingImages.ts

import * as React from "react";

type ImageInputRef = React.RefObject<HTMLInputElement | null>;

function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(
    prev.map((file: File) => `${file.name}__${file.size}__${file.lastModified}`),
  );

  const filtered = add.filter(
    (file: File) => !exists.has(`${file.name}__${file.size}__${file.lastModified}`),
  );

  return [...prev, ...filtered];
}

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
        names: next.slice(0, 6).map((file: File) => file.name),
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
        .filter((file: File) => String(file.type || "").startsWith("image/")) as File[];

      if (files.length === 0) return;

      const next = dedupeFiles(images, files);
      setImages(next);
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
        names: next.slice(0, 6).map((file: File) => file.name),
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
    setMainImageIndex(0);
  }, []);

  const [imagePreviewUrls, setImagePreviewUrls] = React.useState<string[]>([]);

  React.useEffect(() => {
    if (images.length === 0) {
      setImagePreviewUrls([]);
      return;
    }

    const urls = images.map((file: File) => URL.createObjectURL(file));
    setImagePreviewUrls(urls);

    return () => {
      urls.forEach((url: string) => {
        try {
          URL.revokeObjectURL(url);
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
    imageInputRef,
    onSelectImages,
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,
  };
}