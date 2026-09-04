// frontend/amol/src/features/wallet/utils/tokenGuards.ts

import { isRecord } from "../../../components/utils/typeGuards";
import type { TokenMetadataAttributeDTO, TokenMetadataDTO } from "../../shared/types/tokenTypes";

export function toTokenMetadataDTO(value: unknown): TokenMetadataDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  const attributes = toTokenMetadataAttributes(value.attributes);

  return {
    name: getString(value, "name"),
    symbol: getString(value, "symbol"),
    description: getString(value, "description"),
    image: getString(value, "image"),
    externalUrl: getString(value, "external_url"),
    attributes,
    createdAt: getString(value, "created_at"),
    tokenBlueprintId: getTokenBlueprintId(attributes),
    raw: value,
  };
}

function toTokenMetadataAttributes(value: unknown): TokenMetadataAttributeDTO[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(isRecord)
    .map((item) => ({
      traitType: getString(item, "trait_type"),
      value: getString(item, "value"),
    }))
    .filter((item) => Boolean(item.traitType) || Boolean(item.value));
}

function getTokenBlueprintId(attributes: TokenMetadataAttributeDTO[]): string {
  const attribute = attributes.find((item) => item.traitType === "TokenBlueprintID");
  return attribute?.value || "";
}

function getString(value: Record<string, unknown>, key: string): string {
  const raw = value[key];
  return typeof raw === "string" ? raw : "";
}