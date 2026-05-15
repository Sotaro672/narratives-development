// frontend/console/mintRequest/src/application/mapper/mintRequestOptionsMapper.ts

import { asNonEmptyString } from "./modelInspectionMapper";

import type {
  BrandOptionVM,
  TokenBlueprintOptionVM,
} from "../../presentation/viewModel/mintRequestDetail.vm";

export function toBrandOptionVM(input: unknown): BrandOptionVM | null {
  const id = String((input as any)?.id ?? "").trim();
  const name = String((input as any)?.name ?? "").trim();

  if (!id || !name) return null;

  return {
    id,
    name,
  };
}

export function toBrandOptionVMs(inputs: unknown[] | null | undefined): BrandOptionVM[] {
  return (inputs ?? [])
    .map((item) => toBrandOptionVM(item))
    .filter((item): item is BrandOptionVM => item !== null);
}

export function toTokenBlueprintOptionVM(
  input: unknown,
): TokenBlueprintOptionVM | null {
  const id = String((input as any)?.id ?? "").trim();
  const name = String((input as any)?.name ?? "").trim();
  const symbol = String((input as any)?.symbol ?? "").trim();
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