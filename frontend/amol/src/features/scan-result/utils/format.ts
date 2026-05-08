// frontend/amol/src/features/scan-result/utils/format.ts
import type {
  MallOwnerInfo,
  MallPreviewTransferInfo,
  ProductBlueprintPatchItem,
  TokenBlueprintPatchVM,
} from "../types";

export function trimText(value: unknown): string {
  return String(value ?? "").trim();
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function ownerLabel(owner: MallOwnerInfo | null | undefined): string {
  if (!owner) return "-";

  const avatarName = owner.avatarName.trim();
  const brandName = owner.brandName.trim();
  const avatarId = owner.avatarId.trim();
  const brandId = owner.brandId.trim();

  if (avatarName) return avatarName;
  if (brandName) return brandName;
  if (avatarId) return avatarId;
  if (brandId) return brandId;

  return "-";
}

export function shortAddress(value: string): string {
  const s = value.trim();
  if (s.length <= 16) return s;
  return `${s.slice(0, 8)}...${s.slice(-8)}`;
}

export function withCm(value: unknown): string {
  const s = trimText(value);
  if (!s) return "-";
  if (/\s*cm$/i.test(s)) return s;
  return `${s}cm`;
}

export function safeUrl(raw: string): string {
  const s = raw.trim();
  if (!s) return "";

  try {
    const u = new URL(s);
    if (u.protocol && u.host) return u.toString();
  } catch {
    // noop
  }

  return encodeURI(s);
}

export function shouldHidePatchKey(rawKey: string): boolean {
  const key = rawKey.trim();
  if (!key) return true;

  const hidden = new Set(["assigneeId", "brandId"]);
  if (hidden.has(key)) return true;

  const keyParts = key.split(".");
  const tail = keyParts[keyParts.length - 1] || "";
  const tailNoIndex = tail.replace(/\[\d+\]/g, "");

  return hidden.has(tailNoIndex);
}

export function jpLabelForPatchKey(key: string): string {
  const k = key.trim();
  if (!k) return "";

  if (k.endsWith("productIdTag.Type") || k.includes("productIdTag.Type")) {
    return "商品タグ";
  }

  const exact: Record<string, string> = {
    fit: "フィット",
    weight: "重さ",
    material: "素材",
    itemType: "アイテム",
    qualityAssurance: "品質保証",
    productIdTag: "商品タグ",
    productName: "商品名",
  };

  if (exact[k]) return exact[k];

  const keyParts = k.split(".");
  const tail = keyParts[keyParts.length - 1] || "";
  const tailNoIndex = tail.replace(/\[\d+\]/g, "");

  if (tailNoIndex === "Type") {
    const parent =
      keyParts.length >= 2
        ? keyParts[keyParts.length - 2].replace(/\[\d+\]/g, "")
        : "";
    if (parent === "productIdTag") return "商品タグ";
  }

  return exact[tailNoIndex] || "";
}

function stringifyPatchValue(value: unknown): string {
  if (value == null) return "-";

  if (typeof value === "string") {
    const s = value.trim();
    return s || "-";
  }

  if (
    typeof value === "number" ||
    typeof value === "boolean" ||
    typeof value === "bigint"
  ) {
    return String(value);
  }

  return String(value);
}

export function flattenProductBlueprintPatch(
  raw: unknown
): ProductBlueprintPatchItem[] {
  const items: Array<{ key: string; value: string }> = [];

  const add = (key: string, value: unknown) => {
    const k = key.trim();
    if (!k) return;
    items.push({ key: k, value: stringifyPatchValue(value) });
  };

  const walk = (value: unknown, prefix = "") => {
    if (value == null) {
      add(prefix, null);
      return;
    }

    if (Array.isArray(value)) {
      value.forEach((child, index) => walk(child, `${prefix}[${index}]`));
      return;
    }

    if (isRecord(value)) {
      Object.keys(value)
        .sort()
        .forEach((key) => {
          const next = prefix ? `${prefix}.${key}` : key;
          walk(value[key], next);
        });
      return;
    }

    add(prefix, value);
  };

  if (Array.isArray(raw) || isRecord(raw)) {
    walk(raw);
  } else {
    add("value", raw);
  }

  return items
    .filter((item) => item.key.trim() !== "")
    .map((item) => ({
      key: item.key,
      label: jpLabelForPatchKey(item.key),
      value: item.value,
    }))
    .filter((item) => item.label && !shouldHidePatchKey(item.key));
}

export function tokenBlueprintPatchHasAnyField(
  vm: TokenBlueprintPatchVM | null
): boolean {
  if (!vm) return false;

  return Boolean(
    vm.id.trim() ||
      vm.tokenName.trim() ||
      vm.symbol.trim() ||
      vm.brandName.trim() ||
      vm.companyName.trim() ||
      vm.description.trim() ||
      vm.tokenIcon.trim()
  );
}

export function transferDisplayName(
  transfer: MallPreviewTransferInfo,
  side: "from" | "to"
): string {
  const prefix = side === "from" ? "from" : "to";

  const avatarName = transfer[`${prefix}AvatarName`].trim();
  const brandName = transfer[`${prefix}BrandName`].trim();
  const avatarId = transfer[`${prefix}AvatarId`].trim();
  const brandId = transfer[`${prefix}BrandId`].trim();
  const walletAddress =
    side === "from"
      ? transfer.fromWalletAddress.trim()
      : transfer.toWalletAddress.trim();

  if (avatarName) return avatarName;
  if (brandName) return brandName;
  if (avatarId) return avatarId;
  if (brandId) return brandId;
  if (walletAddress) return walletAddress;

  return "-";
}

export function transferIconUrl(
  transfer: MallPreviewTransferInfo,
  side: "from" | "to"
): string {
  const prefix = side === "from" ? "from" : "to";
  const avatarIcon = transfer[`${prefix}AvatarIcon`].trim();
  const brandIcon = transfer[`${prefix}BrandIcon`].trim();
  return safeUrl(avatarIcon || brandIcon);
}