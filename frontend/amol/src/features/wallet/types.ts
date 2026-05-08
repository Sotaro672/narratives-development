// frontend/amol/src/features/wallet/types.ts

export type AvatarResponse = {
  avatarId?: string;
  userId?: string;
  avatarName?: string;
  avatarIcon?: string | null;
  walletAddress?: string;
  profile?: string | null;
};

export type AvatarStateResponse = {
  avatarId?: string;
  followerCount?: number | null;
  followingCount?: number | null;
  postCount?: number | null;
};

export type PublicAvatarAggregateResponse = {
  avatar?: AvatarResponse | null;
  state?: AvatarStateResponse | null;
};

export type WalletAvatar = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  profile: string;
  followerCount: number;
  followingCount: number;
};

export type WalletTabKey = "history" | "tokens";