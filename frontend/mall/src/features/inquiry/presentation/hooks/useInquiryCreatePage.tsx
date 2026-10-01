// frontend/mall/src/features/inquiry/presentation/hooks/useInquiryCreatePage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import {
  createScanResultPageViewModel,
  type ScanResultPageViewModel,
} from "../../../scan-result/application/scanPageViewModelFactory";
import { loadPreviewState } from "../../../scan-result/infrastructure/scanResultApi";
import {
  createInquiry,
  uploadInquiryImage,
  type CreateInquiryRequest,
} from "../../api/inquiryApi";

const PRODUCT_INQUIRY_TYPE = "product" as const;
const MAX_FILES = 10;
const PRIVACY_POLICY_PATH = "/assets/privacy-policy.txt";

export function useInquiryCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const productId = useMemo(() => {
    return (searchParams.get("productId") ?? "").trim();
  }, [searchParams]);

  const backTo = useMemo(() => {
    if (!productId) {
      return "/scan/result";
    }

    return `/scan/result/${encodeURIComponent(productId)}`;
  }, [productId]);

  const [content, setContent] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [productViewModel, setProductViewModel] = useState<ScanResultPageViewModel | null>(null);
  const [productLoading, setProductLoading] = useState(false);
  const [productError, setProductError] = useState<string | null>(null);

  const [privacyPolicy, setPrivacyPolicy] = useState("");
  const [privacyLoading, setPrivacyLoading] = useState(true);
  const [privacyError, setPrivacyError] = useState<string | null>(null);
  const [agreedToPrivacyPolicy, setAgreedToPrivacyPolicy] = useState(false);

  useEffect(() => {
    let cancelled = false;

    if (!productId) {
      setProductViewModel(null);
      setProductLoading(false);
      setProductError(null);
      return () => {
        cancelled = true;
      };
    }

    async function loadProduct() {
      setProductLoading(true);
      setProductError(null);
      setProductViewModel(null);

      try {
        const previewState = await loadPreviewState(productId);

        if (cancelled) {
          return;
        }

        const viewModel = createScanResultPageViewModel({
          previewState,
          ownedByWallet: null,
        });

        if (!viewModel) {
          throw new Error("商品情報を取得できませんでした。");
        }

        setProductViewModel(viewModel);
      } catch (caught) {
        if (cancelled) {
          return;
        }

        const message =
          caught instanceof Error
            ? caught.message
            : "商品情報の読み込みに失敗しました。";

        setProductError(message);
      } finally {
        if (!cancelled) {
          setProductLoading(false);
        }
      }
    }

    void loadProduct();

    return () => {
      cancelled = true;
    };
  }, [productId]);

  useEffect(() => {
    let cancelled = false;

    async function loadPrivacyPolicy() {
      setPrivacyLoading(true);
      setPrivacyError(null);

      try {
        const response = await fetch(PRIVACY_POLICY_PATH);

        if (!response.ok) {
          throw new Error("プライバシーポリシーの読み込みに失敗しました。");
        }

        const policy = await response.text();

        if (cancelled) {
          return;
        }

        setPrivacyPolicy(policy);
      } catch (caught) {
        if (cancelled) {
          return;
        }

        const message =
          caught instanceof Error
            ? caught.message
            : "プライバシーポリシーの読み込みに失敗しました。";

        setPrivacyPolicy("");
        setPrivacyError(message);
      } finally {
        if (!cancelled) {
          setPrivacyLoading(false);
        }
      }
    }

    void loadPrivacyPolicy();

    return () => {
      cancelled = true;
    };
  }, []);

  const canSubmit =
    Boolean(productId) &&
    Boolean(productViewModel) &&
    agreedToPrivacyPolicy &&
    Boolean(content.trim()) &&
    !submitting;

  const handleFilesAdd = useCallback((nextFiles: File[]) => {
    const imageFiles = nextFiles.filter((file) => file.type.startsWith("image/"));

    if (imageFiles.length === 0) {
      return;
    }

    setFiles((previousFiles) => {
      const availableCount = Math.max(0, MAX_FILES - previousFiles.length);

      if (availableCount === 0) {
        return previousFiles;
      }

      return [...previousFiles, ...imageFiles.slice(0, availableCount)];
    });
  }, []);

  const handleRemoveFile = useCallback((index: number) => {
    setFiles((previousFiles) => {
      if (index < 0 || index >= previousFiles.length) {
        return previousFiles;
      }

      return previousFiles.filter((_, fileIndex) => fileIndex !== index);
    });
  }, []);

  const submitInquiry = useCallback(async () => {
    if (!canSubmit) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const uploadedImages = await Promise.all(
        files.map((file) =>
          uploadInquiryImage({
            productId,
            file,
          }),
        ),
      );

      const payload: CreateInquiryRequest = {
        productId,
        content: content.trim(),
        inquiryType: PRODUCT_INQUIRY_TYPE,
        images: uploadedImages,
      };

      const createdInquiry = await createInquiry(payload);

      setContent("");
      setFiles([]);

      navigate(`/chats/${encodeURIComponent(createdInquiry.id)}`, {
        replace: true,
      });
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
    content,
    files,
    navigate,
    productId,
  ]);

  const handleBackToScanResult = useCallback(() => {
    navigate(backTo);
  }, [backTo, navigate]);

  return {
    navigate,
    productId,
    backTo,

    productViewModel,
    productLoading,
    productError,

    privacyPolicy,
    privacyLoading,
    privacyError,
    agreedToPrivacyPolicy,
    setAgreedToPrivacyPolicy,

    content,
    setContent,
    files,

    submitting,
    error,
    canSubmit,

    submitInquiry,
    handleFilesAdd,
    handleRemoveFile,
    handleBackToScanResult,
  };
}