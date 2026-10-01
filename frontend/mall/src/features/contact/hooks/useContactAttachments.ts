// frontend/mall/src/features/contact/hooks/useContactAttachments.ts

import {
  type ChangeEvent,
  useCallback,
  useRef,
  useState,
} from "react";

import type { ContactAttachmentItem } from "../../shared/types/contact";

function createAttachmentItem(file: File): ContactAttachmentItem {
  return {
    id: `${file.name}-${file.lastModified}-${Math.random().toString(36).slice(2)}`,
    type: "image",
    previewUrl: URL.createObjectURL(file),
    fileName: file.name,
    title: file.name,
    file,
  };
}

export function useContactAttachments() {
  const mediaInputRef = useRef<HTMLInputElement>(null);
  const carouselRef = useRef<HTMLDivElement>(null);

  const [carouselIndex, setCarouselIndex] = useState(0);
  const [attachments, setAttachments] = useState<ContactAttachmentItem[]>([]);

  const addImageFiles = useCallback((files: File[]) => {
    const imageFiles = files.filter((file) => file.type.startsWith("image/"));

    if (imageFiles.length === 0) {
      return;
    }

    const nextItems = imageFiles.map(createAttachmentItem);
    setAttachments((currentAttachments) => [
      ...currentAttachments,
      ...nextItems,
    ]);
  }, []);

  const removeAttachmentAtIndex = useCallback((index: number) => {
    setAttachments((currentAttachments) => {
      if (index < 0 || index >= currentAttachments.length) {
        return currentAttachments;
      }

      const target = currentAttachments[index];

      if (target.previewUrl) {
        URL.revokeObjectURL(target.previewUrl);
      }

      return currentAttachments.filter(
        (_attachment, currentIndex) => currentIndex !== index,
      );
    });
  }, []);

  const handleFilesSelected = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.currentTarget.files ?? []);
      event.currentTarget.value = "";

      if (files.length === 0) {
        return;
      }

      const imageFiles = files.filter((file) => file.type.startsWith("image/"));

      if (imageFiles.length !== files.length) {
        window.alert("添付できるファイルは画像のみです。");
      }

      addImageFiles(imageFiles);
    },
    [addImageFiles],
  );

  const handleFilesAdd = useCallback(
    (files: File[]) => {
      addImageFiles(files);
    },
    [addImageFiles],
  );

  const handleRemoveAttachment = useCallback((id: string) => {
    setAttachments((currentAttachments) => {
      const target = currentAttachments.find(
        (attachment) => attachment.id === id,
      );

      if (target?.previewUrl) {
        URL.revokeObjectURL(target.previewUrl);
      }

      return currentAttachments.filter(
        (attachment) => attachment.id !== id,
      );
    });
  }, []);

  const handleRemoveFile = useCallback(
    (index: number) => {
      removeAttachmentAtIndex(index);
    },
    [removeAttachmentAtIndex],
  );

  const handleCarouselScroll = useCallback(() => {
    const node = carouselRef.current;

    if (!node) {
      return;
    }

    const cardWidth = node.clientWidth;

    if (cardWidth <= 0) {
      return;
    }

    const nextIndex = Math.round(node.scrollLeft / cardWidth);
    setCarouselIndex(nextIndex);
  }, []);

  const handleMoveToSlide = useCallback((index: number) => {
    const node = carouselRef.current;

    if (!node) {
      return;
    }

    node.scrollTo({
      left: node.clientWidth * index,
      behavior: "smooth",
    });

    setCarouselIndex(index);
  }, []);

  const revokeAllAttachmentPreviewUrls = useCallback(() => {
    attachments.forEach((attachment) => {
      if (attachment.previewUrl) {
        URL.revokeObjectURL(attachment.previewUrl);
      }
    });
  }, [attachments]);

  return {
    mediaInputRef,
    carouselRef,
    carouselIndex,
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesSelected,
    handleFilesAdd,
    handleRemoveAttachment,
    handleRemoveFile,
    handleCarouselScroll,
    handleMoveToSlide,
    revokeAllAttachmentPreviewUrls,
  };
}