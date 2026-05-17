//frontend\amol\src\features\payment\utils\format.ts
import type { UserProfile } from "../../shipping-address/types";
import type { CanonicalShippingAddress, PaymentMethod } from "../types";

export function formatCardBrand(brand: string): string {
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

export function formatCardholderName(method: PaymentMethod): string {
  return method.cardholderName.trim() || "-";
}

export function formatCardLast4(method: PaymentMethod): string {
  return method.last4 ? `•••• ${method.last4}` : "-";
}

export function formatCardExpiry(method: PaymentMethod): string {
  if (!method.expMonth || !method.expYear) {
    return "-";
  }

  return `${String(method.expMonth).padStart(2, "0")}/${method.expYear}`;
}

export function getUserFullName(userProfile: UserProfile | null): string {
  if (!userProfile) {
    return "";
  }

  const lastName =
    "last_name" in userProfile && typeof userProfile.last_name === "string"
      ? userProfile.last_name
      : "";

  const firstName =
    "first_name" in userProfile && typeof userProfile.first_name === "string"
      ? userProfile.first_name
      : "";

  return `${lastName} ${firstName}`.trim();
}

export function getShippingAddressLabel(
  address: CanonicalShippingAddress,
): string {
  const zipLine = address.zipCode ? `〒${address.zipCode}` : "";
  const addressLine =
    `${address.state}${address.city}${address.street}${address.street2}`.trim();

  return [zipLine, addressLine].filter(Boolean).join("\n");
}