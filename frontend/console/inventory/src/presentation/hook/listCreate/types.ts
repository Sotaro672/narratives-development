// frontend/console/inventory/src/presentation/hook/listCreate/types.ts

import * as React from "react";

import type { usePriceCard } from "../../../../../list/presentation/hook/usePriceCard";
import type { PriceRow } from "../../../application/listCreate/listCreateService";
import type { ImageInputRef } from "../../../application/listCreate/listCreate.types";
import type { ListCreateDTO } from "../../../infrastructure/http/inventoryRepositoryHTTP";

export type ListingDecision = "list" | "hold";

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
  assigneeCandidates: Array<{ id: string; name: string }>;
  loadingMembers: boolean;
  handleSelectAssignee: (id: string) => void;

  // decision
  decision: ListingDecision;
  setDecision: React.Dispatch<React.SetStateAction<ListingDecision>>;
};