// frontend/amol/src/features/wallet/types.ts

export type AvatarStateResponse = {
  avatarId?: string;
  followerCount?: number | null;
  followingCount?: number | null;
  postCount?: number | null;
};

export type WalletAvatar = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  profile: string;
  followerCount: number;
  followingCount: number;
};

export type WalletTabKey =
  | "history"
  | "tokens"
  | "resales";