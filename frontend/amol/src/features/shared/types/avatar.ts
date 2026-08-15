// frontend/amol/src/features/shared/types/avatar.ts

export type AvatarFormMode = "create" | "edit";

export type PickIconResult = {
  file: File | null;
  fileName: string | null;
  mimeType: string | null;
  previewUrl: string | null;
  error?: string;
};

export type AvatarCreateResult =
  | { ok: true; message: string; nextRoute: string; createdAvatarId: string }
  | { ok: false; message: string };

export type AvatarUpdateResult =
  | { ok: true; message: string; avatarId: string }
  | { ok: false; message: string };

export type AvatarMutationResponse = {
  avatarId: string;
  userId: string;
  avatarName: string;
  avatarIcon?: string | null;
  walletAddress?: string | null;
  profile?: string | null;
  externalLink?: string | null;
};

export type MyAvatarResponse = {
  avatarId: string;
  userId: string;
  avatarName: string;
  avatarIcon?: string | null;
  walletAddress: string;
  profile?: string | null;
  externalLink?: string | null;
};

export type AvatarPayloadBase = {
  avatarName: string;
  avatarIcon?: string;
  profile?: string;
  externalLink?: string;
};

export type CreateAvatarPayload = AvatarPayloadBase & {
  userUid: string;
};