// frontend/amol/src/features/inquiry/hooks/useInquiryCreatePage.tsx

import {
  type ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  useNavigate,
  useSearchParams,
} from "react-router-dom";

import type {
  MediaUploaderItem,
} from "../../../../components/ui/MediaUploader";

import {
  createInquiry,
  uploadInquiryImage,
  type CreateInquiryRequest,
} from "../../api/inquiryApi";

export type InquiryMediaItem =
  MediaUploaderItem & {
    file: File;
  };

const DEFAULT_INQUIRY_TYPE = "product";

function createMediaItemId(
  file: File,
): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  ) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

export function useInquiryCreatePage() {
  const navigate = useNavigate();
  const [searchParams] =
    useSearchParams();

  const fileInputRef =
    useRef<HTMLInputElement>(null);

  const carouselRef =
    useRef<HTMLDivElement>(null);

  const objectUrlSetRef =
    useRef<Set<string>>(
      new Set(),
    );

  const productId = useMemo(() => {
    return (
      searchParams.get(
        "productId",
      ) ?? ""
    ).trim();
  }, [searchParams]);

  const backTo = useMemo(() => {
    if (!productId) {
      return "/scan/result";
    }

    return `/scan/result/${encodeURIComponent(
      productId,
    )}`;
  }, [productId]);

  const [
    subject,
    setSubject,
  ] = useState("");

  const [
    content,
    setContent,
  ] = useState("");

  const [
    mediaItems,
    setMediaItems,
  ] = useState<
    InquiryMediaItem[]
  >([]);

  const [
    currentMediaIndex,
    setCurrentMediaIndex,
  ] = useState(0);

  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [
    submitted,
    setSubmitted,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<
    string | null
  >(null);

  useEffect(() => {
    return () => {
      objectUrlSetRef.current.forEach(
        (url) => {
          URL.revokeObjectURL(
            url,
          );
        },
      );

      objectUrlSetRef.current.clear();
    };
  }, []);

  const canSubmit =
    Boolean(productId) &&
    Boolean(subject.trim()) &&
    Boolean(content.trim()) &&
    !submitting &&
    !submitted;

  const handleFilesSelected =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const files = Array.from(
          event.target.files ?? [],
        ).filter((file) =>
          file.type.startsWith(
            "image/",
          ),
        );

        if (files.length === 0) {
          event.target.value = "";
          return;
        }

        const nextItems =
          files.map(
            (
              file,
            ): InquiryMediaItem => {
              const previewUrl =
                URL.createObjectURL(
                  file,
                );

              objectUrlSetRef.current.add(
                previewUrl,
              );

              return {
                id:
                  createMediaItemId(
                    file,
                  ),
                type: "image",
                previewUrl,
                title: file.name,
                fileName:
                  file.name,
                file,
              };
            },
          );

        setMediaItems(
          (previousItems) => [
            ...previousItems,
            ...nextItems,
          ],
        );

        event.target.value = "";
      },
      [],
    );

  const handleRemoveMediaItem =
    useCallback(
      (id: string) => {
        setMediaItems(
          (previousItems) => {
            const target =
              previousItems.find(
                (item) =>
                  item.id === id,
              );

            if (
              target?.previewUrl
            ) {
              URL.revokeObjectURL(
                target.previewUrl,
              );

              objectUrlSetRef.current.delete(
                target.previewUrl,
              );
            }

            const nextItems =
              previousItems.filter(
                (item) =>
                  item.id !== id,
              );

            setCurrentMediaIndex(
              (currentIndex) => {
                if (
                  nextItems.length ===
                  0
                ) {
                  return 0;
                }

                return Math.min(
                  currentIndex,
                  nextItems.length -
                    1,
                );
              },
            );

            return nextItems;
          },
        );
      },
      [],
    );

  const handleCarouselScroll =
    useCallback(() => {
      const carousel =
        carouselRef.current;

      if (
        !carousel ||
        carousel.clientWidth ===
          0
      ) {
        return;
      }

      const nextIndex =
        Math.round(
          carousel.scrollLeft /
            carousel.clientWidth,
        );

      setCurrentMediaIndex(
        Math.max(
          0,
          Math.min(
            nextIndex,
            mediaItems.length -
              1,
          ),
        ),
      );
    }, [mediaItems.length]);

  const handleMoveToSlide =
    useCallback(
      (index: number) => {
        const carousel =
          carouselRef.current;

        const target =
          carousel?.children.item(
            index,
          );

        if (!target) {
          setCurrentMediaIndex(
            index,
          );

          return;
        }

        target.scrollIntoView({
          behavior: "smooth",
          block: "nearest",
          inline: "start",
        });

        setCurrentMediaIndex(
          index,
        );
      },
      [],
    );

  const clearMediaItems =
    useCallback(() => {
      objectUrlSetRef.current.forEach(
        (url) => {
          URL.revokeObjectURL(
            url,
          );
        },
      );

      objectUrlSetRef.current.clear();

      setMediaItems([]);
      setCurrentMediaIndex(0);
    }, []);

  const submitInquiry =
    useCallback(async () => {
      if (!canSubmit) {
        return;
      }

      setSubmitting(true);
      setError(null);

      try {
        const uploadedImages =
          await Promise.all(
            mediaItems.map(
              (item) =>
                uploadInquiryImage(
                  {
                    productId,
                    file: item.file,
                  },
                ),
            ),
          );

        const payload:
          CreateInquiryRequest =
          {
            productId,
            subject:
              subject.trim(),
            content:
              content.trim(),
            inquiryType:
              DEFAULT_INQUIRY_TYPE,
            images:
              uploadedImages,
          };

        await createInquiry(
          payload,
        );

        setSubmitted(true);
        setSubject("");
        setContent("");
        clearMediaItems();
      } catch (caught) {
        const message =
          caught instanceof Error
            ? caught.message
            : "問い合わせの送信に失敗しました。";

        setError(message);
      } finally {
        setSubmitting(false);
      }
    }, [
      canSubmit,
      clearMediaItems,
      content,
      mediaItems,
      productId,
      subject,
    ]);

  const handleBackToScanResult =
    useCallback(() => {
      navigate(backTo);
    }, [
      backTo,
      navigate,
    ]);

  return {
    navigate,
    productId,
    backTo,

    subject,
    setSubject,
    content,
    setContent,
    mediaItems,
    currentMediaIndex,
    fileInputRef,
    carouselRef,

    submitting,
    submitted,
    error,
    canSubmit,

    submitInquiry,
    handleFilesSelected,
    handleRemoveMediaItem,
    handleCarouselScroll,
    handleMoveToSlide,
    handleBackToScanResult,
  };
}