// frontend/amol/src/features/payment/types.ts
import type { CartDisplayItem } from "../cart/types";
import type { ShippingAddress } from "../shipping-address/types";

export type PaymentMethod = {
  id: string;
  userId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type PaymentContext = {
  avatarId?: string;
  avatarUid?: string;
};

export type PaymentMethodListResponse = {
  data?: PaymentMethod[];
  error?: string;
};

export type PaymentMethodDefaultResponse = {
  data?: PaymentMethod | null;
  error?: string;
};

export type CreatedPayment = {
  id?: string;
  paymentId?: string;
  paymentMethodId?: string;
  stripeCustomerId?: string;
  stripePaymentMethodId?: string;
  stripePaymentIntentId?: string;
  amount?: number;
  status?: string;
  clientSecret?: string;
  requiresAction?: boolean;
  createdAt?: string;
};

export type CreatedOrder = {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  paid?: boolean;
  createdAt?: string;
};

export type CartItemType = "list" | "resale";

export type CanonicalCartDisplayItem = CartDisplayItem & {
  avatarId: string;
  type: CartItemType;

  // list item identifiers
  inventoryId?: string;
  listId?: string;
  modelId?: string;

  // resale item identifier
  resaleId?: string;

  // product identifiers used by cart display data
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;

  // display fields
  title?: string;
  productName?: string;
  listImage?: string;
  imageUrl?: string;

  price?: number;
  qty: number;
};

export type CanonicalShippingAddress = ShippingAddress & {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type OrderShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: "JP";
};

export type ListCreateOrderItemRequest = {
  type: "list";

  listId: string;
  modelId: string;
  qty: number;

  isCanceled: false;
  isDispatched: false;
};

export type ResaleCreateOrderItemRequest = {
  type: "resale";

  resaleId: string;
  qty: 1;

  isCanceled: false;
  isDispatched: false;
};

export type CreateOrderItemRequest =
  | ListCreateOrderItemRequest
  | ResaleCreateOrderItemRequest;

export type CreateOrderRequest = {
  id: string;
  shippingSnapshot: OrderShippingSnapshot;
  paymentMethodId: string;
  items: CreateOrderItemRequest[];
};

export type CreatePaymentRequest = {
  paymentId: string;
  paymentMethodId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  amount: number;
};