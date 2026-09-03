// frontend/console/shell/src/features/brand/presentation/hook/useBrandDetail.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { Account } from "../../../../shared/types/account";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import type { Brand, BrandPatch } from "../../../../shared/types/brand";

import { accountRepositoryHTTP } from "../../../account/infrastructure/http/accountRepositoryHTTP";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import { validateBrandImage } from "../../application/brandImageValidation";
import {
  BRAND_IMAGE_ALLOWED_MIME_TYPES,
  type BrandImageTarget,
} from "../../config/brandImagePolicy.generated";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";
import { uploadBrandAssetToFirebaseStorage } from "../../infrastructure/storage/brandAssetStorage";
import {
  createCompletedBrandCreateProgress,
  createCreatingBrandCreateProgress,
  createFailedBrandCreateProgress,
  createInitialBrandCreateProgress,
  createSavingBrandCreateProgress,
  createUploadingBrandCreateProgress,
  isBrandCreateProgressVisible,
  type BrandCreateProgress,
} from "../model/brandCreateProgress";

const BRAND_IMAGE_ACCEPT = BRAND_IMAGE_ALLOWED_MIME_TYPES.join(",");

export type BrandAccountCandidate = {
  id: string;
  label: string;
  status: Account["status"];
};

type BrandDraft = {
  name: string;
  description: string;
  websiteUrl: string;
  brandIcon: string;
  brandBackgroundImage: string;
  isActive: boolean;
};

type UploadBrandAssetsResult = {
  uploadedBrandIcon: string;
  uploadedBrandBackgroundImage: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

function createEmptyBrand(brandId: string): Brand {
  return {
    id: brandId,
    companyId: "",
    accountId: "",
    name: "",
    description: "",
    websiteUrl: "",
    brandIcon: "",
    brandBackgroundImage: "",
    isActive: false,
    managerId: null,
    memberName: "",
    walletAddress: "",
    createdAt: "",
    createdBy: null,
    updatedAt: null,
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
  };
}

function createDraft(brand: Brand): BrandDraft {
  return {
    name: brand.name,
    description: brand.description,
    websiteUrl: brand.websiteUrl ?? "",
    brandIcon: brand.brandIcon ?? "",
    brandBackgroundImage: brand.brandBackgroundImage ?? "",
    isActive: brand.isActive,
  };
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function buildAccountLabel(account: Account): string {
  const bankName = String(account.bankName ?? "").trim();
  const branchName = String(account.branchName ?? "").trim();
  const accountNumber = Number(account.accountNumber ?? 0);
  const stripeAccountId = String(account.stripeAccountId ?? "").trim();
  const bankLabel = [bankName, branchName].filter(Boolean).join(" ");
  const numberLabel = accountNumber > 0 ? String(accountNumber) : "";

  if (bankLabel && numberLabel) return `${bankLabel} ${numberLabel}`;
  if (bankLabel) return bankLabel;
  if (stripeAccountId) return stripeAccountId;
  return account.id;
}

export function useBrandDetail() {
  const navigate = useNavigate();
  const { brandId } = useParams<{ brandId: string }>();
  const resolvedBrandId = brandId ?? "";

  const [brand, setBrand] = useState<Brand>(() =>
    createEmptyBrand(resolvedBrandId),
  );
  const [draft, setDraft] = useState<BrandDraft>(() =>
    createDraft(createEmptyBrand(resolvedBrandId)),
  );

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  const [progress, setProgress] = useState<BrandCreateProgress>(
    createInitialBrandCreateProgress,
  );

  const [accountId, setAccountId] = useState("");
  const [accountCandidates, setAccountCandidates] = useState<BrandAccountCandidate[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(false);
  const [accountError, setAccountError] = useState<string | null>(null);

  const {
    assigneeId: managerId,
    assigneeName: editingManagerName,
    assigneeCandidates: managerCandidates,
    loadingMembers,
    handleSelectAssignee: handleSelectManager,
    resetAssignee,
  } = useAssigneeSelection({
    initialAssigneeId: brand.managerId,
    initialAssigneeName: brand.memberName,
    defaultToCurrentMember: false,
  });

  const [brandIconError, setBrandIconError] = useState<string | null>(null);
  const [brandBackgroundImageError, setBrandBackgroundImageError] =
    useState<string | null>(null);

  const brandIconInputRef = useRef<HTMLInputElement | null>(null);
  const brandBackgroundInputRef = useRef<HTMLInputElement | null>(null);

  const [brandIconFile, setBrandIconFile] = useState<File | null>(null);
  const [brandBackgroundFile, setBrandBackgroundFile] = useState<File | null>(null);

  const [brandIconPreviewUrl, setBrandIconPreviewUrl] = useState("");
  const [brandBackgroundPreviewUrl, setBrandBackgroundPreviewUrl] = useState("");

  const isUploading = progress.phase === "uploading" && saving;
  const progressOpen = isBrandCreateProgressVisible(progress);

  useEffect(() => {
    let cancelled = false;

    const loadBrand = async () => {
      if (!resolvedBrandId) return;

      try {
        setLoading(true);
        setError(null);

        const response = await brandRepositoryHTTP.getById(resolvedBrandId);
        if (cancelled) return;

        setBrand(response);
        setDraft(createDraft(response));
        setAccountId(String(response.accountId ?? ""));
        setBrandIconFile(null);
        setBrandBackgroundFile(null);
        setBrandIconError(null);
        setBrandBackgroundImageError(null);
      } catch (error: unknown) {
        if (!cancelled) {
          setError(
            error instanceof Error
              ? error
              : new Error(String(error)),
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void loadBrand();

    return () => {
      cancelled = true;
    };
  }, [resolvedBrandId]);

  useEffect(() => {
    let cancelled = false;

    const loadAccounts = async () => {
      try {
        setLoadingAccounts(true);
        setAccountError(null);

        const accounts = await accountRepositoryHTTP.list();
        if (cancelled) return;

        const candidates = accounts
          .filter((account) => account.status !== "deleted")
          .map(
            (account): BrandAccountCandidate => ({
              id: account.id,
              label: buildAccountLabel(account),
              status: account.status,
            }),
          );

        setAccountCandidates(candidates);
      } catch (error: unknown) {
        if (cancelled) return;

        setAccountCandidates([]);
        setAccountError(getErrorMessage(error));
      } finally {
        if (!cancelled) setLoadingAccounts(false);
      }
    };

    void loadAccounts();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!brandIconFile) {
      setBrandIconPreviewUrl(
        isEditing
          ? draft.brandIcon
          : brand.brandIcon ?? "",
      );
      return;
    }

    const objectUrl = URL.createObjectURL(brandIconFile);
    setBrandIconPreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [
    brandIconFile,
    draft.brandIcon,
    brand.brandIcon,
    isEditing,
  ]);

  useEffect(() => {
    if (!brandBackgroundFile) {
      setBrandBackgroundPreviewUrl(
        isEditing
          ? draft.brandBackgroundImage
          : brand.brandBackgroundImage ?? "",
      );
      return;
    }

    const objectUrl = URL.createObjectURL(brandBackgroundFile);
    setBrandBackgroundPreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [
    brandBackgroundFile,
    draft.brandBackgroundImage,
    brand.brandBackgroundImage,
    isEditing,
  ]);

  useEffect(() => {
    if (!isUploading) return;

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [isUploading]);

  const registeredAt = useMemo(
    () => safeDateTimeLabelJa(brand.createdAt, ""),
    [brand.createdAt],
  );

  const updatedAt = useMemo(
    () => safeDateTimeLabelJa(brand.updatedAt ?? "", ""),
    [brand.updatedAt],
  );

  const statusLabel = brand.isActive
    ? "アクティブ"
    : "停止";

  const handleBack = useCallback(() => {
    if (saving) return;
    navigate("/brand");
  }, [navigate, saving]);

  const handleOpenAccountConnect = useCallback(() => {
    if (saving) return;
    navigate("/account/connect");
  }, [navigate, saving]);

  const handleSelectAccount = useCallback((id: string) => {
    setAccountId(id);
    if (id) setAccountError(null);
  }, []);

  const handleEdit = useCallback(() => {
    setDraft(createDraft(brand));
    setAccountId(String(brand.accountId ?? ""));
    setBrandIconFile(null);
    setBrandBackgroundFile(null);
    setBrandIconError(null);
    setBrandBackgroundImageError(null);
    setAccountError(null);
    setError(null);
    setProgress(createInitialBrandCreateProgress());

    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }

    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }

    setIsEditing(true);
  }, [brand]);

  const handleCancelEdit = useCallback(() => {
    if (saving) return;

    setDraft(createDraft(brand));
    setAccountId(String(brand.accountId ?? ""));
    resetAssignee();
    setBrandIconFile(null);
    setBrandBackgroundFile(null);
    setBrandIconError(null);
    setBrandBackgroundImageError(null);
    setAccountError(null);
    setError(null);
    setProgress(createInitialBrandCreateProgress());

    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }

    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }

    setIsEditing(false);
  }, [
    brand,
    resetAssignee,
    saving,
  ]);

  const handlePickBrandIcon = useCallback(() => {
    if (!isEditing || saving) return;
    brandIconInputRef.current?.click();
  }, [
    isEditing,
    saving,
  ]);

  const handlePickBrandBackground = useCallback(() => {
    if (!isEditing || saving) return;
    brandBackgroundInputRef.current?.click();
  }, [
    isEditing,
    saving,
  ]);

  const validateSelectedImage = useCallback(
    (
      file: File,
      target: BrandImageTarget,
    ): string | null => {
      const validation = validateBrandImage(file, target);
      if (!validation.valid) return validation.message;
      return null;
    },
    [],
  );

  const handleBrandIconChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.currentTarget.files?.[0] ?? null;
      if (!file) return;

      const validationError = validateSelectedImage(
        file,
        "brandIcon",
      );

      if (validationError) {
        setBrandIconFile(null);
        setBrandIconError(validationError);
        event.currentTarget.value = "";
        alert(validationError);
        return;
      }

      setBrandIconFile(file);
      setBrandIconError(null);
    },
    [validateSelectedImage],
  );

  const handleBrandBackgroundChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.currentTarget.files?.[0] ?? null;
      if (!file) return;

      const validationError = validateSelectedImage(
        file,
        "brandBackgroundImage",
      );

      if (validationError) {
        setBrandBackgroundFile(null);
        setBrandBackgroundImageError(validationError);
        event.currentTarget.value = "";
        alert(validationError);
        return;
      }

      setBrandBackgroundFile(file);
      setBrandBackgroundImageError(null);
    },
    [validateSelectedImage],
  );

  const handleClearBrandIcon = useCallback(() => {
    setBrandIconFile(null);
    setBrandIconError(null);

    setDraft((currentDraft) => ({
      ...currentDraft,
      brandIcon: "",
    }));

    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }
  }, []);

  const handleClearBrandBackground = useCallback(() => {
    setBrandBackgroundFile(null);
    setBrandBackgroundImageError(null);

    setDraft((currentDraft) => ({
      ...currentDraft,
      brandBackgroundImage: "",
    }));

    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }
  }, []);

  const validateSelectedImagesBeforeSave =
    useCallback((): boolean => {
      if (brandIconFile) {
        const validationError = validateSelectedImage(
          brandIconFile,
          "brandIcon",
        );

        if (validationError) {
          setBrandIconError(validationError);
          alert(validationError);
          return false;
        }
      }

      if (brandBackgroundFile) {
        const validationError = validateSelectedImage(
          brandBackgroundFile,
          "brandBackgroundImage",
        );

        if (validationError) {
          setBrandBackgroundImageError(validationError);
          alert(validationError);
          return false;
        }
      }

      setBrandIconError(null);
      setBrandBackgroundImageError(null);
      return true;
    }, [
      brandIconFile,
      brandBackgroundFile,
      validateSelectedImage,
    ]);

  const uploadBrandAssets = useCallback(
    async (): Promise<UploadBrandAssetsResult> => {
      if (!resolvedBrandId) {
        throw new Error("brandId が取得できません。");
      }

      if (!brand.companyId) {
        throw new Error("companyId が取得できません。");
      }

      let uploadedBrandIcon = draft.brandIcon;
      let uploadedBrandBackgroundImage =
        draft.brandBackgroundImage;

      let completedBytes = 0;
      let completedUploadCount = 0;

      const totalBytes =
        (brandIconFile?.size ?? 0) +
        (brandBackgroundFile?.size ?? 0);

      const expectedUploadCount =
        (brandIconFile ? 1 : 0) +
        (brandBackgroundFile ? 1 : 0);

      if (brandIconFile) {
        const currentFile = brandIconFile;

        const uploaded =
          await uploadBrandAssetToFirebaseStorage({
            companyId: brand.companyId,
            brandId: resolvedBrandId,
            target: "brandIcon",
            file: currentFile,
            onProgress: (uploadProgress) => {
              setProgress(
                createUploadingBrandCreateProgress({
                  fileName: currentFile.name,
                  transferredBytes:
                    completedBytes +
                    uploadProgress.transferredBytes,
                  totalBytes,
                  completedUploadCount,
                  expectedUploadCount,
                  title: "ブランド画像を転送中",
                  message:
                    "ブランド画像をアップロードしています。画像転送が完了するまで、この画面を閉じたり移動したりしないでください。",
                }),
              );
            },
          });

        uploadedBrandIcon = uploaded.downloadUrl;
        completedBytes += currentFile.size;
        completedUploadCount += 1;
      }

      if (brandBackgroundFile) {
        const currentFile = brandBackgroundFile;

        const uploaded =
          await uploadBrandAssetToFirebaseStorage({
            companyId: brand.companyId,
            brandId: resolvedBrandId,
            target: "brandBackgroundImage",
            file: currentFile,
            onProgress: (uploadProgress) => {
              setProgress(
                createUploadingBrandCreateProgress({
                  fileName: currentFile.name,
                  transferredBytes:
                    completedBytes +
                    uploadProgress.transferredBytes,
                  totalBytes,
                  completedUploadCount,
                  expectedUploadCount,
                  title: "ブランド画像を転送中",
                  message:
                    "ブランド画像をアップロードしています。画像転送が完了するまで、この画面を閉じたり移動したりしないでください。",
                }),
              );
            },
          });

        uploadedBrandBackgroundImage =
          uploaded.downloadUrl;

        completedBytes += currentFile.size;
        completedUploadCount += 1;
      }

      return {
        uploadedBrandIcon,
        uploadedBrandBackgroundImage,
        transferredBytes: completedBytes,
        totalBytes,
        completedUploadCount,
        expectedUploadCount,
      };
    },
    [
      resolvedBrandId,
      brand.companyId,
      draft.brandIcon,
      draft.brandBackgroundImage,
      brandIconFile,
      brandBackgroundFile,
    ],
  );

  const handleSave = useCallback(async () => {
    if (!resolvedBrandId || saving) return;

    if (!draft.name) {
      setError(new Error("ブランド名は必須です。"));
      return;
    }

    if (!managerId) {
      setError(new Error("ブランド責任者は必須です。"));
      return;
    }

    if (!accountId) {
      setAccountError("売上受取口座は必須です。");
      setError(new Error("売上受取口座は必須です。"));
      return;
    }

    if (!validateSelectedImagesBeforeSave()) return;

    try {
      setSaving(true);
      setError(null);
      setAccountError(null);

      setProgress(
        createCreatingBrandCreateProgress({
          title: "ブランド情報を更新中",
          message: "ブランド情報の更新準備をしています。",
        }),
      );

      const {
        uploadedBrandIcon,
        uploadedBrandBackgroundImage,
        transferredBytes,
        totalBytes,
        completedUploadCount,
        expectedUploadCount,
      } = await uploadBrandAssets();

      setProgress(
        createSavingBrandCreateProgress({
          transferredBytes,
          totalBytes,
          completedUploadCount,
          expectedUploadCount,
          title: "ブランド情報を保存中",
          message:
            expectedUploadCount > 0
              ? "画像転送が完了しました。ブランド情報を保存しています。"
              : "ブランド情報を保存しています。",
        }),
      );

      const patch: BrandPatch = {
        accountId,
        name: draft.name,
        description: draft.description,
        websiteUrl: draft.websiteUrl,
        brandIcon: uploadedBrandIcon,
        brandBackgroundImage:
          uploadedBrandBackgroundImage,
        isActive: draft.isActive,
        managerId,
      };

      const savedBrand =
        await brandRepositoryHTTP.update(
          resolvedBrandId,
          patch,
        );

      setBrand(savedBrand);
      setDraft(createDraft(savedBrand));
      setAccountId(String(savedBrand.accountId ?? ""));
      setBrandIconFile(null);
      setBrandBackgroundFile(null);
      setBrandIconError(null);
      setBrandBackgroundImageError(null);
      setAccountError(null);

      if (brandIconInputRef.current) {
        brandIconInputRef.current.value = "";
      }

      if (brandBackgroundInputRef.current) {
        brandBackgroundInputRef.current.value = "";
      }

      setIsEditing(false);

      setProgress(
        createCompletedBrandCreateProgress({
          transferredBytes,
          totalBytes,
          completedUploadCount,
          expectedUploadCount,
          title: "更新が完了しました",
          message: "ブランド情報の更新が完了しました。",
        }),
      );
    } catch (error: unknown) {
      const message = getErrorMessage(error);

      setError(new Error(message));

      setProgress(
        createFailedBrandCreateProgress(
          message,
          {
            title: "ブランド更新に失敗しました",
            message:
              "ブランド情報または画像の保存中にエラーが発生しました。",
          },
        ),
      );
    } finally {
      setSaving(false);
    }
  }, [
    resolvedBrandId,
    saving,
    draft,
    managerId,
    accountId,
    uploadBrandAssets,
    validateSelectedImagesBeforeSave,
  ]);

  const onCloseProgress = useCallback(() => {
    if (progress.isBlockingNavigation) return;

    setProgress(
      createInitialBrandCreateProgress(),
    );
  }, [progress.isBlockingNavigation]);

  const statusBadgeClass = useMemo(
    () =>
      brand.isActive
        ? "inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700 text-xs font-semibold"
        : "inline-flex items-center px-2 py-1 rounded-full bg-slate-50 text-slate-500 text-xs font-semibold",
    [brand.isActive],
  );

  return {
    brand,
    setBrand,
    registeredAt,
    updatedAt,
    statusLabel,
    isEditing,
    draft,
    setDraft,
    handleEdit,
    handleCancelEdit,
    handleSave,
    handleBack,
    statusBadgeClass,
    loading,
    saving,
    error,

    progress,
    progressOpen,
    isUploading,
    onCloseProgress,

    managerId,
    managerCandidates,
    loadingMembers,
    editingManagerName,
    handleSelectManager,

    accountId,
    accountCandidates,
    loadingAccounts,
    accountError,
    handleSelectAccount,
    handleOpenAccountConnect,

    brandImageAccept: BRAND_IMAGE_ACCEPT,
    brandIconInputRef,
    brandBackgroundInputRef,
    brandIconFile,
    brandBackgroundFile,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,
    brandIconError,
    brandBackgroundImageError,
    handlePickBrandIcon,
    handlePickBrandBackground,
    handleBrandIconChange,
    handleBrandBackgroundChange,
    handleClearBrandIcon,
    handleClearBrandBackground,
  };
}