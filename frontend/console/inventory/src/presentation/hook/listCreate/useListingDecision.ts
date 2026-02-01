// frontend/console/inventory/src/presentation/hook/listCreate/useListingDecision.ts
import * as React from "react";
import type { ListingDecision } from "./types";

export function useListingDecision(): {
  decision: ListingDecision;
  setDecision: React.Dispatch<React.SetStateAction<ListingDecision>>;
} {
  const [decision, setDecision] = React.useState<ListingDecision>("list");
  return { decision, setDecision };
}
