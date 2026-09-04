// frontend/amol/src/features/payment/context/PaymentCheckoutContext.tsx

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from "react";

import type { CartDisplayItem } from "../../shared/types/cart";

export type PaymentCheckoutDraft = {
  paymentPath: string;
  orderId: string;
  cartItems: CartDisplayItem[];
  selectedPaymentMethodId: string;
};

type PaymentCheckoutContextValue = {
  draft: PaymentCheckoutDraft | null;
  setDraft: Dispatch<SetStateAction<PaymentCheckoutDraft | null>>;
  updateDraft: (patch: Partial<PaymentCheckoutDraft>) => void;
  clearDraft: () => void;
};

const PaymentCheckoutContext =
  createContext<PaymentCheckoutContextValue | null>(null);

type PaymentCheckoutProviderProps = {
  children: ReactNode;
};

export function PaymentCheckoutProvider({
  children,
}: PaymentCheckoutProviderProps) {
  const [draft, setDraft] =
    useState<PaymentCheckoutDraft | null>(null);

  const updateDraft = useCallback(
    (patch: Partial<PaymentCheckoutDraft>) => {
      setDraft((current) => {
        if (!current) {
          return current;
        }

        return {
          ...current,
          ...patch,
        };
      });
    },
    [],
  );

  const clearDraft = useCallback(() => {
    setDraft(null);
  }, []);

  const value = useMemo<PaymentCheckoutContextValue>(
    () => ({
      draft,
      setDraft,
      updateDraft,
      clearDraft,
    }),
    [
      draft,
      updateDraft,
      clearDraft,
    ],
  );

  return (
    <PaymentCheckoutContext.Provider value={value}>
      {children}
    </PaymentCheckoutContext.Provider>
  );
}

export function usePaymentCheckout(): PaymentCheckoutContextValue {
  const context = useContext(PaymentCheckoutContext);

  if (!context) {
    throw new Error(
      "usePaymentCheckout must be used within PaymentCheckoutProvider.",
    );
  }

  return context;
}