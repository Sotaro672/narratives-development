// frontend/console/mintRequest/src/application/mapper/mintRequestOptionsMapper.ts

import { asNonEmptyString } from "../util/primitive";

import type {
  BrandOptionVM,
  TokenBlueprintOptionVM,
} from "../../presentation/viewModel/mintRequestDetail.vm";

const asOptionalString = (value: unknown): string | undefined => {
  const text = asNonEmptyString(value);
  return text || undefined;
};

const asBool = (value: unknown): boolean | undefined => {
  if (typeof value === "boolean") return value;

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    if (normalized === "true") return true;
    if (normalized === "false") return false;
  }

  return undefined;
};

export function toBrandOptionVM(input: unknown): BrandOptionVM | null {
  const id = asNonEmptyString((input as any)?.id);
  const name = asNonEmptyString((input as any)?.name);

  if (!id || !name) return null;

  return {
    id,
    name,
  };
}

export function toBrandOptionVMs(
  inputs: unknown[] | null | undefined,
): BrandOptionVM[] {
  return (inputs ?? [])
    .map((item) => toBrandOptionVM(item))
    .filter((item): item is BrandOptionVM => item !== null);
}

export function toTokenBlueprintOptionVM(
  input: unknown,
): TokenBlueprintOptionVM | null {
  const raw = input as any;

  const id = asNonEmptyString(raw?.id);

  const tokenName =
    asNonEmptyString(raw?.tokenName) || asNonEmptyString(raw?.name);

  const name = tokenName;

  const symbol = asNonEmptyString(raw?.symbol);
  const iconUrl = asOptionalString(raw?.iconUrl);

  if (!id || !name || !symbol) return null;

  return {
    id,

    // selector 表示用
    name,

    // TokenBlueprintCard 表示用
    tokenName,
    symbol,

    brandId: asOptionalString(raw?.brandId),
    brandName: asOptionalString(raw?.brandName),
    companyId: asOptionalString(raw?.companyId),
    description: asOptionalString(raw?.description),
    minted: asBool(raw?.minted),
    metadataUri: asOptionalString(raw?.metadataUri),

    iconUrl,
  };
}

export function toTokenBlueprintOptionVMs(
  inputs: unknown[] | null | undefined,
): TokenBlueprintOptionVM[] {
  return (inputs ?? [])
    .map((item) => toTokenBlueprintOptionVM(item))
    .filter((item): item is TokenBlueprintOptionVM => item !== null);
}