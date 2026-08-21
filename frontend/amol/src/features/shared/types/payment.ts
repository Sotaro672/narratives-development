// frontend/amol/src/features/shared/types/payment.ts

import type {

  ShippingAddress,

} from "./shippingAddress";

export type PaymentContext = {
  uid?: string;
  avatarId?: string;
  userId?: string;
};

export type CreatedPayment = {
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

export type OrderShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

export type ShippingQuoteItemSnapshot = {
  listId: string;
  inventoryId: string;
  modelId: string;
  originShippingAddressId: string;
  destinationShippingAddressId: string;
  carrier: string;
  transportationId?: string;
  size: number;
  qty: number;
  unitAmount: number;
  amount: number;
  currency: string;
};

export type ShippingQuoteSnapshot = {
  items:ShippingQuoteItemSnapshot[];
  amount: number;
  currency: string;
};

export type CreatedOrder = {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  shippingSnapshot?:OrderShippingSnapshot;
  shippingQuoteSnapshot?:ShippingQuoteSnapshot;
  paid?: boolean;
  createdAt?: string;
};

export type CanonicalShippingAddress =

  ShippingAddress & {
    zipCode: string;
    state: string;
    city: string;
    street: string;
    street2: string;
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
  shippingAddressId: string;
  paymentMethodId: string;
  items:CreateOrderItemRequest[];
};

export type CreatePaymentRequest = {
  paymentId: string;
  paymentMethodId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  amount: number;
};