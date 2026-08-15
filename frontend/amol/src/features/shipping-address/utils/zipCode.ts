// frontend/amol/src/features/shipping-address/utils/zipCode.ts

import type { ShippingAddress, ZipCloudResponse } from "../../shared/types/shippingAddress";

export function getShippingAddressId(address: ShippingAddress | null): string {
  return address?.id ?? "";
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