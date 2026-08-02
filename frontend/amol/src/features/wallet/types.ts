// frontend/amol/src/features/wallet/types.ts
export type WalletAvatar = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  profile: string;
};

export type WalletTabKey =
  | "history"
  | "tokens"
  | "resales";