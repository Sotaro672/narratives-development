// frontend/console/inventory/src/presentation/hook/listCreate/useListingFields.ts
import * as React from "react";

export function useListingFields(): {
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;
} {
  const [listingTitle, setListingTitle] = React.useState<string>("");
  const [description, setDescription] = React.useState<string>("");
  return { listingTitle, setListingTitle, description, setDescription };
}
