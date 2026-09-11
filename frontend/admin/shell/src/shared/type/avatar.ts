// frontend/admin/shell/src/shared/type/avatar.ts

export type Avatar = {
  id: string;
  avatarName: string;
  avatarIcon?: string | null;
  profile?: string | null;
  externalLink?: string | null;
  userName: string;
  resaleCount: number;
  reportCount: number;
  createdAt: string;
  updatedAt: string;
};

export type AvatarListResponse = {
  items: Avatar[];
};