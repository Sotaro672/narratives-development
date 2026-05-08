// frontend/amol/src/features/wallet/types/followTypes.ts

export type PublicWalletFollowTabKey = "following" | "followers";

export type PublicWalletFollowUser = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  followedAt: string;
};

export type PublicWalletFollowState = {
  avatarId: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
  followers: PublicWalletFollowUser[];
  following: PublicWalletFollowUser[];
  lastActiveAt: string;
  updatedAt: string;
};

export type FetchPublicWalletFollowStateInput = {
  backendUrl: string;
  idToken: string;
  avatarId: string;
};