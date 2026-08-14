// frontend/amol/src/features/payment/utils/guards.ts

import type { CreatedPayment } from "../../shared/types/payment";

export function isPaymentSucceeded(payment: CreatedPayment): boolean {
  return payment.status === "succeeded";
}

export function isPaymentRequiresAction(payment: CreatedPayment): boolean {
  return payment.requiresAction === true;
}