// frontend/amol/src/features/brand/types/brandPage.ts

import type {
  BrandDetail,
  BrandListItem,
} from "./brand";

export type BrandPageIdleState = {
  status: "idle";
  brand: null;
  listItems: BrandListItem[];
  error: "";
};

export type BrandPageLoadingState = {
  status: "loading";
  brand: null;
  listItems: BrandListItem[];
  error: "";
};

export type BrandPageSuccessState = {
  status: "success";
  brand: BrandDetail;
  listItems: BrandListItem[];
  error: "";
};

export type BrandPageErrorState = {
  status: "error";
  brand: null;
  listItems: BrandListItem[];
  error: string;
};

export type BrandPageState =
  | BrandPageIdleState
  | BrandPageLoadingState
  | BrandPageSuccessState
  | BrandPageErrorState;