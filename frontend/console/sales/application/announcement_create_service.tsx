// frontend/console/sales/application/announcement_create_service.tsx
import type { TokenBlueprint } from "../../tokenBlueprint/src/domain/tokenBlueprint";
import { fetchTokenBlueprintDetail } from "../../tokenBlueprint/src/application/tokenBlueprintDetailService";
import { safeDateTimeLabelJa } from "../../shell/src/shared/util/dateJa";
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

// ============================================================
// View model
// ============================================================

export type AnnouncementOwnerVM = {
  avatarId: string;
};

export type AnnouncementEntity = {
  tokenBlueprintId: string;
};

export type AnnouncementCreateVM = {
  sales: AnnouncementEntity | null;
  title: string;
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
};

export type AnnouncementCreateLocationState = {
  owners?: AnnouncementCreateLocationOwner[];
};

type AnnouncementActionParams = {
  sales: AnnouncementEntity | null;
  payload: AnnouncementCreateInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

// ============================================================
// Normalization helpers
// ============================================================

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = String(value ?? "").trim();

    if (!normalized || seen.has(normalized)) {
      continue;
    }

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

function toOwnersFromState(
  ownersValue: unknown,
): AnnouncementOwnerVM[] {
  if (!Array.isArray(ownersValue)) {
    return [];
  }

  const avatarIds = ownersValue.map((owner) => {
    if (!owner || typeof owner !== "object") {
      return "";
    }

    const item = owner as AnnouncementCreateLocationOwner;

    return String(item.avatarId ?? "").trim();
  });

  return uniqueStrings(avatarIds).map((avatarId) => ({
    avatarId,
  }));
}

function uniqueAvatarIds(values: unknown): string[] {
  return uniqueStrings(values);
}

// ============================================================
// Client ID
// ============================================================

function createClientId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random()
    .toString(36)
    .slice(2, 12)}`;
}

// ============================================================
// Attachment helpers
// ============================================================

function sanitizePathSegment(value: string): string {
  const normalized = value.trim();

  if (!normalized) {
    return "file";
  }

  return normalized
    .replace(/[\\/#?[\]*]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/_+/g, "_");
}

function getFileExtension(fileName: string): string {
  const normalized = fileName.trim();
  const index = normalized.lastIndexOf(".");

  if (
    index < 0 ||
    index === normalized.length - 1
  ) {
    return "";
  }

  return normalized.slice(index);
}

function buildAnnouncementAttachmentStorageFileName(params: {
  file: File;
  index: number;
}): string {
  const extension = getFileExtension(
    params.file.name || "image",
  );

  const attachmentId = createClientId();
  const displayOrder = String(params.index + 1).padStart(
    2,
    "0",
  );

  return sanitizePathSegment(
    `${displayOrder}-${attachmentId}${extension}`,
  );
}

function buildAnnouncementAttachmentObjectPath(params: {
  announcementId: string;
  storageFileName: string;
}): string {
  const announcementId = sanitizePathSegment(
    params.announcementId,
  );

  const storageFileName = sanitizePathSegment(
    params.storageFileName,
  );

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
  const storageFileName =
    buildAnnouncementAttachmentStorageFileName({
      file: params.file,
      index: params.index,
    });

  const mimeType =
    params.file.type || "application/octet-stream";

  const objectPath =
    buildAnnouncementAttachmentObjectPath({
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
  const validImages = params.images.filter(
    (file) =>
      file instanceof File &&
      file.type.startsWith("image/"),
  );

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

// ============================================================
// Location state
// ============================================================

export function normalizeAnnouncementCreateLocationState(
  state: unknown,
): AnnouncementCreateLocationState {
  if (!state || typeof state !== "object") {
    return {
      owners: [],
    };
  }

  const value =
    state as AnnouncementCreateLocationState;

  return {
    owners: Array.isArray(value.owners)
      ? value.owners
      : [],
  };
}

// ============================================================
// View model builder
// ============================================================

export function buildAnnouncementCreateVM(
  blueprint: TokenBlueprint | null,
  tokenBlueprintId: string | undefined,
  locationState: AnnouncementCreateLocationState,
): AnnouncementCreateVM {
  const blueprintValue = blueprint as
    | (TokenBlueprint & {
        id?: string;
        createdBy?: string;
        createdByName?: string;
        createdAt?: string | null;
        updatedBy?: string;
        updatedByName?: string;
        updatedAt?: string | null;
        minted?: boolean;
      })
    | null;

  const id = String(
    blueprintValue?.id ?? tokenBlueprintId ?? "",
  ).trim();

  const createdById = String(
    blueprintValue?.createdBy ?? "",
  ).trim();

  const updatedById = String(
    blueprintValue?.updatedBy ?? "",
  ).trim();

  const createdByName =
    String(
      blueprintValue?.createdByName ?? "",
    ).trim() || createdById;

  const updatedByName =
    String(
      blueprintValue?.updatedByName ?? "",
    ).trim() || updatedById;

  return {
    sales: id
      ? {
          tokenBlueprintId: id,
        }
      : null,

    title: "告知",
    minted: Boolean(blueprintValue?.minted),

    createdById,
    createdByName,
    createdAt: safeDateTimeLabelJa(
      blueprintValue?.createdAt,
      "",
    ),

    updatedById,
    updatedByName,
    updatedAt: safeDateTimeLabelJa(
      blueprintValue?.updatedAt,
      "",
    ),

    owners: toOwnersFromState(
      locationState.owners,
    ),
  };
}

export async function fetchAnnouncementCreateVM(
  tokenBlueprintId: string | undefined,
  locationState: AnnouncementCreateLocationState,
): Promise<AnnouncementCreateVM> {
  const id = String(
    tokenBlueprintId ?? "",
  ).trim();

  if (!id) {
    return buildAnnouncementCreateVM(
      null,
      tokenBlueprintId,
      locationState,
    );
  }

  const blueprint =
    await fetchTokenBlueprintDetail(id);

  return buildAnnouncementCreateVM(
    blueprint,
    tokenBlueprintId,
    locationState,
  );
}

// ============================================================
// Validation
// ============================================================

function validateAnnouncementPayload(
  payload: AnnouncementCreateInputPayload,
): void {
  if (!payload.title.trim()) {
    throw new Error(
      "タイトルを入力してください。",
    );
  }

  if (!payload.text.trim()) {
    throw new Error(
      "本文を入力してください。",
    );
  }
}

function validateTargetAvatarIds(
  targetAvatarIds: string[],
): string[] {
  const normalizedIds =
    uniqueAvatarIds(targetAvatarIds);

  if (normalizedIds.length === 0) {
    throw new Error(
      "告知先のアバターを選択してください。",
    );
  }

  return normalizedIds;
}

function validateAnnouncementActionParams(
  params: AnnouncementActionParams,
): {
  tokenBlueprintId: string;
  createdBy: string;
  targetAvatarIds: string[];
} {
  const tokenBlueprintId =
    params.sales?.tokenBlueprintId.trim() ?? "";

  if (!tokenBlueprintId) {
    throw new Error(
      "targetToken is required",
    );
  }

  const createdBy = params.createdBy.trim();

  if (!createdBy) {
    throw new Error(
      "createdBy is required",
    );
  }

  validateAnnouncementPayload(params.payload);

  return {
    tokenBlueprintId,
    createdBy,
    targetAvatarIds:
      validateTargetAvatarIds(
        params.targetAvatarIds,
      ),
  };
}

// ============================================================
// Service
// ============================================================

export async function saveAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    tokenBlueprintId,
    createdBy,
    targetAvatarIds,
  } = validateAnnouncementActionParams(params);

  const announcementId = createClientId();

  const attachments =
    await uploadAnnouncementImages({
      announcementId,
      images: params.payload.images,
    });

  return createAnnouncement({
    id: announcementId,
    title: params.payload.title.trim(),
    content: params.payload.text.trim(),
    targetToken: tokenBlueprintId,
    targetAvatars: targetAvatarIds,
    attachments,
    published: false,
    publishedAt: null,
    createdBy,
  });
}

export async function sendAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    tokenBlueprintId,
    createdBy,
    targetAvatarIds,
  } = validateAnnouncementActionParams(params);

  const announcementId = createClientId();

  const attachments =
    await uploadAnnouncementImages({
      announcementId,
      images: params.payload.images,
    });

  await createAnnouncement({
    id: announcementId,
    title: params.payload.title.trim(),
    content: params.payload.text.trim(),
    targetToken: tokenBlueprintId,
    targetAvatars: targetAvatarIds,
    attachments,
    published: false,
    publishedAt: null,
    createdBy,
  });

  return markAnnouncementPublished(
    announcementId,
    {
      updatedBy: createdBy,
    },
  );
}

export function createEmptyAnnouncementCreateVM(): AnnouncementCreateVM {
  return {
    sales: null,
    title: "告知",
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