//frontend\console\sales\application\announcement_create_service.tsx
import type { TokenBlueprint } from "../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../shell/src/shared/util/dateJa";
import { fetchTokenBlueprintDetail } from "../../tokenBlueprint/src/application/tokenBlueprintDetailService";
import {
  createAnnouncement,
  markAnnouncementPublished,
  type AnnouncementAttachmentInput,
} from "../infrastructure/announcement_repository_http";

import {
  getDownloadURL,
  getStorage,
  ref as storageRef,
  uploadBytes,
} from "firebase/storage";

export type AnnouncementOwnerVM = {
  avatarId: string;
  avatarName: string;
  avatarIconUrl: string;
  mintAddress: string;
  productName: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
};

export type AnnouncementEntity = {
  tokenBlueprintId: string;
};

export type AnnouncementCreateVM = {
  sales: AnnouncementEntity | null;
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

  owners: AnnouncementOwnerVM[];
};

export type AnnouncementCreateInputPayload = {
  title: string;
  text: string;
  images: File[];
  imageUrls?: string[];
};

export type AnnouncementCreateLocationOwner = {
  avatarId?: string;
  avatarName?: string;
  avatarIcon?: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
};

export type AnnouncementCreateLocationProductBlueprint = {
  productBlueprintId?: string;
  productName?: string;
};

export type AnnouncementCreateLocationState = {
  mintAddresses?: string[];
  owners?: AnnouncementCreateLocationOwner[];
  productBlueprints?: AnnouncementCreateLocationProductBlueprint[];
};

type SaveAnnouncementParams = {
  sales: AnnouncementEntity | null;
  payload: AnnouncementCreateInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

type SendAnnouncementParams = {
  sales: AnnouncementEntity | null;
  payload: AnnouncementCreateInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const v of values) {
    const s = String(v ?? "").trim();
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
      (item as AnnouncementCreateLocationProductBlueprint).productName ?? "",
    ).trim();

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
): AnnouncementOwnerVM[] {
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
        ? (owner as AnnouncementCreateLocationOwner)
        : {};

    return {
      avatarId: String(item.avatarId ?? "").trim(),
      avatarName: String(item.avatarName ?? "").trim(),
      avatarIconUrl: String(item.avatarIcon ?? "").trim(),
      mintAddress: String(mintAddresses[index] ?? "").trim(),
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

function createClientId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2, 12)}`;
}

function sanitizePathSegment(value: string): string {
  const normalized = String(value ?? "").trim();

  if (!normalized) {
    return "file";
  }

  return normalized
    .replace(/[\\/#?[\]*]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/_+/g, "_");
}

function getFileExtension(fileName: string): string {
  const normalized = String(fileName ?? "").trim();
  const index = normalized.lastIndexOf(".");

  if (index < 0 || index === normalized.length - 1) {
    return "";
  }

  return normalized.slice(index);
}

function buildAnnouncementAttachmentStorageFileName(params: {
  file: File;
  index: number;
}): string {
  const extension = getFileExtension(params.file.name || "image");
  const attachmentId = createClientId();
  const displayOrder = String(params.index + 1).padStart(2, "0");

  return sanitizePathSegment(`${displayOrder}-${attachmentId}${extension}`);
}

function buildAnnouncementAttachmentObjectPath(params: {
  announcementId: string;
  storageFileName: string;
}): string {
  const announcementId = sanitizePathSegment(params.announcementId);
  const storageFileName = sanitizePathSegment(params.storageFileName);

  return [
    "announcements",
    announcementId,
    "attachments",
    storageFileName,
  ].join("/");
}

async function uploadAnnouncementImage(params: {
  announcementId: string;
  file: File;
  index: number;
}): Promise<AnnouncementAttachmentInput> {
  const storageFileName = buildAnnouncementAttachmentStorageFileName({
    file: params.file,
    index: params.index,
  });

  const mimeType = String(params.file.type || "application/octet-stream").trim();

  const objectPath = buildAnnouncementAttachmentObjectPath({
    announcementId: params.announcementId,
    storageFileName,
  });

  const storage = getStorage();
  const ref = storageRef(storage, objectPath);

  await uploadBytes(ref, params.file, {
    contentType: mimeType,
    customMetadata: {
      announcementId: params.announcementId,
      fileName: storageFileName,
      originalFileName: params.file.name,
    },
  });

  const fileUrl = await getDownloadURL(ref);

  return {
    fileName: storageFileName,
    fileUrl,
    fileSize: params.file.size,
    mimeType,
    objectPath,
  };
}

async function uploadAnnouncementImages(params: {
  announcementId: string;
  images: File[];
}): Promise<AnnouncementAttachmentInput[]> {
  const images = Array.isArray(params.images) ? params.images : [];

  const validImages = images.filter((file) => {
    return file instanceof File && file.type.startsWith("image/");
  });

  if (validImages.length === 0) {
    return [];
  }

  return Promise.all(
    validImages.map((file, index) =>
      uploadAnnouncementImage({
        announcementId: params.announcementId,
        file,
        index,
      }),
    ),
  );
}

export function normalizeAnnouncementCreateLocationState(
  state: unknown,
): AnnouncementCreateLocationState {
  const value =
    state && typeof state === "object"
      ? (state as AnnouncementCreateLocationState)
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

export function buildAnnouncementCreateVM(
  blueprint: TokenBlueprint | null,
  tokenBlueprintId: string | undefined,
  locationState: AnnouncementCreateLocationState,
): AnnouncementCreateVM {
  const id = String((blueprint as any)?.id ?? tokenBlueprintId ?? "").trim();

  const sales = id ? { tokenBlueprintId: id } : null;
  const createdById = String((blueprint as any)?.createdBy ?? "").trim();
  const updatedById = String((blueprint as any)?.updatedBy ?? "").trim();

  const createdByName =
    String((blueprint as any)?.createdByName ?? "").trim() || createdById;

  const updatedByName =
    String((blueprint as any)?.updatedByName ?? "").trim() || updatedById;

  return {
    sales,
    title: "告知",
    assigneeId: String((blueprint as any)?.assigneeId ?? "").trim(),
    assigneeName:
      String((blueprint as any)?.assigneeName ?? "").trim() ||
      String((blueprint as any)?.assigneeId ?? "").trim(),
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

export async function fetchAnnouncementCreateVM(
  tokenBlueprintId: string | undefined,
  locationState: AnnouncementCreateLocationState,
): Promise<AnnouncementCreateVM> {
  const id = String(tokenBlueprintId ?? "").trim();

  if (!id) {
    return buildAnnouncementCreateVM(null, tokenBlueprintId, locationState);
  }

  const blueprint = await fetchTokenBlueprintDetail(id);

  return buildAnnouncementCreateVM(blueprint, tokenBlueprintId, locationState);
}

function validateAnnouncementPayload(payload: AnnouncementCreateInputPayload) {
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

export async function saveAnnouncement({
  sales,
  payload,
  createdBy,
  targetAvatarIds,
}: SaveAnnouncementParams) {
  if (!sales?.tokenBlueprintId) {
    throw new Error("targetToken is required");
  }

  if (!createdBy) {
    throw new Error("createdBy is required");
  }

  validateAnnouncementPayload(payload);
  validateTargetAvatarIds(targetAvatarIds);

  const announcementId = createClientId();

  const attachments = await uploadAnnouncementImages({
    announcementId,
    images: payload.images,
  });

  return createAnnouncement({
    id: announcementId,
    title: payload.title,
    content: payload.text,
    targetToken: sales.tokenBlueprintId,
    targetAvatars: uniqueAvatarIds(targetAvatarIds),
    attachments,
    published: false,
    publishedAt: null,
    createdBy,
  });
}

export async function sendAnnouncement({
  sales,
  payload,
  createdBy,
  targetAvatarIds,
}: SendAnnouncementParams) {
  if (!sales?.tokenBlueprintId) {
    throw new Error("targetToken is required");
  }

  if (!createdBy) {
    throw new Error("createdBy is required");
  }

  validateAnnouncementPayload(payload);
  validateTargetAvatarIds(targetAvatarIds);

  const announcementId = createClientId();

  const attachments = await uploadAnnouncementImages({
    announcementId,
    images: payload.images,
  });

  await createAnnouncement({
    id: announcementId,
    title: payload.title,
    content: payload.text,
    targetToken: sales.tokenBlueprintId,
    targetAvatars: uniqueAvatarIds(targetAvatarIds),
    attachments,
    published: false,
    publishedAt: null,
    createdBy,
  });

  return markAnnouncementPublished(announcementId, {
    updatedBy: createdBy,
  });
}

export function createEmptyAnnouncementCreateVM(): AnnouncementCreateVM {
  return {
    sales: null,
    title: "告知",
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