// frontend/console/shell/src/features/brand/presentation/hook/useBrandCreate.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import {
  IMAGE_STORAGE_ACCEPT,
  validateImageForStorage,
  type ImageStorageTarget,
} from "../../../../shared/storage/imageStoragePolicy";
import type { Account } from "../../../../shared/types/account";
import type { IconCropPosition } from "../../../../shared/types/iconCrop";
import { cropIconImage } from "../../../../shared/util/cropIconImage";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import { accountRepositoryHTTP } from "../../../account/infrastructure/http/accountRepositoryHTTP";
import {
  brandRepositoryHTTP,
  type CreateBrandInput,
} from "../../infrastructure/http/brandRepositoryHTTP";
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

type BrandImageTarget = Extract<
  ImageStorageTarget,
  "brandIcon" | "brandBackgroundImage"
>;

export type BrandAccountCandidate = {
  id: string;
  label: string;
  status: Account["status"];
};

type UploadBrandAssetsResult = {
  uploadedBrandIcon: string;
  uploadedBrandBackgroundImage: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

const INITIAL_BRAND_ICON_CROP_POSITION: IconCropPosition = {
  x: 0,
  y: 0,
};

const INITIAL_BRAND_ICON_CROP_SCALE = 1;

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function buildAccountLabel(account: Account): string {
  const bankName = String(account.bankName ?? "").trim();
  const branchName = String(account.branchName ?? "").trim();
  const accountNumber = Number(account.accountNumber ?? 0);
  const bankLabel = [bankName, branchName].filter(Boolean).join(" ");
  const numberLabel = accountNumber > 0 ? String(accountNumber) : "";

  if (bankLabel && numberLabel) return `${bankLabel} ${numberLabel}`;
  if (bankLabel) return bankLabel;
  if (numberLabel) return numberLabel;
  return account.id;
}

export function useBrandCreate() {
  const navigate = useNavigate();
  const { currentMember } = useAuthContext();

  const companyId = useMemo(
    () => String(currentMember?.companyId ?? ""),
    [currentMember?.companyId],
  );

  const [accountId, setAccountId] = useState("");
  const [accountCandidates, setAccountCandidates] = useState<BrandAccountCandidate[]>([]);
  const [loadingAccounts, setLoadingAccounts] = useState(true);
  const [accountIdError, setAccountIdError] = useState<string | null>(null);
  const [accountLoadError, setAccountLoadError] = useState<string | null>(null);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [websiteUrl, setWebsiteUrl] = useState("");

  const [brandIcon, setBrandIcon] = useState("");
  const [brandBackgroundImage, setBrandBackgroundImage] = useState("");

  const {
    assigneeId: managerId,
    assigneeName: managerDisplayName,
    assigneeCandidates: managerCandidates,
    loadingMembers: loadingManagers,
    handleSelectAssignee,
  } = useAssigneeSelection({ defaultToCurrentMember: true });

  const [nameError, setNameError] = useState<string | null>(null);
  const [managerIdError, setManagerIdError] = useState<string | null>(null);
  const [brandIconError, setBrandIconError] = useState<string | null>(null);
  const [brandBackgroundImageError, setBrandBackgroundImageError] = useState<string | null>(null);

  const [saving, setSaving] = useState(false);
  const [progress, setProgress] = useState<BrandCreateProgress>(
    createInitialBrandCreateProgress,
  );
  const [createdBrandId, setCreatedBrandId] = useState("");

  const brandIconInputRef = useRef<HTMLInputElement | null>(null);
  const brandBackgroundInputRef = useRef<HTMLInputElement | null>(null);

  const [brandIconFile, setBrandIconFile] = useState<File | null>(null);
  const [brandBackgroundFile, setBrandBackgroundFile] = useState<File | null>(null);
  const [brandIconCropPosition, setBrandIconCropPosition] = useState<IconCropPosition>(
    INITIAL_BRAND_ICON_CROP_POSITION,
  );
  const [brandIconCropScale, setBrandIconCropScale] = useState(INITIAL_BRAND_ICON_CROP_SCALE);
  const [brandIconCropViewportSize, setBrandIconCropViewportSize] = useState(0);

  const [brandIconPreviewUrl, setBrandIconPreviewUrl] = useState("");
  const [brandBackgroundPreviewUrl, setBrandBackgroundPreviewUrl] = useState("");

  const isActive = true;
  const isUploading = progress.phase === "uploading" && saving;
  const progressOpen = isBrandCreateProgressVisible(progress);

  const resetBrandIconCrop = useCallback(() => {
    setBrandIconCropPosition(INITIAL_BRAND_ICON_CROP_POSITION);
    setBrandIconCropScale(INITIAL_BRAND_ICON_CROP_SCALE);
    setBrandIconCropViewportSize(0);
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadAccounts = async () => {
      try {
        setLoadingAccounts(true);
        setAccountLoadError(null);

        const accounts = await accountRepositoryHTTP.list();
        if (cancelled) return;

        const candidates = accounts
          .filter((account) => account.status !== "deleted")
          .map((account) => ({
            id: account.id,
            label: buildAccountLabel(account),
            status: account.status,
          }));

        setAccountCandidates(candidates);
        if (candidates.length === 1) setAccountId(candidates[0].id);
      } catch (error: unknown) {
        if (cancelled) return;

        setAccountCandidates([]);
        setAccountId("");
        setAccountLoadError(getErrorMessage(error));
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
      setBrandIconPreviewUrl(brandIcon);
      return;
    }

    const objectUrl = URL.createObjectURL(brandIconFile);
    setBrandIconPreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [brandIconFile, brandIcon]);

  useEffect(() => {
    if (!brandBackgroundFile) {
      setBrandBackgroundPreviewUrl(brandBackgroundImage);
      return;
    }

    const objectUrl = URL.createObjectURL(brandBackgroundFile);
    setBrandBackgroundPreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [brandBackgroundFile, brandBackgroundImage]);

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

  const displayBrandName = useMemo(
    () => name || "ブランド名未入力",
    [name],
  );

  const displayWebsiteUrl = useMemo(
    () => websiteUrl || "Webサイト未設定",
    [websiteUrl],
  );

  const hasBrandIconSelection = useMemo(
    () => Boolean(brandIconFile || brandIcon),
    [brandIconFile, brandIcon],
  );

  const hasBrandBackgroundSelection = useMemo(
    () => Boolean(brandBackgroundFile || brandBackgroundImage),
    [brandBackgroundFile, brandBackgroundImage],
  );

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
    if (id) setAccountIdError(null);
  }, []);

  const handleSelectManager = useCallback(
    (id: string) => {
      handleSelectAssignee(id);
      if (id) setManagerIdError(null);
    },
    [handleSelectAssignee],
  );

  const handlePickBrandIcon = useCallback(() => {
    if (saving) return;
    brandIconInputRef.current?.click();
  }, [saving]);

  const handlePickBrandBackground = useCallback(() => {
    if (saving) return;
    brandBackgroundInputRef.current?.click();
  }, [saving]);

  const validateSelectedImage = useCallback(
    (file: File, target: BrandImageTarget): string | null => {
      const validation = validateImageForStorage(file, target);
      return validation.valid ? null : validation.reason;
    },
    [],
  );

  const handleBrandIconChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.currentTarget.files?.[0] ?? null;
      event.currentTarget.value = "";

      if (!file) {
        return;
      }

      const validationError = validateSelectedImage(file, "brandIcon");

      if (validationError) {
        setBrandIconFile(null);
        setBrandIcon("");
        setBrandIconError(validationError);
        resetBrandIconCrop();
        alert(validationError);
        return;
      }

      setBrandIconFile(file);
      setBrandIcon("");
      setBrandIconError(null);
      resetBrandIconCrop();
    },
    [validateSelectedImage, resetBrandIconCrop],
  );

  const handleBrandBackgroundChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file = event.currentTarget.files?.[0] ?? null;

      if (!file) {
        setBrandBackgroundFile(null);
        setBrandBackgroundImageError(null);
        return;
      }

      const validationError = validateSelectedImage(file, "brandBackgroundImage");

      if (validationError) {
        setBrandBackgroundFile(null);
        setBrandBackgroundImage("");
        setBrandBackgroundImageError(validationError);
        event.currentTarget.value = "";
        alert(validationError);
        return;
      }

      setBrandBackgroundFile(file);
      setBrandBackgroundImage("");
      setBrandBackgroundImageError(null);
    },
    [validateSelectedImage],
  );

  const handleClearBrandIcon = useCallback(() => {
    setBrandIconFile(null);
    setBrandIcon("");
    setBrandIconError(null);
    resetBrandIconCrop();

    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }
  }, [resetBrandIconCrop]);

  const handleClearBrandBackground = useCallback(() => {
    setBrandBackgroundFile(null);
    setBrandBackgroundImage("");
    setBrandBackgroundImageError(null);

    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }
  }, []);

  const validateSelectedImagesBeforeSave = useCallback((): boolean => {
    if (brandIconFile) {
      const validationError = validateSelectedImage(brandIconFile, "brandIcon");

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

  const buildBrandIconFileForUpload = useCallback(async (): Promise<File | null> => {
    if (!brandIconFile) {
      return null;
    }

    if (brandIconCropViewportSize <= 0) {
      throw new Error(
        "ブランドアイコンの切り抜き領域を取得できませんでした。画像を選択し直してください。",
      );
    }

    const croppedFile = await cropIconImage({
      file: brandIconFile,
      position: brandIconCropPosition,
      scale: brandIconCropScale,
      viewportSize: brandIconCropViewportSize,
    });

    const validation = validateImageForStorage(croppedFile, "brandIcon");

    if (!validation.valid) {
      throw new Error(validation.reason);
    }

    return croppedFile;
  }, [
    brandIconFile,
    brandIconCropPosition,
    brandIconCropScale,
    brandIconCropViewportSize,
  ]);

  const uploadBrandAssets = useCallback(
    async (
      brandId: string,
      croppedBrandIconFile: File | null,
    ): Promise<UploadBrandAssetsResult> => {
      let uploadedBrandIcon = brandIcon;
      let uploadedBrandBackgroundImage = brandBackgroundImage;
      let completedBytes = 0;
      let completedUploadCount = 0;

      const totalBytes =
        (croppedBrandIconFile?.size ?? 0) +
        (brandBackgroundFile?.size ?? 0);

      const expectedUploadCount =
        (croppedBrandIconFile ? 1 : 0) +
        (brandBackgroundFile ? 1 : 0);

      if (croppedBrandIconFile) {
        const currentFile = croppedBrandIconFile;

        const uploaded = await uploadBrandAssetToFirebaseStorage({
          companyId,
          brandId,
          target: "brandIcon",
          file: currentFile,
          onProgress: (uploadProgress) => {
            setProgress(
              createUploadingBrandCreateProgress({
                fileName: currentFile.name,
                transferredBytes: completedBytes + uploadProgress.transferredBytes,
                totalBytes,
                completedUploadCount,
                expectedUploadCount,
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

        const uploaded = await uploadBrandAssetToFirebaseStorage({
          companyId,
          brandId,
          target: "brandBackgroundImage",
          file: currentFile,
          onProgress: (uploadProgress) => {
            setProgress(
              createUploadingBrandCreateProgress({
                fileName: currentFile.name,
                transferredBytes: completedBytes + uploadProgress.transferredBytes,
                totalBytes,
                completedUploadCount,
                expectedUploadCount,
              }),
            );
          },
        });

        uploadedBrandBackgroundImage = uploaded.downloadUrl;
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
      companyId,
      brandIcon,
      brandBackgroundImage,
      brandBackgroundFile,
    ],
  );

  const onCloseProgress = useCallback(() => {
    if (saving || progress.isBlockingNavigation) return;

    if (
      (progress.phase === "completed" || progress.phase === "failed") &&
      createdBrandId
    ) {
      setProgress(createInitialBrandCreateProgress());
      setCreatedBrandId("");
      navigate("/brand");
      return;
    }

    setProgress(createInitialBrandCreateProgress());
  }, [
    saving,
    progress.isBlockingNavigation,
    progress.phase,
    createdBrandId,
    navigate,
  ]);

  const handleSave = useCallback(async () => {
    if (saving) return;

    const normalizedAccountId = String(accountId ?? "").trim();
    const normalizedName = String(name ?? "");
    const normalizedManagerId = String(managerId ?? "");

    let hasError = false;

    if (!normalizedAccountId) {
      setAccountIdError("受取口座は必須です。");
      hasError = true;
    } else {
      setAccountIdError(null);
    }

    if (!normalizedName) {
      setNameError("ブランド名は必須です。");
      hasError = true;
    } else {
      setNameError(null);
    }

    if (!normalizedManagerId) {
      setManagerIdError("ブランド責任者は必須です。");
      hasError = true;
    } else {
      setManagerIdError(null);
    }

    if (hasError) {
      alert("受取口座、ブランド名、ブランド責任者を入力してください。");
      return;
    }

    if (!companyId) {
      alert("companyId が取得できません。");
      return;
    }

    if (!validateSelectedImagesBeforeSave()) return;

    let localCreatedBrandId = "";

    try {
      const croppedBrandIconFile = await buildBrandIconFileForUpload();

      setSaving(true);
      setCreatedBrandId("");
      setProgress(
        createCreatingBrandCreateProgress({
          title: "ブランドを登録中",
          message: "ブランド情報を登録しています。",
        }),
      );

      const createPayload: CreateBrandInput = {
        companyId,
        accountId: normalizedAccountId,
        name: normalizedName,
        description,
        websiteUrl,
        brandIcon: "",
        brandBackgroundImage: "",
        managerId: normalizedManagerId,
        createdBy: currentMember?.id ?? null,
      };

      const created = await brandRepositoryHTTP.create(createPayload);
      localCreatedBrandId = String(created.id ?? "");

      if (!localCreatedBrandId) {
        throw new Error("brandId が取得できません。");
      }

      setCreatedBrandId(localCreatedBrandId);

      const {
        uploadedBrandIcon,
        uploadedBrandBackgroundImage,
        transferredBytes,
        totalBytes,
        completedUploadCount,
        expectedUploadCount,
      } = await uploadBrandAssets(
        localCreatedBrandId,
        croppedBrandIconFile,
      );

      if (uploadedBrandIcon || uploadedBrandBackgroundImage) {
        setProgress(
          createSavingBrandCreateProgress({
            transferredBytes,
            totalBytes,
            completedUploadCount,
            expectedUploadCount,
            title: "ブランド情報を保存中",
            message: "画像転送が完了しました。ブランド情報を更新しています。",
          }),
        );

        await brandRepositoryHTTP.update(localCreatedBrandId, {
          brandIcon: uploadedBrandIcon,
          brandBackgroundImage: uploadedBrandBackgroundImage,
        });
      }

      setProgress(
        createCompletedBrandCreateProgress({
          transferredBytes,
          totalBytes,
          completedUploadCount,
          expectedUploadCount,
          title: "登録が完了しました",
          message: "ブランドの登録が完了しました。",
        }),
      );
    } catch (error: unknown) {
      const message = getErrorMessage(error);

      if (localCreatedBrandId) {
        setCreatedBrandId(localCreatedBrandId);
        setProgress(
          createFailedBrandCreateProgress(
            message,
            {
              title: "画像の保存に失敗しました",
              message:
                "ブランド本体は登録されましたが、画像のアップロードまたはURL保存に失敗しました。",
            },
          ),
        );
      } else {
        setBrandIconError(
          brandIconFile
            ? message
            : null,
        );

        setProgress(
          createFailedBrandCreateProgress(
            message,
            {
              title: "ブランド登録に失敗しました",
              message: "ブランド情報の登録中にエラーが発生しました。",
            },
          ),
        );
      }
    } finally {
      setSaving(false);
    }
  }, [
    saving,
    accountId,
    name,
    managerId,
    companyId,
    description,
    websiteUrl,
    currentMember?.id,
    brandIconFile,
    uploadBrandAssets,
    buildBrandIconFileForUpload,
    validateSelectedImagesBeforeSave,
  ]);

  return {
    companyId,

    accountId,
    accountIdError,
    accountCandidates,
    loadingAccounts,
    accountLoadError,
    handleSelectAccount,
    handleOpenAccountConnect,

    name,
    setName,
    nameError,

    description,
    setDescription,

    websiteUrl,
    setWebsiteUrl,

    brandIcon,
    setBrandIcon,

    brandBackgroundImage,
    setBrandBackgroundImage,

    managerId,
    managerIdError,
    managerDisplayName,
    managerCandidates,
    loadingManagers,
    handleSelectManager,

    displayBrandName,
    displayWebsiteUrl,

    brandImageAccept: IMAGE_STORAGE_ACCEPT,

    hasBrandIconSelection,
    hasBrandBackgroundSelection,

    brandIconInputRef,
    brandBackgroundInputRef,

    brandIconFile,
    brandBackgroundFile,

    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,

    brandIconCropPosition,
    brandIconCropScale,
    handleBrandIconCropPositionChange: setBrandIconCropPosition,
    handleBrandIconCropScaleChange: setBrandIconCropScale,
    handleBrandIconCropViewportSizeChange: setBrandIconCropViewportSize,

    brandIconError,
    brandBackgroundImageError,

    handlePickBrandIcon,
    handlePickBrandBackground,

    handleBrandIconChange,
    handleBrandBackgroundChange,

    handleClearBrandIcon,
    handleClearBrandBackground,

    isActive,
    saving,
    isUploading,

    progress,
    progressOpen,
    onCloseProgress,

    handleBack,
    handleSave,
  };
}