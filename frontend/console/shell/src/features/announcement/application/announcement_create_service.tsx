// frontend/console/shell/src/features/announcement/application/announcement_create_service.tsx

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import { fetchTokenBlueprintDetail } from "../../tokenBlueprint/application/tokenBlueprintDetailService";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

import {
  createAnnouncement,
  markAnnouncementPublished,
} from "../infrastructure/announcement_repository_http";

import {
  createAnnouncementClientId,
  uploadAnnouncementImages,
} from "./announcement_attachment_service";

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

    const item =
      owner as AnnouncementCreateLocationOwner;

    return String(
      item.avatarId ?? "",
    ).trim();
  });

  return uniqueStrings(avatarIds).map(
    (avatarId) => ({
      avatarId,
    }),
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
    blueprintValue?.id ??
      tokenBlueprintId ??
      "",
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
    minted: Boolean(
      blueprintValue?.minted,
    ),

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
    uniqueStrings(targetAvatarIds);

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
    params.sales?.tokenBlueprintId.trim() ??
    "";

  if (!tokenBlueprintId) {
    throw new Error(
      "targetToken is required",
    );
  }

  const createdBy =
    params.createdBy.trim();

  if (!createdBy) {
    throw new Error(
      "createdBy is required",
    );
  }

  validateAnnouncementPayload(
    params.payload,
  );

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

async function createDraftAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    tokenBlueprintId,
    createdBy,
    targetAvatarIds,
  } = validateAnnouncementActionParams(
    params,
  );

  const announcementId =
    createAnnouncementClientId();

  const attachments =
    await uploadAnnouncementImages({
      announcementId,
      images: params.payload.images,
    });

  const announcement =
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

  return {
    announcement,
    createdBy,
  };
}

export async function saveAnnouncement(
  params: AnnouncementActionParams,
) {
  const { announcement } =
    await createDraftAnnouncement(params);

  return announcement;
}

export async function sendAnnouncement(
  params: AnnouncementActionParams,
) {
  const {
    announcement,
    createdBy,
  } = await createDraftAnnouncement(
    params,
  );

  return markAnnouncementPublished(
    announcement.id,
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