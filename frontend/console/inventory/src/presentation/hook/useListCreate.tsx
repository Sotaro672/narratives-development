// frontend/console/inventory/src/presentation/hook/useListCreate.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
  type NavigateFunction,
} from "react-router-dom";

import { usePriceCard } from "../../../../list/presentation/hook/usePriceCard";
import { useAdminCard } from "../../../../admin/src/presentation/hook/useAdminCard";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  buildAfterCreatePath,
  buildBackPath,
  buildInventoryListCreatePath,
  canFetchListCreate,
  createListWithImages,
  extractDisplayStrings,
  getInventoryIdFromDTO,
  loadListCreateDTOFromParams,
  resolveListCreateParams,
  shouldRedirectToInventoryIdRoute,
  type ListCreateRouteParams,
  type PriceRow,
  type ResolvedListCreateParams,
} from "../../application/listCreate/listCreateService";

import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";

type ImageInputRef = React.RefObject<HTMLInputElement | null>;

type ListingDecision = "list" | "hold";

type AssigneeCandidate = {
  id: string;
  name: string;
};

export type UseListCreateResult = {
  title: string;
  onBack: () => void;
  onCreate: () => void;

  // dto
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;

  // display strings
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;

  // price
  priceRows: PriceRow[];
  onChangePrice: (index: number, price: number | null) => void;
  priceCard: ReturnType<typeof usePriceCard>;

  // listing local states
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;

  // images
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef;
  onAddImages: (files: FileList | null) => void;

  // assignee
  assigneeName: string;
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;
  handleSelectAssignee: (id: string) => void;

  // decision
  decision: ListingDecision;
  setDecision: React.Dispatch<React.SetStateAction<ListingDecision>>;
};

type UsePriceRowsResult = {
  priceRows: PriceRow[];
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  onChangePrice: (index: number, price: number | null) => void;
  priceCard: ReturnType<typeof usePriceCard>;
};

function getMemberUid(member: unknown): string {
  const m = member as any;

  return String(m?.uid ?? "");
}

function getMemberDisplayName(member: unknown): string {
  const m = member as any;

  const fullName = String(m?.fullName ?? "");
  if (fullName) return fullName;

  const nameParts = [m?.lastName, m?.firstName]
    .map((v) => String(v ?? ""))
    .filter(Boolean);

  const joinedName = nameParts.join(" ");
  if (joinedName) return joinedName;

  const email = String(m?.email ?? "");
  if (email) return email;

  const uid = getMemberUid(member);
  if (uid) return uid;

  return String(m?.id ?? "");
}

function normalizeAssigneeCandidates(
  rawCandidates: unknown,
): AssigneeCandidate[] {
  const rows = Array.isArray(rawCandidates) ? rawCandidates : [];

  return rows
    .map((raw) => {
      const c = raw as any;

      const id = String(c?.uid ?? c?.id ?? "");
      if (!id) return null;

      const nameParts = [c?.lastName, c?.firstName]
        .map((v) => String(v ?? ""))
        .filter(Boolean);

      const joinedName = nameParts.join(" ");

      const name =
        String(c?.name ?? "") ||
        String(c?.fullName ?? "") ||
        joinedName ||
        String(c?.email ?? "") ||
        id;

      return {
        id,
        name,
      };
    })
    .filter(Boolean) as AssigneeCandidate[];
}

function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(
    prev.map((file: File) => `${file.name}__${file.size}__${file.lastModified}`),
  );

  const filtered = add.filter(
    (file: File) =>
      !exists.has(`${file.name}__${file.size}__${file.lastModified}`),
  );

  return [...prev, ...filtered];
}

function normalizeImageFiles(
  files: FileList | File[] | null | undefined,
): File[] {
  return Array.from(files ?? [])
    .filter(Boolean)
    .filter((file: File) =>
      String(file.type || "").startsWith("image/"),
    ) as File[];
}

function useListCreateParamsAndTitle(): {
  resolvedParams: ResolvedListCreateParams;
  inventoryId: string | undefined;
  title: string;
} {
  const params = useParams<ListCreateRouteParams>();

  const resolvedParams: ResolvedListCreateParams = React.useMemo(
    () => resolveListCreateParams(params),
    [params],
  );

  const { inventoryId } = resolvedParams;

  const title = "出品作成";

  return {
    resolvedParams,
    inventoryId,
    title,
  };
}

function useListingDecision(): {
  decision: ListingDecision;
  setDecision: React.Dispatch<React.SetStateAction<ListingDecision>>;
} {
  const [decision, setDecision] = React.useState<ListingDecision>("list");

  return {
    decision,
    setDecision,
  };
}

function useListingFields(): {
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;
} {
  const [listingTitle, setListingTitle] = React.useState<string>("");
  const [description, setDescription] = React.useState<string>("");

  return {
    listingTitle,
    setListingTitle,
    description,
    setDescription,
  };
}

function useListingImages(): {
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef;
  onSelectImages: (files: FileList | null) => void;
  onDropImages: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages: (e: React.DragEvent<HTMLDivElement>) => void;
  removeImageAt: (idx: number) => void;
  clearImages: () => void;
} {
  const [images, setImages] = React.useState<File[]>([]);
  const [mainImageIndex, setMainImageIndex] = React.useState<number>(0);
  const [imagePreviewUrls, setImagePreviewUrls] = React.useState<string[]>([]);

  const imageInputRef = React.useRef<HTMLInputElement | null>(null);

  const appendImages = React.useCallback(
    (filesLike: FileList | File[] | null) => {
      const files = normalizeImageFiles(filesLike);
      if (files.length === 0) return;

      setImages((prev) => dedupeFiles(prev, files));
    },
    [],
  );

  const onSelectImages = React.useCallback(
    (files: FileList | null) => {
      appendImages(files);
    },
    [appendImages],
  );

  const onDropImages = React.useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      appendImages(e.dataTransfer.files);
    },
    [appendImages],
  );

  const onDragOverImages = React.useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
    },
    [],
  );

  const removeImageAt = React.useCallback((idx: number) => {
    setImages((prev) => prev.filter((_, i) => i !== idx));

    setMainImageIndex((prevMain) => {
      if (idx === prevMain) return 0;
      if (idx < prevMain) return Math.max(0, prevMain - 1);
      return prevMain;
    });
  }, []);

  const clearImages = React.useCallback(() => {
    setImages([]);
    setMainImageIndex(0);
  }, []);

  React.useEffect(() => {
    if (images.length === 0) {
      setImagePreviewUrls([]);
      return;
    }

    const urls = images.map((file: File) => URL.createObjectURL(file));
    setImagePreviewUrls(urls);

    return () => {
      urls.forEach((url: string) => {
        try {
          URL.revokeObjectURL(url);
        } catch {
          // noop
        }
      });
    };
  }, [images]);

  React.useEffect(() => {
    if (images.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }

    if (mainImageIndex < 0 || mainImageIndex > images.length - 1) {
      setMainImageIndex(0);
    }
  }, [images.length, mainImageIndex]);

  return {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,
  };
}

function usePriceRows(): UsePriceRowsResult {
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);

  const initializedPriceRowsRef = React.useRef(false);

  const onChangePrice = React.useCallback(
    (index: number, price: number | null) => {
      setPriceRows((prev) => {
        const next = [...prev];
        if (!next[index]) return prev;

        next[index] = {
          ...next[index],
          price,
        };

        return next;
      });
    },
    [],
  );

  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows,
    mode: "edit",
    currencySymbol: "¥",
    onChangePrice,
  });

  return {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  };
}

function useListCreateNavigation(
  resolvedParams: ResolvedListCreateParams,
): {
  navigate: NavigateFunction;
  onBack: () => void;
} {
  const navigate = useNavigate();

  const onBack = React.useCallback(() => {
    navigate(buildBackPath(resolvedParams));
  }, [navigate, resolvedParams]);

  return {
    navigate,
    onBack,
  };
}

function useListCreateDTO(args: {
  navigate: NavigateFunction;
  inventoryId: string | undefined;
  resolvedParams: ResolvedListCreateParams;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
}): {
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  const {
    navigate,
    inventoryId,
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  } = args;

  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  const redirectedRef = React.useRef(false);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      const canFetch = canFetchListCreate(resolvedParams);
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await loadListCreateDTOFromParams(resolvedParams);
        if (cancelled) return;

        const gotInventoryId = getInventoryIdFromDTO(data);
        const currentInventoryId = String(inventoryId ?? "");

        if (
          shouldRedirectToInventoryIdRoute({
            currentInventoryId,
            gotInventoryId,
            alreadyRedirected: redirectedRef.current,
          })
        ) {
          redirectedRef.current = true;
          navigate(buildInventoryListCreatePath(gotInventoryId), {
            replace: true,
          });
        }

        setDTO(data);

        if (!initializedPriceRowsRef.current) {
          setPriceRows(data.priceRows);
          initializedPriceRowsRef.current = true;
        }
      } catch (e) {
        if (cancelled) return;

        const msg = String(e instanceof Error ? e.message : e);
        setDTOError(msg);
      } finally {
        if (cancelled) return;
        setLoadingDTO(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [
    navigate,
    inventoryId,
    resolvedParams,
    setPriceRows,
    initializedPriceRowsRef,
  ]);

  const { productBrandName, productName, tokenBrandName, tokenName } =
    React.useMemo(() => extractDisplayStrings(dto), [dto]);

  return {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  };
}

function useCreateList(args: {
  navigate: NavigateFunction;
  resolvedParams: ResolvedListCreateParams;
  decision: ListingDecision;
  listingTitle: string;
  description: string;
  priceRows: PriceRow[];
  assigneeId: string | undefined;
  images: File[];
  mainImageIndex: number;
}): {
  onCreate: () => Promise<void>;
} {
  const {
    navigate,
    resolvedParams,
    decision,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
  } = args;

  const onCreate = React.useCallback(async () => {
    let imageUploadFailedMessage = "";

    try {
      if (images.length === 0) {
        const msg = "商品画像は1枚以上必須です。画像を追加してください。";
        alert(msg);
        throw new Error(msg);
      }

      const inventoryId = String(resolvedParams.inventoryId ?? "");

      await createListWithImages({
        params: {
          ...resolvedParams,
          inventoryId,
        },
        listingTitle,
        description,
        priceRows,
        decision,
        assigneeId,
        images,
        mainImageIndex,
        onImageUploadFailed: (message) => {
          imageUploadFailedMessage = message;
        },
      });

      if (imageUploadFailedMessage) {
        alert(imageUploadFailedMessage);
      } else {
        alert("作成しました");
      }

      navigate(buildAfterCreatePath(resolvedParams));
    } catch (e) {
      const msg = String(e instanceof Error ? e.message : e);
      alert(msg);
    }
  }, [
    assigneeId,
    decision,
    description,
    images,
    listingTitle,
    mainImageIndex,
    navigate,
    priceRows,
    resolvedParams,
  ]);

  return {
    onCreate,
  };
}

export function useListCreate(): UseListCreateResult {
  const { resolvedParams, inventoryId, title } = useListCreateParamsAndTitle();
  const { currentMember } = useAuth();

  const { decision, setDecision } = useListingDecision();

  const { listingTitle, setListingTitle, description, setDescription } =
    useListingFields();

  const {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
  } = useListingImages();

  const {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  } = usePriceRows();

  const { navigate, onBack } = useListCreateNavigation(resolvedParams);

  const { assigneeCandidates: rawAssigneeCandidates, loadingMembers } =
    useAdminCard();

  const assigneeCandidates = React.useMemo(
    () => normalizeAssigneeCandidates(rawAssigneeCandidates),
    [rawAssigneeCandidates],
  );

  const [assigneeId, setAssigneeId] = React.useState("");
  const [assigneeName, setAssigneeName] = React.useState("");

  React.useEffect(() => {
    if (!currentMember) return;
    if (assigneeId) return;

    const memberUid = getMemberUid(currentMember);
    if (!memberUid) return;

    const label = getMemberDisplayName(currentMember);

    setAssigneeId(memberUid);
    setAssigneeName(label);
  }, [currentMember, assigneeId]);

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "");
      if (!nextId) return;

      const matched = assigneeCandidates.find((c) => c.id === nextId);

      let nextName = "";
      if (matched) {
        nextName = matched.name;
      } else if (getMemberUid(currentMember) === nextId) {
        nextName = getMemberDisplayName(currentMember);
      } else {
        nextName = nextId;
      }

      setAssigneeId(nextId);
      setAssigneeName(nextName);
    },
    [assigneeCandidates, currentMember],
  );

  const {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  } = useListCreateDTO({
    navigate,
    inventoryId,
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  });

  const { onCreate } = useCreateList({
    navigate,
    resolvedParams,
    decision,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
  });

  return {
    title,
    onBack,
    onCreate,

    dto,
    loadingDTO,
    dtoError,

    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    priceRows,
    onChangePrice,
    priceCard,

    listingTitle,
    setListingTitle,
    description,
    setDescription,

    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onAddImages: onSelectImages,

    assigneeName,
    assigneeCandidates,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}