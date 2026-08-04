// frontend/amol/src/features/cart/types/cartPage.ts

import type {
  CartDisplayItem,
} from "./cart";

export type CartPageStatus =
  | "idle"
  | "loading"
  | "success"
  | "error";

export type CartPageState = {
  status: CartPageStatus;
  items: CartDisplayItem[];
  error: string;
};