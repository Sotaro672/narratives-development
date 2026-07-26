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

import { useAuth } from "../../../../auth/presentation/hook/useCurrentMember";

import type { Member } from "../../../member/domain/entity/member";
import type { MemberFilter } from "../../../member/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../member/infrastructure/http/memberRepositoryHTTP";

import type { Brand } from "../../domain/entity/brand";
import { validateBrandImage } from "../../application/brandImageValidation";
import {
  BRAND_IMAGE_ALLOWED_MIME_TYPES,
  type BrandImageTarget,
} from "../../config/brandImagePolicy.generated";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";
import { uploadBrandAssetToFirebaseStorage } from "../../infrastructure/storage/brandAssetStorage";

const memberRepo = new MemberRepositoryHTTP();

const BRAND_IMAGE_ACCEPT =
  BRAND_IMAGE_ALLOWED_MIME_TYPES.join(",");

function formatLastFirst(
  lastName?: string | null,
  firstName?: string | null,
): string {
  const normalizedLastName = String(lastName ?? "");
  const normalizedFirstName = String(firstName ?? "");

  if (normalizedLastName && normalizedFirstName) {
    return `${normalizedLastName} ${normalizedFirstName}`;
  }

  if (normalizedLastName) {
    return normalizedLastName;
  }

  if (normalizedFirstName) {
    return normalizedFirstName;
  }

  return "";
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : String(error);
}

export function useBrandCreate() {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  const companyId = useMemo(
    () => String(currentMember?.companyId ?? ""),
    [currentMember?.companyId],
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [websiteUrl, setWebsiteUrl] = useState("");

  const [brandIcon, setBrandIcon] = useState("");
  const [brandBackgroundImage, setBrandBackgroundImage] =
    useState("");

  const [managerId, setManagerId] =
    useState<string | null>(null);

  const [nameError, setNameError] =
    useState<string | null>(null);

  const [managerIdError, setManagerIdError] =
    useState<string | null>(null);

  const [brandIconError, setBrandIconError] =
    useState<string | null>(null);

  const [
    brandBackgroundImageError,
    setBrandBackgroundImageError,
  ] = useState<string | null>(null);

  const [managerOptions, setManagerOptions] =
    useState<Member[]>([]);

  const [loadingManagers, setLoadingManagers] =
    useState(false);

  const [managerError, setManagerError] =
    useState<string | null>(null);

  const [saving, setSaving] = useState(false);

  const brandIconInputRef =
    useRef<HTMLInputElement | null>(null);

  const brandBackgroundInputRef =
    useRef<HTMLInputElement | null>(null);

  const [brandIconFile, setBrandIconFile] =
    useState<File | null>(null);

  const [brandBackgroundFile, setBrandBackgroundFile] =
    useState<File | null>(null);

  const [brandIconPreviewUrl, setBrandIconPreviewUrl] =
    useState("");

  const [
    brandBackgroundPreviewUrl,
    setBrandBackgroundPreviewUrl,
  ] = useState("");

  const isActive = true;

  useEffect(() => {
    let cancelled = false;

    async function loadManagers() {
      try {
        setLoadingManagers(true);
        setManagerError(null);

        const filter: MemberFilter = {};

        const { items } = await memberRepo.list(
          {
            number: 1,
            perPage: 200,
            totalPages: 1,
          },
          filter,
        );

        if (cancelled) {
          return;
        }

        setManagerOptions(items);

        setManagerId((currentManagerId) => {
          if (currentManagerId) {
            return currentManagerId;
          }

          return items[0]?.id ?? null;
        });
      } catch (error: unknown) {
        if (!cancelled) {
          setManagerError(
            getErrorMessage(error) ||
              "ブランド責任者候補の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoadingManagers(false);
        }
      }
    }

    void loadManagers();

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
      setBrandBackgroundPreviewUrl(
        brandBackgroundImage,
      );
      return;
    }

    const objectUrl = URL.createObjectURL(
      brandBackgroundFile,
    );

    setBrandBackgroundPreviewUrl(objectUrl);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [
    brandBackgroundFile,
    brandBackgroundImage,
  ]);

  const selectedManager = useMemo(
    () =>
      managerOptions.find(
        (manager) => manager.id === managerId,
      ) ?? null,
    [managerOptions, managerId],
  );

  const managerDisplayName = useMemo(() => {
    if (!selectedManager) {
      return "責任者未設定";
    }

    return (
      formatLastFirst(
        selectedManager.lastName,
        selectedManager.firstName,
      ) ||
      selectedManager.email ||
      selectedManager.id
    );
  }, [selectedManager]);

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
    () =>
      Boolean(
        brandBackgroundFile ||
          brandBackgroundImage,
      ),
    [
      brandBackgroundFile,
      brandBackgroundImage,
    ],
  );

  const handleBack = useCallback(() => {
    navigate("/brand");
  }, [navigate]);

  const handlePickBrandIcon = useCallback(() => {
    brandIconInputRef.current?.click();
  }, []);

  const handlePickBrandBackground =
    useCallback(() => {
      brandBackgroundInputRef.current?.click();
    }, []);

  const validateSelectedImage = useCallback(
    (
      file: File,
      target: BrandImageTarget,
    ): string | null => {
      const validation = validateBrandImage(
        file,
        target,
      );

      if (!validation.valid) {
        return validation.message;
      }

      return null;
    },
    [],
  );

  const handleBrandIconChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const file =
        event.currentTarget.files?.[0] ?? null;

      if (!file) {
        setBrandIconFile(null);
        setBrandIconError(null);
        return;
      }

      const validationError =
        validateSelectedImage(
          file,
          "brandIcon",
        );

      if (validationError) {
        setBrandIconFile(null);
        setBrandIcon("");
        setBrandIconError(validationError);
        event.currentTarget.value = "";

        alert(validationError);
        return;
      }

      setBrandIconFile(file);
      setBrandIcon("");
      setBrandIconError(null);
    },
    [validateSelectedImage],
  );

  const handleBrandBackgroundChange =
    useCallback(
      (
        event: ChangeEvent<HTMLInputElement>,
      ) => {
        const file =
          event.currentTarget.files?.[0] ?? null;

        if (!file) {
          setBrandBackgroundFile(null);
          setBrandBackgroundImageError(null);
          return;
        }

        const validationError =
          validateSelectedImage(
            file,
            "brandBackgroundImage",
          );

        if (validationError) {
          setBrandBackgroundFile(null);
          setBrandBackgroundImage("");
          setBrandBackgroundImageError(
            validationError,
          );
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

  const handleClearBrandIcon =
    useCallback(() => {
      setBrandIconFile(null);
      setBrandIcon("");
      setBrandIconError(null);

      if (brandIconInputRef.current) {
        brandIconInputRef.current.value = "";
      }
    }, []);

  const handleClearBrandBackground =
    useCallback(() => {
      setBrandBackgroundFile(null);
      setBrandBackgroundImage("");
      setBrandBackgroundImageError(null);

      if (brandBackgroundInputRef.current) {
        brandBackgroundInputRef.current.value =
          "";
      }
    }, []);

  const validateSelectedImagesBeforeSave =
    useCallback((): boolean => {
      if (brandIconFile) {
        const validationError =
          validateSelectedImage(
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
        const validationError =
          validateSelectedImage(
            brandBackgroundFile,
            "brandBackgroundImage",
          );

        if (validationError) {
          setBrandBackgroundImageError(
            validationError,
          );
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
    async (
      brandId: string,
    ): Promise<{
      uploadedBrandIcon: string;
      uploadedBrandBackgroundImage: string;
    }> => {
      let uploadedBrandIcon = brandIcon;
      let uploadedBrandBackgroundImage =
        brandBackgroundImage;

      if (brandIconFile) {
        const uploaded =
          await uploadBrandAssetToFirebaseStorage({
            companyId,
            brandId,
            target: "brandIcon",
            file: brandIconFile,
          });

        uploadedBrandIcon = uploaded.downloadUrl;
      }

      if (brandBackgroundFile) {
        const uploaded =
          await uploadBrandAssetToFirebaseStorage({
            companyId,
            brandId,
            target: "brandBackgroundImage",
            file: brandBackgroundFile,
          });

        uploadedBrandBackgroundImage =
          uploaded.downloadUrl;
      }

      return {
        uploadedBrandIcon,
        uploadedBrandBackgroundImage,
      };
    },
    [
      companyId,
      brandIcon,
      brandBackgroundImage,
      brandIconFile,
      brandBackgroundFile,
    ],
  );

  const handleSave = useCallback(async () => {
    if (saving) {
      return;
    }

    const normalizedName = String(name ?? "");
    const normalizedManagerId = String(
      managerId ?? "",
    );

    let hasError = false;

    if (!normalizedName) {
      setNameError("ブランド名は必須です。");
      hasError = true;
    } else {
      setNameError(null);
    }

    if (!normalizedManagerId) {
      setManagerIdError(
        "ブランド責任者は必須です。",
      );
      hasError = true;
    } else {
      setManagerIdError(null);
    }

    if (hasError) {
      alert(
        "ブランド名とブランド責任者を入力してください。",
      );
      return;
    }

    if (!companyId) {
      alert("companyId が取得できません。");
      return;
    }

    if (!validateSelectedImagesBeforeSave()) {
      return;
    }

    let createdBrandId = "";

    try {
      setSaving(true);

      const createPayload: Omit<
        Brand,
        "createdAt" | "updatedAt"
      > = {
        id: "",
        companyId,
        name: normalizedName,
        description,
        websiteUrl,
        brandIcon: "",
        brandBackgroundImage: "",
        isActive: true,
        managerId: normalizedManagerId,
        walletAddress: "pending",
        createdBy:
          (currentMember?.id ?? null) as any,
        updatedBy: null as any,
        deletedAt: null as any,
        deletedBy: null as any,
      } as any;

      const created =
        await brandRepositoryHTTP.create(
          createPayload,
        );

      createdBrandId = String(created.id ?? "");

      if (!createdBrandId) {
        throw new Error(
          "brandId が取得できません。",
        );
      }

      const {
        uploadedBrandIcon,
        uploadedBrandBackgroundImage,
      } = await uploadBrandAssets(
        createdBrandId,
      );

      if (
        uploadedBrandIcon ||
        uploadedBrandBackgroundImage
      ) {
        await brandRepositoryHTTP.update(
          createdBrandId,
          {
            brandIcon: uploadedBrandIcon,
            brandBackgroundImage:
              uploadedBrandBackgroundImage,
          },
        );
      }

      alert("ブランドを登録しました。");
      navigate("/brand");
    } catch (error: unknown) {
      const message = getErrorMessage(error);

      if (createdBrandId) {
        alert(
          "ブランド本体は登録されましたが、" +
            `画像のアップロードまたはURL保存に失敗しました: ${message}`,
        );
      } else {
        alert(
          `ブランド登録に失敗しました: ${message}`,
        );
      }
    } finally {
      setSaving(false);
    }
  }, [
    saving,
    name,
    managerId,
    companyId,
    description,
    websiteUrl,
    currentMember?.id,
    uploadBrandAssets,
    validateSelectedImagesBeforeSave,
    navigate,
  ]);

  return {
    companyId,

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
    setManagerId,
    managerIdError,

    managerOptions,
    loadingManagers,
    managerError,

    formatLastFirst,
    selectedManager,
    managerDisplayName,

    displayBrandName,
    displayWebsiteUrl,

    brandImageAccept: BRAND_IMAGE_ACCEPT,

    hasBrandIconSelection,
    hasBrandBackgroundSelection,

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

    isActive,
    saving,

    handleBack,
    handleSave,
  };
}