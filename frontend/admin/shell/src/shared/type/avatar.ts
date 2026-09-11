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

export type AvatarResaleImage = {
  id: string;
  url: string;
  displayOrder: number;
};

export type AvatarResale = {
  id: string;
  companyId: string;
  productBlueprintId: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  reportCaseId: string;
  status: string;
  price: number;
  condition: string;
  images: AvatarResaleImage[];
  reportCount: number;
  createdAt: string;
  updatedAt?: string | null;
};

export type AvatarResaleListResponse = {
  items: AvatarResale[];
};