// frontend/mall/src/features/identityVerification/types.ts

export const IDENTITY_VERIFICATION_STATUSES = [
  "unverified",
  "verified",
] as const;

export type IdentityVerificationStatus =
  (typeof IDENTITY_VERIFICATION_STATUSES)[number];

export const IDENTITY_VERIFICATION_METHODS = [
  "my_number_card",
] as const;

export type IdentityVerificationMethod =
  (typeof IDENTITY_VERIFICATION_METHODS)[number];

export const IDENTITY_VERIFICATION_PROVIDERS = [
  "mock",
] as const;

export type IdentityVerificationProvider =
  (typeof IDENTITY_VERIFICATION_PROVIDERS)[number];

export type IdentityVerification = {
  status: IdentityVerificationStatus;
  method?: IdentityVerificationMethod;
  provider?: IdentityVerificationProvider;
  verifiedAt?: string;
};

export function isIdentityVerified(
  verification: IdentityVerification | null | undefined,
): boolean {
  return verification?.status === "verified";
}