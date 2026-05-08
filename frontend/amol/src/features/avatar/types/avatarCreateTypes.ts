// frontend/src/features/avatar/types/avatarCreateTypes.ts

export type AvatarFormMode = "create" | "edit";

export type PickIconResult = {
  file: File | null;
  fileName: string | null;
  mimeType: string | null;
  previewUrl: string | null;
  error?: string;
};

export type AvatarCreateResult = {
  ok: boolean;
  message: string;
  nextRoute?: string;
  createdAvatarId?: string;
};

export type AvatarUpdateResult = {
  ok: boolean;
  message: string;
  avatarId?: string;
};

export type CreateAvatarResponse = {
  avatarId: string;
  userId?: string;
  avatarName?: string;
  avatarIcon?: string | null;
  profile?: string | null;
  externalLink?: string | null;
};

export type UpdateAvatarResponse = {
  avatarId: string;
  userId?: string;
  avatarName?: string;
  avatarIcon?: string | null;
  profile?: string | null;
  externalLink?: string | null;
};

export type MyAvatarResponse = {
  avatarId: string;
  userId?: string;
  avatarName: string;
  profile?: string | null;
  externalLink?: string | null;
  avatarIcon?: string | null;
};

export type CreateAvatarPayload = {
  userId: string;
  userUid?: string;
  avatarName: string;
  avatarIcon?: string;
  profile?: string;
  externalLink?: string;
};

export type UpdateAvatarPayload = {
  avatarName: string;
  profile?: string;
  externalLink?: string;
  avatarIcon?: string;
};