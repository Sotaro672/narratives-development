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

export type AvatarResale = {
  id: string;
  productName: string;
  tokenName: string;
  status: string;
  price: number;
  reportCount: number;
  createdAt: string;
  updatedAt?: string | null;
};

export type AvatarResaleListResponse = {
  items: AvatarResale[];
};