//frontend\amol\src\features\shipping-address\utils\zipCode.ts
import type {
  ErrorResponse,
  ShippingAddress,
  UserProfile,
  ZipCloudResponse,
} from "../types";

export function getShippingAddressId(
  address: ShippingAddress | null
): string {
  if (!address) return "";

  return address.id || address.shippingAddressId || address.ID || "";
}

export function isShippingAddress(value: unknown): value is ShippingAddress {
  if (!value || typeof value !== "object") return false;

  const address = value as Partial<ShippingAddress>;

  return (
    typeof address.zipCode === "string" &&
    typeof address.state === "string" &&
    typeof address.city === "string" &&
    typeof address.street === "string"
  );
}

export function isUserProfile(value: unknown): value is UserProfile {
  if (!value || typeof value !== "object") return false;

  return !("error" in value);
}

export function isErrorResponse(value: unknown): value is ErrorResponse {
  if (!value || typeof value !== "object") return false;

  return "error" in value;
}

export function normalizeZipCode(value: string): string {
  return value.replace(/[^\d]/g, "");
}

export function formatZipCode(value: string): string {
  const normalized = normalizeZipCode(value);

  if (normalized.length <= 3) {
    return normalized;
  }

  return `${normalized.slice(0, 3)}-${normalized.slice(3, 7)}`;
}

export function isZipCloudResponse(value: unknown): value is ZipCloudResponse {
  if (!value || typeof value !== "object") return false;

  const response = value as Partial<ZipCloudResponse>;

  return typeof response.status === "number" && "results" in response;
}