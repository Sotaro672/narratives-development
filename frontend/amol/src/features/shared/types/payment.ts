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

export type CreatedOrder = {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
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
  shippingSnapshot:
    OrderShippingSnapshot;
  paymentMethodId: string;
  items:
    CreateOrderItemRequest[];
};

export type CreatePaymentRequest = {
  paymentId: string;
  paymentMethodId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  amount: number;
};