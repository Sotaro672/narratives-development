// frontend/console/brand/src/presentation/hook/useBrandCreate.ts
import {
  useState,
  useCallback,
  useMemo,
  useEffect,
  useRef,
  type ChangeEvent,
} from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { MemberFilter } from "../../../../member/src/domain/repository/memberRepository";
import type { Brand } from "../../domain/entity/brand";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";
import { uploadBrandAssetToFirebaseStorage } from "../../infrastructure/storage/brandAssetStorage";

import { MemberRepositoryHTTP } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";

const memberRepo = new MemberRepositoryHTTP();

function formatLastFirst(
  lastName?: string | null,
  firstName?: string | null,
) {
  const ln = String(lastName ?? "");
  const fn = String(firstName ?? "");
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
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
  const [brandBackgroundImage, setBrandBackgroundImage] = useState("");

  const [managerId, setManagerId] = useState<string | null>(null);

  const [nameError, setNameError] = useState<string | null>(null);
  const [managerIdError, setManagerIdError] = useState<string | null>(null);

  const [managerOptions, setManagerOptions] = useState<Member[]>([]);
  const [loadingManagers, setLoadingManagers] = useState(false);
  const [managerError, setManagerError] = useState<string | null>(null);

  const brandIconInputRef = useRef<HTMLInputElement | null>(null);
  const brandBackgroundInputRef = useRef<HTMLInputElement | null>(null);

  const [brandIconFile, setBrandIconFile] = useState<File | null>(null);
  const [brandBackgroundFile, setBrandBackgroundFile] = useState<File | null>(
    null,
  );

  const [brandIconPreviewUrl, setBrandIconPreviewUrl] = useState("");
  const [brandBackgroundPreviewUrl, setBrandBackgroundPreviewUrl] =
    useState("");

  const isActive = true;

  useEffect(() => {
    let cancelled = false;

    async function loadManagers() {
      try {
        setLoadingManagers(true);
        setManagerError(null);

        const filter: MemberFilter = {};
        const { items } = await memberRepo.list(
          { number: 1, perPage: 200, totalPages: 1 },
          filter,
        );

        if (cancelled) return;

        setManagerOptions(items);
        if (!managerId && items.length > 0) {
          setManagerId(items[0].id);
        }
      } catch (e: any) {
        if (!cancelled) {
          setManagerError(
            e?.message ?? "ブランド責任者候補の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) setLoadingManagers(false);
      }
    }

    void loadManagers();
    return () => {
      cancelled = true;
    };
  }, [managerId]);

  useEffect(() => {
    if (!brandIconFile) {
      setBrandIconPreviewUrl(brandIcon || "");
      return;
    }

    const url = URL.createObjectURL(brandIconFile);
    setBrandIconPreviewUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [brandIconFile, brandIcon]);

  useEffect(() => {
    if (!brandBackgroundFile) {
      setBrandBackgroundPreviewUrl(brandBackgroundImage || "");
      return;
    }

    const url = URL.createObjectURL(brandBackgroundFile);
    setBrandBackgroundPreviewUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [brandBackgroundFile, brandBackgroundImage]);

  const selectedManager = useMemo(
    () => managerOptions.find((m) => m.id === managerId) ?? null,
    [managerOptions, managerId],
  );

  const managerDisplayName = useMemo(() => {
    if (!selectedManager) return "責任者未設定";
    return (
      formatLastFirst(selectedManager.lastName, selectedManager.firstName) ||
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
    () => Boolean(brandBackgroundFile || brandBackgroundImage),
    [brandBackgroundFile, brandBackgroundImage],
  );

  const handleBack = useCallback(() => {
    navigate("/brand");
  }, [navigate]);

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
      setBrandIcon("");
    },
    [],
  );

  const handleBrandBackgroundChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0] ?? null;
      setBrandBackgroundFile(file);
      setBrandBackgroundImage("");
    },
    [],
  );

  const handleClearBrandIcon = useCallback(() => {
    setBrandIconFile(null);
    setBrandIcon("");
    if (brandIconInputRef.current) {
      brandIconInputRef.current.value = "";
    }
  }, []);

  const handleClearBrandBackground = useCallback(() => {
    setBrandBackgroundFile(null);
    setBrandBackgroundImage("");
    if (brandBackgroundInputRef.current) {
      brandBackgroundInputRef.current.value = "";
    }
  }, []);

  const uploadBrandAssets = useCallback(
    async (brandId: string) => {
      let uploadedBrandIcon = brandIcon || "";
      let uploadedBrandBackgroundImage = brandBackgroundImage || "";

      if (brandIconFile) {
        const uploaded = await uploadBrandAssetToFirebaseStorage({
          companyId,
          brandId,
          target: "brandIcon",
          file: brandIconFile,
        });

        uploadedBrandIcon = uploaded.downloadUrl;
      }

      if (brandBackgroundFile) {
        const uploaded = await uploadBrandAssetToFirebaseStorage({
          companyId,
          brandId,
          target: "brandBackgroundImage",
          file: brandBackgroundFile,
        });

        uploadedBrandBackgroundImage = uploaded.downloadUrl;
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
    const normalizedName = String(name ?? "");
    const normalizedManagerId = String(managerId ?? "");

    let hasError = false;

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
      alert("ブランド名とブランド責任者を入力してください。");
      return;
    }

    if (!companyId) {
      alert("companyId が取得できません。");
      return;
    }

    try {
      const createPayload: Omit<Brand, "createdAt" | "updatedAt"> = {
        id: "",
        companyId,
        name: normalizedName,
        description: description || "",
        websiteUrl: websiteUrl || "",
        brandIcon: brandIcon || "",
        brandBackgroundImage: brandBackgroundImage || "",
        isActive: true,
        managerId: normalizedManagerId,
        walletAddress: "pending",
        createdBy: (currentMember?.id ?? null) as any,
        updatedBy: null as any,
        deletedAt: null as any,
        deletedBy: null as any,
      } as any;

      const created = await brandRepositoryHTTP.create(createPayload);

      const createdBrandId = String(created?.id ?? "");
      if (!createdBrandId) {
        throw new Error("brandId が取得できません。");
      }

      const { uploadedBrandIcon, uploadedBrandBackgroundImage } =
        await uploadBrandAssets(createdBrandId);

      if (uploadedBrandIcon || uploadedBrandBackgroundImage) {
        await brandRepositoryHTTP.update(createdBrandId, {
          brandIcon: uploadedBrandIcon || "",
          brandBackgroundImage: uploadedBrandBackgroundImage || "",
        });
      }

      alert("ブランドを登録しました。");
      navigate("/brand");
    } catch (e: any) {
      alert(`ブランド登録に失敗しました: ${e?.message ?? e}`);
    }
  }, [
    companyId,
    name,
    description,
    websiteUrl,
    brandIcon,
    brandBackgroundImage,
    managerId,
    currentMember?.id,
    navigate,
    uploadBrandAssets,
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
    hasBrandIconSelection,
    hasBrandBackgroundSelection,

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

    isActive,
    handleBack,
    handleSave,
  };
}