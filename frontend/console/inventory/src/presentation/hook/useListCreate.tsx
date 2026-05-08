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

function s(value: unknown): string {
  return String(value ?? "").trim();
}

function getMemberUid(member: unknown): string {
  const m = member as any;

  return s(m?.uid);
}

function getMemberDisplayName(member: unknown): string {
  const m = member as any;

  return (
    s(m?.fullName) ||
    [s(m?.lastName), s(m?.firstName)].filter(Boolean).join(" ") ||
    s(m?.email) ||
    getMemberUid(member) ||
    s(m?.id)
  );
}

function normalizeAssigneeCandidates(rawCandidates: unknown): AssigneeCandidate[] {
  const rows = Array.isArray(rawCandidates) ? rawCandidates : [];

  return rows
    .map((raw) => {
      const c = raw as any;

      const id = s(c?.uid) || s(c?.id);
      const name =
        s(c?.name) ||
        s(c?.fullName) ||
        [s(c?.lastName), s(c?.firstName)].filter(Boolean).join(" ") ||
        s(c?.email) ||
        id;

      if (!id) return null;

      return {
        id,
        name,
      };
    })
    .filter(Boolean) as AssigneeCandidate[];
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
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,
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
      const nextId = s(id);
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
    onSelectImages,
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,

    assigneeName,
    assigneeCandidates,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}