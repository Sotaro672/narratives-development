// frontend/src/features/contact/hooks/useContactAttachments.ts
import { ChangeEvent, useRef, useState } from "react";

import type { ContactAttachmentItem } from "../types";

export function useContactAttachments() {
  const mediaInputRef = useRef<HTMLInputElement>(null);
  const carouselRef = useRef<HTMLDivElement>(null);

  const [carouselIndex, setCarouselIndex] = useState(0);
  const [attachments, setAttachments] = useState<ContactAttachmentItem[]>([]);

  const handleFilesSelected = (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    if (files.length === 0) {
      return;
    }

    const imageFiles = files.filter((file) => file.type.startsWith("image/"));

    if (imageFiles.length !== files.length) {
      window.alert("添付できるファイルは画像のみです。");
    }

    if (imageFiles.length === 0) {
      event.target.value = "";
      return;
    }

    const nextItems = imageFiles.map((file) => ({
      id: `${file.name}-${file.lastModified}-${Math.random()
        .toString(36)
        .slice(2)}`,
      type: "image" as const,
      previewUrl: URL.createObjectURL(file),
      fileName: file.name,
      title: file.name,
      file,
    }));

    setAttachments((prev) => [...prev, ...nextItems]);
    event.target.value = "";
  };

  const handleRemoveAttachment = (id: string) => {
    setAttachments((prev) => {
      const target = prev.find((item) => item.id === id);

      if (target?.previewUrl) {
        URL.revokeObjectURL(target.previewUrl);
      }

      return prev.filter((item) => item.id !== id);
    });
  };

  const handleCarouselScroll = () => {
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
  };

  const handleMoveToSlide = (index: number) => {
    const node = carouselRef.current;
    if (!node) {
      return;
    }

    node.scrollTo({
      left: node.clientWidth * index,
      behavior: "smooth",
    });

    setCarouselIndex(index);
  };

  const revokeAllAttachmentPreviewUrls = () => {
    attachments.forEach((item) => {
      if (item.previewUrl) {
        URL.revokeObjectURL(item.previewUrl);
      }
    });
  };

  return {
    mediaInputRef,
    carouselRef,
    carouselIndex,
    attachments,
    setAttachments,
    setCarouselIndex,
    handleFilesSelected,
    handleRemoveAttachment,
    handleCarouselScroll,
    handleMoveToSlide,
    revokeAllAttachmentPreviewUrls,
  };
}