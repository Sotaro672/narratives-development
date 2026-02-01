// frontend/console/inventory/src/presentation/hook/useListCreate.tsx
import * as React from "react";

import type { UseListCreateResult } from "./listCreate/types";

import { useListCreateParamsAndTitle } from "./listCreate/useListCreateParamsAndTitle";
import { useListingDecision } from "./listCreate/useListingDecision";
import { useListingFields } from "./listCreate/useListingFields";
import { useListingImages } from "./listCreate/useListingImages";
import { usePriceRows } from "./listCreate/usePriceRows";
import { useListCreateNavigation } from "./listCreate/useListCreateNavigation";
import { useAssignee } from "./listCreate/useAssignee";
import { useListCreateDTO } from "./listCreate/useListCreateDTO";
import { useCreateList } from "./listCreate/useCreateList";

export function useListCreate(): UseListCreateResult {
  const { resolvedParams, inventoryId, title } = useListCreateParamsAndTitle();

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

  const { priceRows, setPriceRows, initializedPriceRowsRef, onChangePrice, priceCard } =
    usePriceRows();

  const { navigate, onBack } = useListCreateNavigation(resolvedParams);

  const {
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    assigneeId,
    handleSelectAssignee,
  } = useAssignee();

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
    assigneeCandidates: (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}
