// frontend/amol/src/features/inquiry/hooks/useInquiryCreatePage.tsx
import {
  ChangeEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import type { MediaUploaderItem } from "../../../components/ui/MediaUploader";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { storage } from "../../../lib/firebase";

type CreateInquiryImage = {
  fileName: string;
  fileUrl: string;
  objectPath: string;
  fileSize: number;
  mimeType: string;
  createdAt: string;
};

type CreateInquiryRequest = {
  productId: string;
  subject: string;
  content: string;
  inquiryType: string;
  images: CreateInquiryImage[];
};

type CreateInquiryResponse = {
  data?: {
    id?: string;
    productId?: string;
    avatarId?: string;
    subject?: string;
    content?: string;
    status?: string;
    inquiryType?: string;
    createdAt?: string;
    updatedAt?: string;
  };
  error?: string;
};

export type InquiryMediaItem = MediaUploaderItem & {
  file: File;
};

const DEFAULT_INQUIRY_TYPE = "product";

function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

function createMediaItemId(file: File): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(fileName: string): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(/[^\w.\-()]/g, "_");
}

async function uploadInquiryImage(params: {
  productId: string;
  item: InquiryMediaItem;
}): Promise<CreateInquiryImage> {
  const imageId = createMediaItemId(params.item.file);
  const safeFileName = sanitizeStorageFileName(params.item.file.name);
  const objectPath = `inquiry-images/${params.productId}/${imageId}/${safeFileName}`;
  const storageRef = ref(storage, objectPath);
  const mimeType = params.item.file.type || "application/octet-stream";

  await uploadBytes(storageRef, params.item.file, {
    contentType: mimeType,
  });

  const fileUrl = await getDownloadURL(storageRef);

  return {
    fileName: params.item.file.name,
    fileUrl,
    objectPath,
    fileSize: params.item.file.size,
    mimeType,
    createdAt: new Date().toISOString(),
  };
}

export function useInquiryCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const carouselRef = useRef<HTMLDivElement>(null);
  const objectUrlSetRef = useRef<Set<string>>(new Set());

  const productId = useMemo(() => {
    return (searchParams.get("productId") ?? "").trim();
  }, [searchParams]);

  const backTo = useMemo(() => {
    if (!productId) {
      return "/scan/result";
    }

    return `/scan/result/${encodeURIComponent(productId)}`;
  }, [productId]);

  const [subject, setSubject] = useState("");
  const [content, setContent] = useState("");
  const [mediaItems, setMediaItems] = useState<InquiryMediaItem[]>([]);
  const [currentMediaIndex, setCurrentMediaIndex] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      objectUrlSetRef.current.forEach((url) => URL.revokeObjectURL(url));
      objectUrlSetRef.current.clear();
    };
  }, []);

  const canSubmit =
    Boolean(productId) &&
    Boolean(subject.trim()) &&
    Boolean(content.trim()) &&
    !submitting &&
    !submitted;

  const handleFilesSelected = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files ?? []).filter((file) =>
        file.type.startsWith("image/"),
      );

      if (files.length === 0) {
        event.target.value = "";
        return;
      }

      const nextItems = files.map((file): InquiryMediaItem => {
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

      setMediaItems((prev) => [...prev, ...nextItems]);
      event.target.value = "";
    },
    [],
  );

  const handleRemoveMediaItem = useCallback((id: string) => {
    setMediaItems((prev) => {
      const target = prev.find((item) => item.id === id);

      if (target?.previewUrl) {
        URL.revokeObjectURL(target.previewUrl);
        objectUrlSetRef.current.delete(target.previewUrl);
      }

      const next = prev.filter((item) => item.id !== id);

      setCurrentMediaIndex((current) => {
        if (next.length === 0) {
          return 0;
        }

        return Math.min(current, next.length - 1);
      });

      return next;
    });
  }, []);

  const handleCarouselScroll = useCallback(() => {
    const carousel = carouselRef.current;

    if (!carousel || carousel.clientWidth === 0) {
      return;
    }

    const nextIndex = Math.round(carousel.scrollLeft / carousel.clientWidth);
    setCurrentMediaIndex(Math.max(0, Math.min(nextIndex, mediaItems.length - 1)));
  }, [mediaItems.length]);

  const handleMoveToSlide = useCallback((index: number) => {
    const carousel = carouselRef.current;
    const target = carousel?.children.item(index);

    if (!target) {
      setCurrentMediaIndex(index);
      return;
    }

    target.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
      inline: "start",
    });

    setCurrentMediaIndex(index);
  }, []);

  const clearMediaItems = useCallback(() => {
    objectUrlSetRef.current.forEach((url) => URL.revokeObjectURL(url));
    objectUrlSetRef.current.clear();
    setMediaItems([]);
    setCurrentMediaIndex(0);
  }, []);

  const submitInquiry = useCallback(async () => {
    if (!canSubmit) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const token = await getFirebaseIdToken();

      const uploadedImages = await Promise.all(
        mediaItems.map((item) =>
          uploadInquiryImage({
            productId,
            item,
          }),
        ),
      );

      const payload: CreateInquiryRequest = {
        productId,
        subject: subject.trim(),
        content: content.trim(),
        inquiryType: DEFAULT_INQUIRY_TYPE,
        images: uploadedImages,
      };

      const url = buildApiUrl("/mall/me/inquiries");

      const res = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(payload),
      });

      const json = (await res.json().catch(() => ({}))) as CreateInquiryResponse;

      if (!res.ok) {
        throw new Error(json.error || "問い合わせの送信に失敗しました。");
      }

      setSubmitted(true);
      setSubject("");
      setContent("");
      clearMediaItems();
    } catch (e) {
      const message =
        e instanceof Error ? e.message : "問い合わせの送信に失敗しました。";
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }, [canSubmit, clearMediaItems, content, mediaItems, productId, subject]);

  const handleBackToScanResult = useCallback(() => {
    navigate(backTo);
  }, [backTo, navigate]);

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