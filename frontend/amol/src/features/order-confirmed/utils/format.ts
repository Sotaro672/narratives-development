//frontend\amol\src\features\order-confirmed\utils\format.ts
import type { ShippingAddress } from "../../shipping-address/types";

export function getShippingAddressLabel(
  address: ShippingAddress | null | undefined,
): string {
  if (!address) {
    return "";
  }

  const zipCode =
    "zipCode" in address && typeof address.zipCode === "string"
      ? address.zipCode
      : "zip_code" in address && typeof address.zip_code === "string"
        ? address.zip_code
        : "";

  const state =
    "state" in address && typeof address.state === "string"
      ? address.state
      : "";

  const city =
    "city" in address && typeof address.city === "string" ? address.city : "";

  const street =
    "street" in address && typeof address.street === "string"
      ? address.street
      : "";

  const street2 =
    "street2" in address && typeof address.street2 === "string"
      ? address.street2
      : "";

  const zipLine = zipCode ? `〒${zipCode}` : "";
  const addressLine = `${state}${city}${street}${street2}`.trim();

  return [zipLine, addressLine].filter(Boolean).join("\n");
}

export function getShippingAddressLines(
  address: ShippingAddress | null | undefined,
): string[] {
  const label = getShippingAddressLabel(address);

  if (!label) {
    return [];
  }

  return label.split("\n").filter(Boolean);
}

export function formatPaymentStatus(status?: string): string {
  const normalized = status?.trim().toUpperCase();

  switch (normalized) {
    case "SUCCEEDED":
      return "決済完了";
    case "PENDING":
      return "処理中";
    case "PROCESSING":
      return "処理中";
    case "REQUIRES_ACTION":
      return "追加認証待ち";
    case "FAILED":
      return "失敗";
    case "CANCELED":
      return "キャンセル";
    default:
      return normalized || "決済完了";
  }
}