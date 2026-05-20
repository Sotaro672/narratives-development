//frontend\amol\src\features\payment-method\utils\paymentMethodUtils.ts
import type {
  PaymentMethodDefaultResponse,
  PaymentMethodListResponse,
  SetupIntentResponse,
  StripeConfigResponse,
  CardPaymentMethod,
} from "../types";

export function cardBrandLabel(brand: string): string {
  if (!brand) {
    return "カード";
  }

  switch (brand.toLowerCase()) {
    case "visa":
      return "Visa";
    case "mastercard":
      return "Mastercard";
    case "amex":
      return "American Express";
    case "jcb":
      return "JCB";
    case "diners":
      return "Diners Club";
    case "discover":
      return "Discover";
    default:
      return brand;
  }
}

export function getBackendUrl(): string {
  const value = import.meta.env.VITE_API_BASE_URL;

  if (typeof value === "string" && value.trim() !== "") {
    return value.replace(/\/+$/, "");
  }

  return "";
}

export async function readJsonResponse<T>(
  response: Response,
): Promise<T | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as T;
}

export function extractSetupIntentClientSecret(
  responseBody: SetupIntentResponse | null,
): string {
  const clientSecret =
    responseBody?.data?.clientSecret ?? responseBody?.clientSecret ?? "";

  return typeof clientSecret === "string" ? clientSecret.trim() : "";
}

export function extractSetupIntentStripeCustomerId(
  responseBody: SetupIntentResponse | null,
): string {
  const stripeCustomerId =
    responseBody?.data?.stripeCustomerId ?? responseBody?.stripeCustomerId ?? "";

  return typeof stripeCustomerId === "string" ? stripeCustomerId.trim() : "";
}

export function selectPrimaryPaymentMethod(
  listResponse: PaymentMethodListResponse | null,
  defaultResponse: PaymentMethodDefaultResponse | null,
): CardPaymentMethod | null {
  if (defaultResponse?.data) {
    return defaultResponse.data;
  }

  const items = Array.isArray(listResponse?.data) ? listResponse.data : [];

  return items.find((item) => item.isDefault) ?? items[0] ?? null;
}

export function getStripePublishableKey(
  responseBody: StripeConfigResponse | null,
): string {
  const publishableKey = responseBody?.publishableKey ?? "";

  return typeof publishableKey === "string" ? publishableKey.trim() : "";
}