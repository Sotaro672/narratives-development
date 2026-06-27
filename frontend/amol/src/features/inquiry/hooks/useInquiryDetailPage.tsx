// frontend/amol/src/features/inquiry/hooks/useInquiryDetailPage.tsx
import {
  ChangeEvent,
  MutableRefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { MediaUploaderItem } from "../../../components/ui/MediaUploader";
import {
  closeInquiry,
  getInquiry,
  listInquiryReplies,
  markInquiryAsRead,
  replyInquiry,
  uploadReplyImage,
  type Inquiry,
  type InquiryReply,
  type ReplyInquiryRequest,
} from "../api/inquiryApi";
import { useInquiryUnreadCounter } from "./useInquiryUnreadCounter";

export type InquiryMediaItem = MediaUploaderItem & {
  file: File;
};

function createMediaItemId(file: File): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

function createInquiryMediaItems(
  files: File[],
  objectUrlSetRef: MutableRefObject<Set<string>>,
): InquiryMediaItem[] {
  return files.map((file): InquiryMediaItem => {
    const previewUrl = URL.createObjectURL(file);
    objectUrlSetRef.current.add(previewUrl);

    return {
      id: createMediaItemId(file),
      type: "image",
      previewUrl,
      title: file.name,
      fileName: file.name,
      file,
    };
  });
}

function textOrDash(value: string | null | undefined): string {
  const text = String(value ?? "").trim();
  return text || "-";
}

function statusLabel(value: string | null | undefined): string {
  const status = String(value ?? "").trim();

  switch (status) {
    case "open":
      return "未対応";
    case "in_progress":
      return "対応中";
    case "resolved":
      return "対応済み";
    case "closed":
      return "クローズ";
    default:
      return status || "-";
  }
}

function typeLabel(value: string | null | undefined): string {
  const inquiryType = String(value ?? "").trim();

  switch (inquiryType) {
    case "product":
      return "商品";
    case "product_description":
      return "商品説明";
    case "exchange":
      return "交換";
    case "shipping":
      return "配送";
    case "payment":
      return "決済";
    case "other":
      return "その他";
    default:
      return inquiryType || "-";
  }
}

function formatDateTime(value: string | null | undefined): string {
  if (!value) return "-";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function useInquiryDetailPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const replyFileInputRef = useRef<HTMLInputElement>(null);
  const replyCarouselRef = useRef<HTMLDivElement>(null);
  const objectUrlSetRef = useRef<Set<string>>(new Set());

  const productId = useMemo(() => {
    return (searchParams.get("productId") ?? "").trim();
  }, [searchParams]);

  const inquiryId = useMemo(() => {
    return (searchParams.get("inquiryId") ?? "").trim();
  }, [searchParams]);

  const backTo = useMemo(() => {
    if (productId) {
      return `/scan/result/${encodeURIComponent(productId)}`;
    }

    return "/scan/result";
  }, [productId]);

  const {
    unreadCount,
    loading: unreadCountLoading,
    error: unreadCountError,
    loadUnreadCount,
    clearUnreadCount,
  } = useInquiryUnreadCounter({
    enabled: Boolean(inquiryId),
  });

  const [inquiry, setInquiry] = useState<Inquiry | null>(null);
  const [replies, setReplies] = useState<InquiryReply[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [replyContent, setReplyContent] = useState("");
  const [replyMediaItems, setReplyMediaItems] = useState<InquiryMediaItem[]>(
    [],
  );
  const [replyCurrentMediaIndex, setReplyCurrentMediaIndex] = useState(0);
  const [replySubmitting, setReplySubmitting] = useState(false);
  const [markingAsRead, setMarkingAsRead] = useState(false);
  const [closing, setClosing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      objectUrlSetRef.current.forEach((url) => URL.revokeObjectURL(url));
      objectUrlSetRef.current.clear();
    };
  }, []);

  const canReply =
    Boolean(inquiryId) &&
    (Boolean(replyContent.trim()) || replyMediaItems.length > 0) &&
    !replySubmitting &&
    inquiry?.status !== "closed";

  const clearReplyMediaItems = useCallback(() => {
    setReplyMediaItems((prev) => {
      prev.forEach((item) => {
        if (item.previewUrl) {
          URL.revokeObjectURL(item.previewUrl);
          objectUrlSetRef.current.delete(item.previewUrl);
        }
      });

      return [];
    });

    setReplyCurrentMediaIndex(0);
  }, []);

  const loadInquiry = useCallback(async () => {
    if (!inquiryId) {
      setInquiry(null);
      setReplies([]);
      return;
    }

    setDetailLoading(true);
    setError(null);

    try {
      const [detail, replyItems] = await Promise.all([
        getInquiry(inquiryId),
        listInquiryReplies(inquiryId),
      ]);

      setInquiry(detail);
      setReplies(replyItems);
    } catch (e) {
      setError(getErrorMessage(e, "問い合わせ詳細の取得に失敗しました。"));
      setInquiry(null);
      setReplies([]);
    } finally {
      setDetailLoading(false);
    }
  }, [inquiryId]);

  useEffect(() => {
    if (!inquiryId) {
      clearUnreadCount();
      setInquiry(null);
      setReplies([]);
      return;
    }

    void loadInquiry();
    void loadUnreadCount();
  }, [clearUnreadCount, inquiryId, loadInquiry, loadUnreadCount]);

  const markAsRead = useCallback(async () => {
    if (!inquiryId) {
      return;
    }

    setMarkingAsRead(true);
    setError(null);

    try {
      const updated = await markInquiryAsRead(inquiryId);

      if (updated) {
        setInquiry(updated);
      }

      await loadInquiry();
      await loadUnreadCount();
    } catch (e) {
      setError(getErrorMessage(e, "既読化に失敗しました。"));
    } finally {
      setMarkingAsRead(false);
    }
  }, [inquiryId, loadInquiry, loadUnreadCount]);

  const closeCurrentInquiry = useCallback(async () => {
    if (!inquiryId) {
      return;
    }

    setClosing(true);
    setError(null);

    try {
      const updated = await closeInquiry(inquiryId);

      if (updated) {
        setInquiry(updated);
      }

      await loadInquiry();
      await loadUnreadCount();
    } catch (e) {
      setError(getErrorMessage(e, "問い合わせのクローズに失敗しました。"));
    } finally {
      setClosing(false);
    }
  }, [inquiryId, loadInquiry, loadUnreadCount]);

  const handleReplyFilesSelected = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []).filter((file) =>
        file.type.startsWith("image/"),
      );

      if (files.length === 0) {
        event.target.value = "";
        return;
      }

      const nextItems = createInquiryMediaItems(files, objectUrlSetRef);

      setReplyMediaItems((prev) => [...prev, ...nextItems]);
      event.target.value = "";
    },
    [],
  );

  const handleRemoveReplyMediaItem = useCallback((id: string) => {
    setReplyMediaItems((prev) => {
      const target = prev.find((item) => item.id === id);

      if (target?.previewUrl) {
        URL.revokeObjectURL(target.previewUrl);
        objectUrlSetRef.current.delete(target.previewUrl);
      }

      const next = prev.filter((item) => item.id !== id);

      setReplyCurrentMediaIndex((current) => {
        if (next.length === 0) {
          return 0;
        }

        return Math.min(current, next.length - 1);
      });

      return next;
    });
  }, []);

  const handleReplyCarouselScroll = useCallback(() => {
    const carousel = replyCarouselRef.current;

    if (!carousel || carousel.clientWidth === 0) {
      return;
    }

    const nextIndex = Math.round(carousel.scrollLeft / carousel.clientWidth);
    setReplyCurrentMediaIndex(
      Math.max(0, Math.min(nextIndex, replyMediaItems.length - 1)),
    );
  }, [replyMediaItems.length]);

  const handleMoveReplyToSlide = useCallback((index: number) => {
    const carousel = replyCarouselRef.current;
    const target = carousel?.children.item(index);

    if (!target) {
      setReplyCurrentMediaIndex(index);
      return;
    }

    target.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
      inline: "start",
    });

    setReplyCurrentMediaIndex(index);
  }, []);

  const submitReply = useCallback(async () => {
    if (!canReply || !inquiryId) {
      return;
    }

    setReplySubmitting(true);
    setError(null);

    try {
      const uploadedImages = await Promise.all(
        replyMediaItems.map((item) =>
          uploadReplyImage({
            inquiryId,
            file: item.file,
          }),
        ),
      );

      const payload: ReplyInquiryRequest = {
        content: replyContent.trim(),
        images: uploadedImages,
      };

      await replyInquiry(inquiryId, payload);

      setReplyContent("");
      clearReplyMediaItems();
      await loadInquiry();
      await loadUnreadCount();
    } catch (e) {
      setError(getErrorMessage(e, "返信の送信に失敗しました。"));
    } finally {
      setReplySubmitting(false);
    }
  }, [
    canReply,
    clearReplyMediaItems,
    inquiryId,
    loadInquiry,
    loadUnreadCount,
    replyContent,
    replyMediaItems,
  ]);

  const navigateToScanResult = useCallback(() => {
    navigate(backTo);
  }, [backTo, navigate]);

  return {
    navigate,
    productId,
    inquiryId,
    backTo,

    inquiry,
    replies,
    unreadCount,
    unreadCountLoading,
    detailLoading,

    replyContent,
    setReplyContent,
    replyMediaItems,
    replyCurrentMediaIndex,
    replyFileInputRef,
    replyCarouselRef,

    replySubmitting,
    markingAsRead,
    closing,
    canReply,

    error: error ?? unreadCountError?.message ?? null,

    textOrDash,
    statusLabel,
    typeLabel,
    formatDateTime,

    navigateToScanResult,
    submitReply,
    markAsRead,
    closeInquiry: closeCurrentInquiry,
    loadInquiry,
    loadUnreadCount,

    handleReplyFilesSelected,
    handleRemoveReplyMediaItem,
    handleReplyCarouselScroll,
    handleMoveReplyToSlide,
  };
}