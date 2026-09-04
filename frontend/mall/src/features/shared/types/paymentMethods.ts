// frontend/amol/src/features/shared/types/paymentMethods.ts

import type { Stripe } from "@stripe/stripe-js";

export type CardPaymentMethod = {
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
  createdAt: string;
  updatedAt: string;
};

type DataResponse<T> = {
  data: T;
};

export type PaymentMethodListResponse = DataResponse<CardPaymentMethod[]>;
export type PaymentMethodDefaultResponse = DataResponse<CardPaymentMethod>;

export type SetupIntentData = {
  clientSecret: string;
  stripeCustomerId: string;
};

export type SetupIntentResponse = DataResponse<SetupIntentData>;

export type StripeConfigResponse = {
  publishableKey: string;
};

export type SavePaymentMethodResponse = DataResponse<CardPaymentMethod>;

export type ConfirmedCardPayload = {
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
};

export type PaymentMethodPageLocationState = {
  returnTo?: string;
  fromRoomPayment?: boolean;
  amount?: number;
  selectedPaymentMethod?: "card" | "paypay";
  shouldResumeCardPayment?: boolean;
};

export type UsePaymentMethodPageResult = {
  paymentMethod: CardPaymentMethod | null;
  cardholderName: string;
  isLoading: boolean;
  isCreatingIntent: boolean;
  clientSecret: string;
  stripeCustomerId: string;
  stripePromise: Promise<Stripe | null> | null;
  errorMessage: string;
  normalizedCardholderName: string;
  setCardholderName: (value: string) => void;
  handleCreateSetupIntent: () => Promise<void>;
  handleCompleted: (payload: ConfirmedCardPayload) => Promise<void>;
};