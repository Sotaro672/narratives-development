// frontend/amol/src/features/token-commnet/types/tokenTransferTypes.ts

export type TokenTransferTargetTabKey = "following" | "followers";

export type TokenTransferTargetAvatar = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  followedAt: string;
};

export type TokenTransferFollowState = {
  avatarId: string;
  followerCount: number;
  followingCount: number;
  followers: TokenTransferTargetAvatar[];
  following: TokenTransferTargetAvatar[];
  updatedAt: string;
};

export type TokenTransferSheetState = {
  open: boolean;
  activeTab: TokenTransferTargetTabKey;
  loading: boolean;
  refreshing: boolean;
  submitting: boolean;
  errorMessage: string;
  selectedTargetAvatarId: string;
  followState: TokenTransferFollowState | null;
};

export type TokenTransferTargetListProps = {
  targets: TokenTransferTargetAvatar[];
  selectedTargetAvatarId: string;
  emptyTitle: string;
  emptyDescription: string;
  onSelectTarget: (targetAvatarId: string) => void;
};

export type TokenTransferTargetItemProps = {
  target: TokenTransferTargetAvatar;
  selected: boolean;
  onSelect: (targetAvatarId: string) => void;
};

export type TokenTransferSheetProps = {
  open: boolean;
  activeTab: TokenTransferTargetTabKey;
  followState: TokenTransferFollowState | null;
  loading: boolean;
  refreshing: boolean;
  submitting: boolean;
  errorMessage: string;
  selectedTargetAvatarId: string;
  onClose: () => void;
  onChangeTab: (tab: TokenTransferTargetTabKey) => void;
  onRefresh: () => void | Promise<void>;
  onSelectTarget: (targetAvatarId: string) => void;
  onSubmit: () => void | Promise<void>;
};

export type FetchTokenTransferFollowStateParams = {
  backendUrl: string;
  idToken: string;
  avatarId: string;
};

export type TransferTokenToAvatarParams = {
  backendUrl: string;
  idToken: string;
  productId: string;
  targetAvatarId: string;
};

export type TransferTokenToAvatarResponse = {
  avatarId: string;
  targetAvatarId: string;
  productId: string;
  txSignature: string;
  fromWallet: string;
  toWallet: string;
  updatedToAddress: boolean;
  mintAddress: string;
  tokenBlueprintId: string;
};