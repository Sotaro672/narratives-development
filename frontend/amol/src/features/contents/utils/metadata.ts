//frontend\amol\src\features\contents\utils\metadata.ts
import type {
  ContentsMetadata,
  ContentsMetadataFile,
} from "../types";

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function getString(
  value: Record<string, unknown>,
  key: string
): string {
  const raw = value[key];
  return typeof raw === "string" ? raw : "";
}

export function parseMetadataFile(
  value: unknown
): ContentsMetadataFile | null {
  if (!isRecord(value)) {
    return null;
  }

  const uri = getString(value, "uri");
  const type = getString(value, "type");
  const name = getString(value, "name");

  if (!uri) {
    return null;
  }

  return {
    name,
    type,
    uri,
  };
}

export function parseContentsMetadata(
  value: unknown
): ContentsMetadata | null {
  if (!isRecord(value)) {
    return null;
  }

  const properties = isRecord(value.properties) ? value.properties : null;
  const filesRaw =
    properties && Array.isArray(properties.files) ? properties.files : [];

  const files = filesRaw
    .map(parseMetadataFile)
    .filter((file): file is ContentsMetadataFile => file !== null);

  return {
    name: getString(value, "name"),
    symbol: getString(value, "symbol"),
    description: getString(value, "description"),
    image: getString(value, "image"),
    createdAt: getString(value, "created_at"),
    files,
  };
}