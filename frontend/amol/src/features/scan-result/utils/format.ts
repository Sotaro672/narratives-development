// frontend/amol/src/features/scan-result/utils/format.ts

import { textOrEmpty } from "../../../components/utils/textOrEmpty";
import { isRecord } from "../../../components/utils/typeGuards";

import type {
  MallOwnerInfo,
  MallPreviewTransferInfo,
  ProductBlueprintPatchItem,
  TokenBlueprintPatchVM,
} from "../../shared/types/scanResult";

export function ownerLabel(
  owner: MallOwnerInfo | null | undefined,
): string {
  if (!owner) {
    return "-";
  }

  const avatarName = owner.avatarName.trim();
  const brandName = owner.brandName.trim();
  const avatarId = owner.avatarId.trim();
  const brandId = owner.brandId.trim();

  if (avatarName) {
    return avatarName;
  }

  if (brandName) {
    return brandName;
  }

  if (avatarId) {
    return avatarId;
  }

  if (brandId) {
    return brandId;
  }

  return "-";
}

export function shortAddress(
  value: string,
): string {
  const address = value.trim();

  if (address.length <= 16) {
    return address;
  }

  return `${address.slice(0, 8)}...${address.slice(-8)}`;
}

export function withCm(
  value: unknown,
): string {
  const text = textOrEmpty(value);

  if (!text) {
    return "-";
  }

  if (/\s*cm$/i.test(text)) {
    return text;
  }

  return `${text}cm`;
}

export function safeUrl(
  raw: string,
): string {
  const value = raw.trim();

  if (!value) {
    return "";
  }

  try {
    const url = new URL(value);

    if (url.protocol && url.host) {
      return url.toString();
    }
  } catch {
    // URLとして解釈できない場合はencodeURIした値を返す
  }

  return encodeURI(value);
}

export function shouldHidePatchKey(
  rawKey: string,
): boolean {
  const key = rawKey.trim();

  if (!key) {
    return true;
  }

  const hidden = new Set([
    "assigneeId",
    "brandId",
  ]);

  if (hidden.has(key)) {
    return true;
  }

  const keyParts = key.split(".");
  const tail =
    keyParts[keyParts.length - 1] || "";

  const tailNoIndex = tail.replace(
    /\[\d+\]/g,
    "",
  );

  return hidden.has(tailNoIndex);
}

export function jpLabelForPatchKey(
  key: string,
): string {
  const normalizedKey = key.trim();

  if (!normalizedKey) {
    return "";
  }

  if (
    normalizedKey.endsWith(
      "productIdTag.Type",
    ) ||
    normalizedKey.includes(
      "productIdTag.Type",
    )
  ) {
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

  if (exact[normalizedKey]) {
    return exact[normalizedKey];
  }

  const keyParts = normalizedKey.split(".");
  const tail =
    keyParts[keyParts.length - 1] || "";

  const tailNoIndex = tail.replace(
    /\[\d+\]/g,
    "",
  );

  if (tailNoIndex === "Type") {
    const parent =
      keyParts.length >= 2
        ? keyParts[
            keyParts.length - 2
          ].replace(/\[\d+\]/g, "")
        : "";

    if (parent === "productIdTag") {
      return "商品タグ";
    }
  }

  return exact[tailNoIndex] || "";
}

function stringifyPatchValue(
  value: unknown,
): string {
  if (value == null) {
    return "-";
  }

  if (typeof value === "string") {
    const text = value.trim();

    return text || "-";
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
  raw: unknown,
): ProductBlueprintPatchItem[] {
  const items: Array<{
    key: string;
    value: string;
  }> = [];

  const add = (
    key: string,
    value: unknown,
  ) => {
    const normalizedKey = key.trim();

    if (!normalizedKey) {
      return;
    }

    items.push({
      key: normalizedKey,
      value: stringifyPatchValue(value),
    });
  };

  const walk = (
    value: unknown,
    prefix = "",
  ) => {
    if (value == null) {
      add(prefix, null);

      return;
    }

    if (Array.isArray(value)) {
      value.forEach((child, index) => {
        walk(
          child,
          `${prefix}[${index}]`,
        );
      });

      return;
    }

    if (isRecord(value)) {
      Object.keys(value)
        .sort()
        .forEach((key) => {
          const next = prefix
            ? `${prefix}.${key}`
            : key;

          walk(value[key], next);
        });

      return;
    }

    add(prefix, value);
  };

  if (
    Array.isArray(raw) ||
    isRecord(raw)
  ) {
    walk(raw);
  } else {
    add("value", raw);
  }

  return items
    .filter(
      (item) =>
        item.key.trim() !== "",
    )
    .map((item) => ({
      key: item.key,
      label: jpLabelForPatchKey(
        item.key,
      ),
      value: item.value,
    }))
    .filter(
      (item) =>
        Boolean(item.label) &&
        !shouldHidePatchKey(item.key),
    );
}

export function tokenBlueprintPatchHasAnyField(
  vm: TokenBlueprintPatchVM | null,
): boolean {
  if (!vm) {
    return false;
  }

  return Boolean(
    vm.id.trim() ||
      vm.tokenName.trim() ||
      vm.symbol.trim() ||
      vm.brandName.trim() ||
      vm.companyName.trim() ||
      vm.description.trim() ||
      vm.tokenIcon.trim(),
  );
}

export function transferDisplayName(
  transfer: MallPreviewTransferInfo,
  side: "from" | "to",
): string {
  const prefix =
    side === "from"
      ? "from"
      : "to";

  const avatarName =
    transfer[
      `${prefix}AvatarName`
    ].trim();

  const brandName =
    transfer[
      `${prefix}BrandName`
    ].trim();

  const avatarId =
    transfer[
      `${prefix}AvatarId`
    ].trim();

  const brandId =
    transfer[
      `${prefix}BrandId`
    ].trim();

  const walletAddress =
    side === "from"
      ? transfer.fromWalletAddress.trim()
      : transfer.toWalletAddress.trim();

  if (avatarName) {
    return avatarName;
  }

  if (brandName) {
    return brandName;
  }

  if (avatarId) {
    return avatarId;
  }

  if (brandId) {
    return brandId;
  }

  if (walletAddress) {
    return walletAddress;
  }

  return "-";
}

export function transferIconUrl(
  transfer: MallPreviewTransferInfo,
  side: "from" | "to",
): string {
  const prefix =
    side === "from"
      ? "from"
      : "to";

  const avatarIcon =
    transfer[
      `${prefix}AvatarIcon`
    ].trim();

  const brandIcon =
    transfer[
      `${prefix}BrandIcon`
    ].trim();

  return safeUrl(
    avatarIcon || brandIcon,
  );
}

export function transferBrandId(
  transfer: MallPreviewTransferInfo,
  side: "from" | "to",
): string {
  const prefix =
    side === "from"
      ? "from"
      : "to";

  return transfer[
    `${prefix}BrandId`
  ].trim();
}