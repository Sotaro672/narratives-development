// frontend/console/sales/src/application/sales_detail_service.tsx
import type { TokenBlueprint } from "../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../shell/src/shared/util/dateJa";
import { fetchTokenBlueprintDetail } from "../../tokenBlueprint/src/application/tokenBlueprintDetailService";
import { createAnnouncement } from "../infrastructure/announcement_repository_http";

export type SalesOwnerVM = {
  avatarId: string;
  avatarName: string;
  avatarIconUrl: string;
  mintAddress: string;
  productName: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
};

export type SalesEntity = {
  tokenBlueprintId: string;
};

export type SalesDetailVM = {
  sales: SalesEntity | null;
  title: string;
  assigneeId: string;
  assigneeName: string;
  minted: boolean;

  createdById: string;
  createdByName: string;
  createdAt: string;

  updatedById: string;
  updatedByName: string;
  updatedAt: string;

  owners: SalesOwnerVM[];
};

export type SalesDetailInputPayload = {
  title: string;
  text: string;
  images: File[];
};

export type SalesDetailLocationOwner = {
  avatarId?: string;
  avatarName?: string;
  avatarIcon?: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
};

export type SalesDetailLocationProductBlueprint = {
  productBlueprintId?: string;
  productName?: string;
};

export type SalesDetailLocationState = {
  mintAddresses?: string[];
  owners?: SalesDetailLocationOwner[];
  productBlueprints?: SalesDetailLocationProductBlueprint[];
};

type SaveSalesAnnouncementParams = {
  sales: SalesEntity | null;
  payload: SalesDetailInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

type SendSalesAnnouncementParams = {
  sales: SalesEntity | null;
  payload: SalesDetailInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const v of values) {
    const s = String(v ?? "");
    if (!s) continue;
    if (seen.has(s)) continue;

    seen.add(s);
    result.push(s);
  }

  return result;
}

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function getFirstProductName(productBlueprintsValue: unknown): string {
  if (
    !Array.isArray(productBlueprintsValue) ||
    productBlueprintsValue.length === 0
  ) {
    return "";
  }

  for (const item of productBlueprintsValue) {
    if (!item || typeof item !== "object") continue;

    const productName = String(
      (item as SalesDetailLocationProductBlueprint).productName ?? "",
    );

    if (productName) {
      return productName;
    }
  }

  return "";
}

function toOwnersFromState(
  ownersValue: unknown,
  mintAddressesValue: unknown,
  productBlueprintsValue: unknown,
): SalesOwnerVM[] {
  const mintAddresses = uniqueStrings(mintAddressesValue);
  const productName = getFirstProductName(productBlueprintsValue);

  if (!Array.isArray(ownersValue) || ownersValue.length === 0) {
    return mintAddresses.map((mintAddress) => ({
      avatarId: "",
      avatarName: "",
      avatarIconUrl: "",
      mintAddress,
      productName,
      followerCount: 0,
      followingCount: 0,
      postCount: 0,
    }));
  }

  return ownersValue.map((owner, index) => {
    const item =
      owner && typeof owner === "object"
        ? (owner as SalesDetailLocationOwner)
        : {};

    return {
      avatarId: String(item.avatarId ?? ""),
      avatarName: String(item.avatarName ?? ""),
      avatarIconUrl: String(item.avatarIcon ?? ""),
      mintAddress: String(mintAddresses[index] ?? ""),
      productName,
      followerCount: toSafeNumber(item.followerCount),
      followingCount: toSafeNumber(item.followingCount),
      postCount: toSafeNumber(item.postCount),
    };
  });
}

function uniqueAvatarIds(values: unknown): string[] {
  return uniqueStrings(values);
}

export function normalizeSalesDetailLocationState(
  state: unknown,
): SalesDetailLocationState {
  const value =
    state && typeof state === "object"
      ? (state as SalesDetailLocationState)
      : {};

  return {
    mintAddresses: Array.isArray(value.mintAddresses)
      ? value.mintAddresses
      : [],
    owners: Array.isArray(value.owners) ? value.owners : [],
    productBlueprints: Array.isArray(value.productBlueprints)
      ? value.productBlueprints
      : [],
  };
}

export function buildSalesDetailVM(
  blueprint: TokenBlueprint | null,
  tokenBlueprintId: string | undefined,
  locationState: SalesDetailLocationState,
): SalesDetailVM {
  const id = String((blueprint as any)?.id ?? tokenBlueprintId ?? "");

  const sales = id ? { tokenBlueprintId: id } : null;
  const createdById = String((blueprint as any)?.createdBy ?? "");
  const updatedById = String((blueprint as any)?.updatedBy ?? "");

  const createdByName =
    String((blueprint as any)?.createdByName ?? "") || createdById;

  const updatedByName =
    String((blueprint as any)?.updatedByName ?? "") || updatedById;

  return {
    sales,
    title: "営業",
    assigneeId: String((blueprint as any)?.assigneeId ?? ""),
    assigneeName:
      String((blueprint as any)?.assigneeName ?? "") ||
      String((blueprint as any)?.assigneeId ?? ""),
    minted: Boolean((blueprint as any)?.minted),

    createdById,
    createdByName,
    createdAt: safeDateTimeLabelJa((blueprint as any)?.createdAt, ""),

    updatedById,
    updatedByName,
    updatedAt: safeDateTimeLabelJa((blueprint as any)?.updatedAt, ""),

    owners: toOwnersFromState(
      locationState.owners,
      locationState.mintAddresses,
      locationState.productBlueprints,
    ),
  };
}

export async function fetchSalesDetailVM(
  tokenBlueprintId: string | undefined,
  locationState: SalesDetailLocationState,
): Promise<SalesDetailVM> {
  const id = String(tokenBlueprintId ?? "");

  if (!id) {
    return buildSalesDetailVM(null, tokenBlueprintId, locationState);
  }

  const blueprint = await fetchTokenBlueprintDetail(id);

  return buildSalesDetailVM(blueprint, tokenBlueprintId, locationState);
}

function validateAnnouncementPayload(payload: SalesDetailInputPayload) {
  if (!payload.title) {
    throw new Error("タイトルを入力してください。");
  }

  if (!payload.text) {
    throw new Error("文章を入力してください。");
  }
}

function validateTargetAvatarIds(targetAvatarIds: string[]) {
  if (uniqueAvatarIds(targetAvatarIds).length === 0) {
    throw new Error("告知先のアバターを選択してください。");
  }
}

export async function saveSalesAnnouncement({
  sales,
  payload,
  createdBy,
  targetAvatarIds,
}: SaveSalesAnnouncementParams) {
  if (!sales?.tokenBlueprintId) {
    throw new Error("targetToken is required");
  }

  if (!createdBy) {
    throw new Error("createdBy is required");
  }

  validateAnnouncementPayload(payload);
  validateTargetAvatarIds(targetAvatarIds);

  return createAnnouncement({
    title: payload.title,
    content: payload.text,
    targetToken: sales.tokenBlueprintId,
    targetAvatars: uniqueAvatarIds(targetAvatarIds),
    attachments: [],
    published: false,
    publishedAt: null,
    createdBy,
  });
}

export async function sendSalesAnnouncement({
  sales,
  payload,
  createdBy,
  targetAvatarIds,
}: SendSalesAnnouncementParams) {
  if (!sales?.tokenBlueprintId) {
    throw new Error("targetToken is required");
  }

  if (!createdBy) {
    throw new Error("createdBy is required");
  }

  validateAnnouncementPayload(payload);
  validateTargetAvatarIds(targetAvatarIds);

  return createAnnouncement({
    title: payload.title,
    content: payload.text,
    targetToken: sales.tokenBlueprintId,
    targetAvatars: uniqueAvatarIds(targetAvatarIds),
    attachments: [],
    published: true,
    publishedAt: new Date().toISOString(),
    createdBy,
  });
}

export function createEmptySalesDetailVM(): SalesDetailVM {
  return {
    sales: null,
    title: "営業",
    assigneeId: "",
    assigneeName: "",
    minted: false,

    createdById: "",
    createdByName: "",
    createdAt: "",

    updatedById: "",
    updatedByName: "",
    updatedAt: "",

    owners: [],
  };
}