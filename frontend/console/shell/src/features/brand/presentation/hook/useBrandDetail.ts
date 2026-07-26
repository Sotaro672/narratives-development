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

import { safeDateLabelJa } from "../../../../shared/util/dateJa";

import type { Member } from "../../../member/domain/entity/member";
import type { MemberFilter } from "../../../member/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../member/infrastructure/http/memberRepositoryHTTP";

import { validateBrandImage } from "../../application/brandImageValidation";
import {
  BRAND_IMAGE_ALLOWED_MIME_TYPES,
  type BrandImageTarget,
} from "../../config/brandImagePolicy.generated";
import type { BrandPatch } from "../../domain/entity/brand";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";
import { uploadBrandAssetToFirebaseStorage } from "../../infrastructure/storage/brandAssetStorage";

const memberRepository = new MemberRepositoryHTTP();

const BRAND_IMAGE_ACCEPT =
  BRAND_IMAGE_ALLOWED_MIME_TYPES.join(",");

export type BrandManagerCandidate = {
  id: string;
  name: string;
};

export interface BrandDetailData {
  id: string;
  companyId: string;
  name: string;
  description: string;
  websiteUrl?: string;
  brandIcon?: string;
  brandBackgroundImage?: string;
  isActive: boolean;
  managerId: string;
  memberName?: string;
  walletAddress: string;
  createdAt: string;
  createdBy?: string | null;
  updatedAtRaw?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
  status: string;
  registeredAt: string;
  updatedAt: string;
  managerName?: string;
}

type BrandDraft = {
  name: string;
  description: string;
  websiteUrl: string;
  brandIcon: string;
  brandBackgroundImage: string;
  isActive: boolean;
  managerId: string;
};

type BrandResponse = {
  id?: string | null;
  companyId?: string | null;
  name?: string | null;
  description?: string | null;
  websiteUrl?: string | null;
  brandIcon?: string | null;
  brandBackgroundImage?: string | null;
  isActive?: boolean | null;
  managerId?: string | null;
  memberName?: string | null;
  walletAddress?: string | null;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
};

function createEmptyBrand(
  brandId: string,
): BrandDetailData {
  return {
    id: brandId,
    companyId: "",
    name: "",
    description: "",
    websiteUrl: "",
    brandIcon: "",
    brandBackgroundImage: "",
    isActive: false,
    managerId: "",
    memberName: "",
    managerName: "",
    walletAddress: "",
    createdAt: "",
    createdBy: null,
    updatedAtRaw: null,
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
    status: "",
    registeredAt: "",
    updatedAt: "",
  };
}

function createDraft(
  brand: BrandDetailData,
): BrandDraft {
  return {
    name: brand.name,
    description: brand.description,
    websiteUrl: brand.websiteUrl ?? "",
    brandIcon: brand.brandIcon ?? "",
    brandBackgroundImage:
      brand.brandBackgroundImage ?? "",
    isActive: brand.isActive,
    managerId: brand.managerId,
  };
}

function formatMemberName(
  member: Member,
): string {
  const lastName = String(member.lastName ?? "");
  const firstName = String(member.firstName ?? "");

  if (lastName && firstName) {
    return `${lastName} ${firstName}`;
  }

  if (lastName) {
    return lastName;
  }

  if (firstName) {
    return firstName;
  }

  return member.email || member.id;
}

function toBrandDetailData(
  data: BrandResponse,
  fallback: BrandDetailData,
): BrandDetailData {
  const id = String(
    data.id ?? fallback.id,
  );

  const companyId = String(
    data.companyId ?? fallback.companyId,
  );

  const name = String(
    data.name ?? fallback.name,
  );

  const description = String(
    data.description ?? fallback.description,
  );

  const websiteUrl = String(
    data.websiteUrl ??
      fallback.websiteUrl ??
      "",
  );

  const brandIcon = String(
    data.brandIcon ??
      fallback.brandIcon ??
      "",
  );

  const brandBackgroundImage = String(
    data.brandBackgroundImage ??
      fallback.brandBackgroundImage ??
      "",
  );

  const isActive = Boolean(
    data.isActive ?? fallback.isActive,
  );

  const managerId = String(
    data.managerId ?? fallback.managerId,
  );

  const memberName = String(
    data.memberName ??
      fallback.memberName ??
      fallback.managerName ??
      "",
  );

  const walletAddress = String(
    data.walletAddress ??
      fallback.walletAddress,
  );

  const createdAt = String(
    data.createdAt ?? fallback.createdAt,
  );

  const createdBy =
    data.createdBy ??
    fallback.createdBy ??
    null;

  const updatedAtRaw =
    data.updatedAt ??
    fallback.updatedAtRaw ??
    null;

  const updatedBy =
    data.updatedBy ??
    fallback.updatedBy ??
    null;

  const deletedAt =
    data.deletedAt ??
    fallback.deletedAt ??
    null;

  const deletedBy =
    data.deletedBy ??
    fallback.deletedBy ??
    null;

  return {
    id,
    companyId,
    name,
    description,
    websiteUrl,
    brandIcon,
    brandBackgroundImage,
    isActive,
    managerId,
    memberName,
    managerName: memberName,
    walletAddress,
    createdAt,
    createdBy,
    updatedAtRaw,
    updatedBy,
    deletedAt,
    deletedBy,
    status: isActive
      ? "アクティブ"
      : "停止",
    registeredAt: safeDateLabelJa(
      createdAt,
      "",
    ),
    updatedAt: safeDateLabelJa(
      updatedAtRaw ?? "",
      "",
    ),
  };
}

function getErrorMessage(
  error: unknown,
): string {
  return error instanceof Error
    ? error.message
    : String(error);
}

export function useBrandDetail() {
  const navigate = useNavigate();

  const { brandId } =
    useParams<{ brandId: string }>();

  const resolvedBrandId = brandId ?? "";

  const [brand, setBrand] =
    useState<BrandDetailData>(() =>
      createEmptyBrand(resolvedBrandId),
    );

  const [draft, setDraft] =
    useState<BrandDraft>(() =>
      createDraft(
        createEmptyBrand(resolvedBrandId),
      ),
    );

  const [loading, setLoading] =
    useState(false);

  const [saving, setSaving] =
    useState(false);

  const [error, setError] =
    useState<Error | null>(null);

  const [isEditing, setIsEditing] =
    useState(false);

  const [
    managerCandidates,
    setManagerCandidates,
  ] = useState<BrandManagerCandidate[]>([]);

  const [
    loadingMembers,
    setLoadingMembers,
  ] = useState(false);

  const [memberError, setMemberError] =
    useState<string | null>(null);

  const [
    brandIconError,
    setBrandIconError,
  ] = useState<string | null>(null);

  const [
    brandBackgroundImageError,
    setBrandBackgroundImageError,
  ] = useState<string | null>(null);

  const brandIconInputRef =
    useRef<HTMLInputElement | null>(null);

  const brandBackgroundInputRef =
    useRef<HTMLInputElement | null>(null);

  const [
    brandIconFile,
    setBrandIconFile,
  ] = useState<File | null>(null);

  const [
    brandBackgroundFile,
    setBrandBackgroundFile,
  ] = useState<File | null>(null);

  const [
    brandIconPreviewUrl,
    setBrandIconPreviewUrl,
  ] = useState("");

  const [
    brandBackgroundPreviewUrl,
    setBrandBackgroundPreviewUrl,
  ] = useState("");

  useEffect(() => {
    let cancelled = false;

    const loadBrand = async () => {
      if (!resolvedBrandId) {
        return;
      }

      try {
        setLoading(true);
        setError(null);

        const response =
          await brandRepositoryHTTP.getById(
            resolvedBrandId,
          );

        const nextBrand = toBrandDetailData(
          response as unknown as BrandResponse,
          createEmptyBrand(resolvedBrandId),
        );

        if (cancelled) {
          return;
        }

        setBrand(nextBrand);
        setDraft(createDraft(nextBrand));

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
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadBrand();

    return () => {
      cancelled = true;
    };
  }, [resolvedBrandId]);

  useEffect(() => {
    if (!isEditing) {
      setLoadingMembers(false);
      return;
    }

    let cancelled = false;

    const loadMembers = async () => {
      try {
        setLoadingMembers(true);
        setMemberError(null);

        const filter: MemberFilter = {};

        const result =
          await memberRepository.list(
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

        const candidates =
          (result.items ?? []).map(
            (member) => ({
              id: member.id,
              name: formatMemberName(member),
            }),
          );

        setManagerCandidates(candidates);
      } catch (error: unknown) {
        if (!cancelled) {
          setManagerCandidates([]);

          setMemberError(
            getErrorMessage(error) ||
              "責任者候補の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoadingMembers(false);
        }
      }
    };

    void loadMembers();

    return () => {
      cancelled = true;
    };
  }, [isEditing]);

  useEffect(() => {
    if (!brandIconFile) {
      setBrandIconPreviewUrl(
        isEditing
          ? draft.brandIcon
          : brand.brandIcon ?? "",
      );

      return;
    }

    const objectUrl =
      URL.createObjectURL(brandIconFile);

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

    const objectUrl =
      URL.createObjectURL(
        brandBackgroundFile,
      );

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

  const editingManagerName =
    useMemo(() => {
      const selectedCandidate =
        managerCandidates.find(
          (candidate) =>
            candidate.id ===
            draft.managerId,
        );

      if (selectedCandidate) {
        return selectedCandidate.name;
      }

      if (
        draft.managerId &&
        draft.managerId === brand.managerId
      ) {
        return (
          brand.managerName ||
          brand.memberName ||
          draft.managerId
        );
      }

      return draft.managerId || "未設定";
    }, [
      managerCandidates,
      draft.managerId,
      brand.managerId,
      brand.managerName,
      brand.memberName,
    ]);

  const handleBack = useCallback(() => {
    navigate("/brand");
  }, [navigate]);

  const handleEdit = useCallback(() => {
    setDraft(createDraft(brand));

    setBrandIconFile(null);
    setBrandBackgroundFile(null);

    setBrandIconError(null);
    setBrandBackgroundImageError(null);

    setMemberError(null);
    setError(null);

    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }

    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value =
        "";
    }

    setIsEditing(true);
  }, [brand]);

  const handleCancelEdit =
    useCallback(() => {
      setDraft(createDraft(brand));

      setBrandIconFile(null);
      setBrandBackgroundFile(null);

      setBrandIconError(null);
      setBrandBackgroundImageError(null);

      setMemberError(null);
      setError(null);

      if (brandIconInputRef.current) {
        brandIconInputRef.current.value = "";
      }

      if (brandBackgroundInputRef.current) {
        brandBackgroundInputRef.current.value =
          "";
      }

      setIsEditing(false);
    }, [brand]);

  const handleSelectManager =
    useCallback((managerId: string) => {
      setDraft((currentDraft) => ({
        ...currentDraft,
        managerId,
      }));

      setMemberError(null);
    }, []);

  const handlePickBrandIcon =
    useCallback(() => {
      if (!isEditing || saving) {
        return;
      }

      brandIconInputRef.current?.click();
    }, [isEditing, saving]);

  const handlePickBrandBackground =
    useCallback(() => {
      if (!isEditing || saving) {
        return;
      }

      brandBackgroundInputRef.current?.click();
    }, [isEditing, saving]);

  const validateSelectedImage =
    useCallback(
      (
        file: File,
        target: BrandImageTarget,
      ): string | null => {
        const validation =
          validateBrandImage(
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

  const handleBrandIconChange =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const file =
          event.currentTarget.files?.[0] ??
          null;

        if (!file) {
          return;
        }

        const validationError =
          validateSelectedImage(
            file,
            "brandIcon",
          );

        if (validationError) {
          setBrandIconFile(null);
          setBrandIconError(
            validationError,
          );

          event.currentTarget.value = "";

          alert(validationError);
          return;
        }

        setBrandIconFile(file);
        setBrandIconError(null);
      },
      [validateSelectedImage],
    );

  const handleBrandBackgroundChange =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const file =
          event.currentTarget.files?.[0] ??
          null;

        if (!file) {
          return;
        }

        const validationError =
          validateSelectedImage(
            file,
            "brandBackgroundImage",
          );

        if (validationError) {
          setBrandBackgroundFile(null);

          setBrandBackgroundImageError(
            validationError,
          );

          event.currentTarget.value = "";

          alert(validationError);
          return;
        }

        setBrandBackgroundFile(file);
        setBrandBackgroundImageError(null);
      },
      [validateSelectedImage],
    );

  const handleClearBrandIcon =
    useCallback(() => {
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

  const handleClearBrandBackground =
    useCallback(() => {
      setBrandBackgroundFile(null);
      setBrandBackgroundImageError(null);

      setDraft((currentDraft) => ({
        ...currentDraft,
        brandBackgroundImage: "",
      }));

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

  const uploadBrandAssets =
    useCallback(
      async (): Promise<{
        uploadedBrandIcon: string;
        uploadedBrandBackgroundImage: string;
      }> => {
        if (!resolvedBrandId) {
          throw new Error(
            "brandId が取得できません。",
          );
        }

        if (!brand.companyId) {
          throw new Error(
            "companyId が取得できません。",
          );
        }

        let uploadedBrandIcon =
          draft.brandIcon;

        let uploadedBrandBackgroundImage =
          draft.brandBackgroundImage;

        if (brandIconFile) {
          const uploaded =
            await uploadBrandAssetToFirebaseStorage(
              {
                companyId:
                  brand.companyId,
                brandId:
                  resolvedBrandId,
                target: "brandIcon",
                file: brandIconFile,
              },
            );

          uploadedBrandIcon =
            uploaded.downloadUrl;
        }

        if (brandBackgroundFile) {
          const uploaded =
            await uploadBrandAssetToFirebaseStorage(
              {
                companyId:
                  brand.companyId,
                brandId:
                  resolvedBrandId,
                target:
                  "brandBackgroundImage",
                file:
                  brandBackgroundFile,
              },
            );

          uploadedBrandBackgroundImage =
            uploaded.downloadUrl;
        }

        return {
          uploadedBrandIcon,
          uploadedBrandBackgroundImage,
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

  const handleSave =
    useCallback(async () => {
      if (
        !resolvedBrandId ||
        saving
      ) {
        return;
      }

      if (!draft.name) {
        setError(
          new Error(
            "ブランド名は必須です。",
          ),
        );

        return;
      }

      if (!draft.managerId) {
        setError(
          new Error(
            "ブランド責任者は必須です。",
          ),
        );

        return;
      }

      if (
        !validateSelectedImagesBeforeSave()
      ) {
        return;
      }

      try {
        setSaving(true);
        setError(null);

        const {
          uploadedBrandIcon,
          uploadedBrandBackgroundImage,
        } = await uploadBrandAssets();

        const patch: BrandPatch = {
          name: draft.name,
          description: draft.description,
          websiteUrl: draft.websiteUrl,
          brandIcon: uploadedBrandIcon,
          brandBackgroundImage:
            uploadedBrandBackgroundImage,
          isActive: draft.isActive,
          managerId: draft.managerId,
        };

        const response =
          await brandRepositoryHTTP.update(
            resolvedBrandId,
            patch,
          );

        const fallbackBrand:
          BrandDetailData = {
          ...brand,
          name: draft.name,
          description: draft.description,
          websiteUrl: draft.websiteUrl,
          brandIcon: uploadedBrandIcon,
          brandBackgroundImage:
            uploadedBrandBackgroundImage,
          isActive: draft.isActive,
          managerId: draft.managerId,
          memberName: editingManagerName,
          managerName: editingManagerName,
        };

        const savedBrand =
          toBrandDetailData(
            response as unknown as BrandResponse,
            fallbackBrand,
          );

        setBrand(savedBrand);
        setDraft(createDraft(savedBrand));

        setBrandIconFile(null);
        setBrandBackgroundFile(null);

        setBrandIconError(null);
        setBrandBackgroundImageError(null);

        if (brandIconInputRef.current) {
          brandIconInputRef.current.value =
            "";
        }

        if (
          brandBackgroundInputRef.current
        ) {
          brandBackgroundInputRef.current.value =
            "";
        }

        setIsEditing(false);
      } catch (error: unknown) {
        setError(
          new Error(
            getErrorMessage(error),
          ),
        );
      } finally {
        setSaving(false);
      }
    }, [
      resolvedBrandId,
      saving,
      draft,
      brand,
      editingManagerName,
      uploadBrandAssets,
      validateSelectedImagesBeforeSave,
    ]);

  const statusBadgeClass =
    useMemo(() => {
      return brand.status === "アクティブ"
        ? "inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700 text-xs font-semibold"
        : "inline-flex items-center px-2 py-1 rounded-full bg-slate-50 text-slate-500 text-xs font-semibold";
    }, [brand.status]);

  return {
    brand,
    setBrand,

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

    managerCandidates,
    loadingMembers,
    memberError,

    editingManagerName,
    handleSelectManager,

    brandImageAccept:
      BRAND_IMAGE_ACCEPT,

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