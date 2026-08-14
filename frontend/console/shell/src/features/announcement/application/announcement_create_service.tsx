// frontend/console/shell/src/features/announcement/application/announcement_create_service.tsx

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";
import { fetchTokenBlueprintDetail } from "../../tokenBlueprint/application/tokenBlueprintDetailService";

import {
  createAnnouncement,
  markAnnouncementPublished,
} from "../infrastructure/announcement_repository_http";

import {
  createAnnouncementClientId,
  uploadAnnouncementImages,
} from "./announcement_attachment_service";

import type { AnnouncementInputPayload } from "./announcement_input";

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

export type AnnouncementCreateLocationOwner = {
  avatarId?: string;
};

export type AnnouncementCreateLocationState = {
  owners?: AnnouncementCreateLocationOwner[];
};

type AnnouncementActionParams = {
  sales: AnnouncementEntity | null;
  payload: AnnouncementInputPayload;
  createdBy: string;
  targetAvatarIds: string[];
};

// ============================================================
// Location state
// ============================================================

export function normalizeAnnouncementCreateLocationState(
  state: unknown,
): AnnouncementCreateLocationState {
  if (!state || typeof state !== "object") {
    return { owners: [] };
  }

  const value = state as AnnouncementCreateLocationState;

  return {
    owners: Array.isArray(value.owners) ? value.owners : [],
  };
}

function toOwnersFromState(
  owners: AnnouncementCreateLocationOwner[] | undefined,
): AnnouncementOwnerVM[] {
  if (!owners) {
    return [];
  }

  const avatarIds = owners
    .map((owner) => owner.avatarId)
    .filter((avatarId): avatarId is string => Boolean(avatarId));

  return [...new Set(avatarIds)].map((avatarId) => ({ avatarId }));
}

// ============================================================
// View model builder
// ============================================================

export function buildAnnouncementCreateVM(
  blueprint: TokenBlueprint | null,
  locationState: AnnouncementCreateLocationState,
): AnnouncementCreateVM {
  if (!blueprint) {
    return createEmptyAnnouncementCreateVM();
  }

  return {
    sales: {
      tokenBlueprintId: blueprint.id,
    },
    title: "告知",
    minted: blueprint.minted,
    createdById: blueprint.createdBy ?? "",
    createdByName: blueprint.createdByName ?? "",
    createdAt: safeDateTimeLabelJa(blueprint.createdAt, ""),
    updatedById: blueprint.updatedBy ?? "",
    updatedByName: blueprint.updatedByName ?? "",
    updatedAt: safeDateTimeLabelJa(blueprint.updatedAt, ""),
    owners: toOwnersFromState(locationState.owners),
  };
}

export async function fetchAnnouncementCreateVM(
  tokenBlueprintId: string | undefined,
  locationState: AnnouncementCreateLocationState,
): Promise<AnnouncementCreateVM> {
  if (!tokenBlueprintId) {
    return createEmptyAnnouncementCreateVM();
  }

  const blueprint = await fetchTokenBlueprintDetail(tokenBlueprintId);

  return buildAnnouncementCreateVM(blueprint, locationState);
}

// ============================================================
// Validation
// ============================================================

function validateAnnouncementPayload(
  payload: AnnouncementInputPayload,
): void {
  if (!payload.title.trim()) {
    throw new Error("タイトルを入力してください。");
  }

  if (!payload.text.trim()) {
    throw new Error("本文を入力してください。");
  }
}

function validateTargetAvatarIds(
  targetAvatarIds: string[],
): string[] {
  const ids = [...new Set(targetAvatarIds.filter(Boolean))];

  if (ids.length === 0) {
    throw new Error("告知先のアバターを選択してください。");
  }

  return ids;
}

function validateAnnouncementActionParams(
  params: AnnouncementActionParams,
): {
  tokenBlueprintId: string;
  createdBy: string;
  targetAvatarIds: string[];
} {
  if (!params.sales?.tokenBlueprintId) {
    throw new Error("targetToken is required");
  }

  if (!params.createdBy) {
    throw new Error("createdBy is required");
  }

  validateAnnouncementPayload(params.payload);

  return {
    tokenBlueprintId: params.sales.tokenBlueprintId,
    createdBy: params.createdBy,
    targetAvatarIds: validateTargetAvatarIds(params.targetAvatarIds),
  };
}

// ============================================================
// Service
// ============================================================

async function createDraftAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    tokenBlueprintId,
    createdBy,
    targetAvatarIds,
  } = validateAnnouncementActionParams(params);

  const announcementId = createAnnouncementClientId();
  const images: File[] = [];

  for (const attachment of params.payload.attachments) {
    if (attachment.type === "new") {
      images.push(attachment.file);
    }
  }

  const attachments = await uploadAnnouncementImages({
    announcementId,
    images,
  });

  const announcement = await createAnnouncement({
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

  return {
    announcement,
    createdBy,
  };
}

export async function saveAnnouncement(
  params: AnnouncementActionParams,
) {
  const { announcement } = await createDraftAnnouncement(params);
  return announcement;
}

export async function sendAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    announcement,
    createdBy,
  } = await createDraftAnnouncement(params);

  return markAnnouncementPublished(announcement.id, {
    updatedBy: createdBy,
  });
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