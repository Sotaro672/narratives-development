// frontend/console/brand/src/presentation/hook/useBrandDetail.ts
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";
import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

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

export function useBrandDetail() {
  const navigate = useNavigate();
  const { brandId } = useParams<{ brandId: string }>();
  const resolvedBrandId = brandId ?? "";

  const [brand, setBrand] = useState<BrandDetailData>(() => ({
    id: resolvedBrandId,
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
  }));

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  const [draft, setDraft] = useState<BrandDraft>(() => ({
    name: "",
    description: "",
    websiteUrl: "",
    brandIcon: "",
    brandBackgroundImage: "",
    isActive: false,
    managerId: "",
  }));

  const brandIconInputRef = useRef<HTMLInputElement | null>(null);
  const brandBackgroundInputRef = useRef<HTMLInputElement | null>(null);

  const [brandIconFile, setBrandIconFile] = useState<File | null>(null);
  const [brandBackgroundFile, setBrandBackgroundFile] = useState<File | null>(
    null,
  );

  const [brandIconPreviewUrl, setBrandIconPreviewUrl] = useState("");
  const [brandBackgroundPreviewUrl, setBrandBackgroundPreviewUrl] =
    useState("");

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      if (!resolvedBrandId) return;

      try {
        setLoading(true);
        setError(null);

        const data: any = await brandRepositoryHTTP.getById(resolvedBrandId);

        const id = String(data?.id ?? "");
        const companyId = String(data?.companyId ?? "");
        const name = String(data?.name ?? "");
        const description = String(data?.description ?? "");
        const websiteUrl = data?.websiteUrl ?? "";
        const brandIcon = data?.brandIcon ?? "";
        const brandBackgroundImage = data?.brandBackgroundImage ?? "";
        const isActive = Boolean(data?.isActive ?? false);
        const managerId = String(data?.managerId ?? "");
        const memberName = data?.memberName ?? "";
        const walletAddress = String(data?.walletAddress ?? "");
        const createdAt = String(data?.createdAt ?? "");
        const createdBy = data?.createdBy ?? null;
        const updatedAtRaw = data?.updatedAt ?? null;
        const updatedBy = data?.updatedBy ?? null;
        const deletedAt = data?.deletedAt ?? null;
        const deletedBy = data?.deletedBy ?? null;

        const nextBrand: BrandDetailData = {
          id,
          companyId,
          name,
          description,
          websiteUrl,
          brandIcon,
          brandBackgroundImage,
          isActive,
          managerId,
          memberName: String(memberName ?? ""),
          managerName: String(memberName ?? ""),
          walletAddress,
          createdAt,
          createdBy,
          updatedAtRaw,
          updatedBy,
          deletedAt,
          deletedBy,
          status: isActive ? "アクティブ" : "停止",
          registeredAt: safeDateLabelJa(createdAt, ""),
          updatedAt: safeDateLabelJa(updatedAtRaw ?? "", ""),
        };

        if (!cancelled) {
          setBrand((prev) => ({ ...prev, ...nextBrand }));
          setDraft({
            name: nextBrand.name,
            description: nextBrand.description,
            websiteUrl: nextBrand.websiteUrl ?? "",
            brandIcon: nextBrand.brandIcon ?? "",
            brandBackgroundImage: nextBrand.brandBackgroundImage ?? "",
            isActive: nextBrand.isActive,
            managerId: nextBrand.managerId,
          });
          setBrandIconFile(null);
          setBrandBackgroundFile(null);
        }
      } catch (e: any) {
        const err = e instanceof Error ? e : new Error(String(e));
        console.error("[useBrandDetail] load error:", err);
        if (!cancelled) setError(err);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [resolvedBrandId]);

  useEffect(() => {
    if (!brandIconFile) {
      setBrandIconPreviewUrl(
        isEditing ? draft.brandIcon || "" : brand.brandIcon || "",
      );
      return;
    }

    const url = URL.createObjectURL(brandIconFile);
    setBrandIconPreviewUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [brandIconFile, draft.brandIcon, brand.brandIcon, isEditing]);

  useEffect(() => {
    if (!brandBackgroundFile) {
      setBrandBackgroundPreviewUrl(
        isEditing
          ? draft.brandBackgroundImage || ""
          : brand.brandBackgroundImage || "",
      );
      return;
    }

    const url = URL.createObjectURL(brandBackgroundFile);
    setBrandBackgroundPreviewUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [
    brandBackgroundFile,
    draft.brandBackgroundImage,
    brand.brandBackgroundImage,
    isEditing,
  ]);

  const handleBack = useCallback(() => {
    navigate("/brand");
  }, [navigate]);

  const handleEdit = useCallback(() => {
    setDraft({
      name: brand.name,
      description: brand.description,
      websiteUrl: brand.websiteUrl ?? "",
      brandIcon: brand.brandIcon ?? "",
      brandBackgroundImage: brand.brandBackgroundImage ?? "",
      isActive: brand.isActive,
      managerId: brand.managerId,
    });
    setBrandIconFile(null);
    setBrandBackgroundFile(null);
    setIsEditing(true);
  }, [brand]);

  const handleCancelEdit = useCallback(() => {
    setDraft({
      name: brand.name,
      description: brand.description,
      websiteUrl: brand.websiteUrl ?? "",
      brandIcon: brand.brandIcon ?? "",
      brandBackgroundImage: brand.brandBackgroundImage ?? "",
      isActive: brand.isActive,
      managerId: brand.managerId,
    });
    setBrandIconFile(null);
    setBrandBackgroundFile(null);
    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }
    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }
    setIsEditing(false);
  }, [brand]);

  const handlePickBrandIcon = useCallback(() => {
    brandIconInputRef.current?.click();
  }, []);

  const handlePickBrandBackground = useCallback(() => {
    brandBackgroundInputRef.current?.click();
  }, []);

  const handleBrandIconChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0] ?? null;
      setBrandIconFile(file);
      setDraft((prev) => ({
        ...prev,
        brandIcon: "",
      }));
    },
    [],
  );

  const handleBrandBackgroundChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0] ?? null;
      setBrandBackgroundFile(file);
      setDraft((prev) => ({
        ...prev,
        brandBackgroundImage: "",
      }));
    },
    [],
  );

  const handleClearBrandIcon = useCallback(() => {
    setBrandIconFile(null);
    setDraft((prev) => ({
      ...prev,
      brandIcon: "",
    }));
    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }
  }, []);

  const handleClearBrandBackground = useCallback(() => {
    setBrandBackgroundFile(null);
    setDraft((prev) => ({
      ...prev,
      brandBackgroundImage: "",
    }));
    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }
  }, []);

  const uploadBrandAssets = useCallback(async () => {
    if (!resolvedBrandId) {
      throw new Error("brandId が取得できません。");
    }

    let uploadedBrandIcon = draft.brandIcon || "";
    let uploadedBrandBackgroundImage = draft.brandBackgroundImage || "";

    if (brandIconFile) {
      const uploaded = await brandRepositoryHTTP.uploadBrandAsset({
        file: brandIconFile,
        target: "brandIcon",
        brandId: resolvedBrandId,
      });
      uploadedBrandIcon = uploaded.publicUrl || uploaded.objectPath || "";
    }

    if (brandBackgroundFile) {
      const uploaded = await brandRepositoryHTTP.uploadBrandAsset({
        file: brandBackgroundFile,
        target: "brandBackgroundImage",
        brandId: resolvedBrandId,
      });
      uploadedBrandBackgroundImage =
        uploaded.publicUrl || uploaded.objectPath || "";
    }

    return {
      uploadedBrandIcon,
      uploadedBrandBackgroundImage,
    };
  }, [
    draft.brandIcon,
    draft.brandBackgroundImage,
    brandIconFile,
    brandBackgroundFile,
    resolvedBrandId,
  ]);

  const handleSave = useCallback(async () => {
    if (!resolvedBrandId) return;

    try {
      setLoading(true);
      setError(null);

      const { uploadedBrandIcon, uploadedBrandBackgroundImage } =
        await uploadBrandAssets();

      const patch: any = {
        name: draft.name,
        description: draft.description,
        websiteUrl: draft.websiteUrl,
        brandIcon: uploadedBrandIcon,
        brandBackgroundImage: uploadedBrandBackgroundImage,
        isActive: draft.isActive,
        managerId: draft.managerId,
      };

      const saved: any = await brandRepositoryHTTP.update(resolvedBrandId, patch);

      const isActive = Boolean(saved?.isActive ?? false);
      const memberName = saved?.memberName ?? "";
      const updatedAtRaw = saved?.updatedAt ?? null;

      setBrand((prev) => ({
        ...prev,
        id: String(saved?.id ?? prev.id),
        companyId: String(saved?.companyId ?? prev.companyId),
        name: String(saved?.name ?? prev.name),
        description: String(saved?.description ?? prev.description),
        websiteUrl: saved?.websiteUrl ?? prev.websiteUrl ?? "",
        brandIcon: saved?.brandIcon ?? prev.brandIcon ?? "",
        brandBackgroundImage:
          saved?.brandBackgroundImage ?? prev.brandBackgroundImage ?? "",
        isActive,
        managerId: String(saved?.managerId ?? prev.managerId),
        memberName: String(memberName ?? prev.memberName ?? ""),
        managerName: String(memberName ?? prev.managerName ?? ""),
        walletAddress: String(saved?.walletAddress ?? prev.walletAddress),
        createdAt: String(saved?.createdAt ?? prev.createdAt),
        createdBy: saved?.createdBy ?? prev.createdBy ?? null,
        updatedAtRaw,
        updatedBy: saved?.updatedBy ?? prev.updatedBy ?? null,
        deletedAt: saved?.deletedAt ?? prev.deletedAt ?? null,
        deletedBy: saved?.deletedBy ?? prev.deletedBy ?? null,
        status: isActive ? "アクティブ" : "停止",
        registeredAt: safeDateLabelJa(saved?.createdAt ?? prev.createdAt ?? "", ""),
        updatedAt: safeDateLabelJa(updatedAtRaw ?? "", ""),
      }));

      setDraft({
        name: String(saved?.name ?? draft.name),
        description: String(saved?.description ?? draft.description),
        websiteUrl: saved?.websiteUrl ?? draft.websiteUrl,
        brandIcon: saved?.brandIcon ?? uploadedBrandIcon,
        brandBackgroundImage:
          saved?.brandBackgroundImage ?? uploadedBrandBackgroundImage,
        isActive: Boolean(saved?.isActive ?? draft.isActive),
        managerId: String(saved?.managerId ?? draft.managerId),
      });

      setBrandIconFile(null);
      setBrandBackgroundFile(null);
      if (brandIconInputRef.current) {
        brandIconInputRef.current.value = "";
      }
      if (brandBackgroundInputRef.current) {
        brandBackgroundInputRef.current.value = "";
      }

      setIsEditing(false);
    } catch (e: any) {
      const err = e instanceof Error ? e : new Error(String(e));
      console.error("[useBrandDetail] save error:", err);
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [resolvedBrandId, draft, uploadBrandAssets]);

  const statusBadgeClass = useMemo(() => {
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
    error,
    brandIconInputRef,
    brandBackgroundInputRef,
    brandIconFile,
    brandBackgroundFile,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,
    handlePickBrandIcon,
    handlePickBrandBackground,
    handleBrandIconChange,
    handleBrandBackgroundChange,
    handleClearBrandIcon,
    handleClearBrandBackground,
  };
}