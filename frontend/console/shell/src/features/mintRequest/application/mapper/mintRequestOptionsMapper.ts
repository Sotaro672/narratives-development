// frontend/console/mintRequest/src/application/mapper/mintRequestOptionsMapper.ts

import { asNonEmptyString } from "../util/primitive";

import type {
  BrandOptionVM,
  TokenBlueprintOptionVM,
} from "../../presentation/viewModel/mintRequestDetail.vm";

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
  const id = asNonEmptyString((input as any)?.id);
  const name = asNonEmptyString((input as any)?.name);
  const symbol = asNonEmptyString((input as any)?.symbol);
  const iconUrl = asNonEmptyString((input as any)?.iconUrl) || undefined;

  if (!id || !name || !symbol) return null;

  return {
    id,
    name,
    symbol,
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