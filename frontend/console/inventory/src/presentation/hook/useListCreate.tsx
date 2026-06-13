// frontend/console/inventory/src/presentation/hook/useListCreate.tsx

import * as React from "react";
import type { UseListCreateResult } from "./listCreate/types";
import { useListCreateParamsAndTitle } from "./listCreate/useListCreateParamsAndTitle";
import { useListingDecision } from "./listCreate/useListingDecision";
import { useListingFields } from "./listCreate/useListingFields";
import { useListingImages } from "./listCreate/useListingImages";
import { usePriceRows } from "./listCreate/usePriceRows";
import { useListCreateNavigation } from "./listCreate/useListCreateNavigation";
import { useListCreateDTO } from "./listCreate/useListCreateDTO";
import { useCreateList } from "./listCreate/useCreateList";
import { useAdminCard } from "../../../../admin/src/presentation/hook/useAdminCard";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

type AssigneeCandidate = {
  id: string; // uid を入れる
  name: string;
};

type ProductBlueprintCategoryKind = "apparel" | "alcohol";

function getMemberUid(member: unknown): string {
  const m = member as any;

  return m?.uid ?? "";
}

function getMemberDisplayName(member: unknown): string {
  const m = member as any;

  const fullName = m?.fullName;
  if (fullName) return fullName;

  const nameParts = [m?.lastName, m?.firstName].filter(Boolean);
  const joinedName = nameParts.join(" ");
  if (joinedName) return joinedName;

  if (m?.email) return m.email;

  const uid = getMemberUid(member);
  if (uid) return uid;

  return m?.id ?? "";
}

function normalizeAssigneeCandidates(
  rawCandidates: unknown,
): AssigneeCandidate[] {
  const rows = Array.isArray(rawCandidates) ? rawCandidates : [];

  return rows
    .map((raw) => {
      const c = raw as any;

      const id = c?.uid || c?.id;
      if (!id) return null;

      const nameParts = [c?.lastName, c?.firstName].filter(Boolean);
      const joinedName = nameParts.join(" ");

      const name = c?.name || c?.fullName || joinedName || c?.email || id;

      return {
        id,
        name,
      };
    })
    .filter(Boolean) as AssigneeCandidate[];
}

function isProductBlueprintCategoryKind(
  value: unknown,
): value is ProductBlueprintCategoryKind {
  return value === "apparel" || value === "alcohol";
}

function resolveProductBlueprintCategory(
  dto: unknown,
): ProductBlueprintCategoryKind | undefined {
  const d = dto as any;

  const categoryKind = d?.productBlueprintCategory?.kind;
  if (isProductBlueprintCategoryKind(categoryKind)) {
    return categoryKind;
  }

  const priceRows = Array.isArray(d?.priceRows) ? d.priceRows : [];

  const firstKind = priceRows.find((row: any) =>
    isProductBlueprintCategoryKind(row?.kind),
  )?.kind;

  if (isProductBlueprintCategoryKind(firstKind)) {
    return firstKind;
  }

  return undefined;
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
    setProductBlueprintCategory,
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
      const nextId = id;
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

  React.useEffect(() => {
    const nextCategory = resolveProductBlueprintCategory(dto);
    setProductBlueprintCategory(nextCategory);
  }, [dto, setProductBlueprintCategory]);

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