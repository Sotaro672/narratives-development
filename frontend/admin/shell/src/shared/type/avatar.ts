// frontend/admin/shell/src/shared/type/avatar.ts

export type Avatar = {
  id: string;
  userId: string;
  avatarName: string;
  avatarIcon?: string | null;
  walletAddress?: string | null;
  profile?: string | null;
  externalLink?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type AvatarListResponse = {
  items: Avatar[];
};