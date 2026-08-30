// frontend/amol/src/features/shared/types/resaleReview.ts

export type ResaleInteractionSummary = {
  resaleId: string;
  likeCount: number;
  commentCount: number;
  likedByMe: boolean;
};

export type ResaleReviewComment = {
  commentId: string;
  resaleId: string;
  avatarId: string;
  body: string;
  deleted: boolean;
  isRead: boolean;
  createdAt: string;
  updatedAt: string;
  avatarName: string;
  avatarIcon: string;
};

export type ResaleReviewCommentPage = {
  items: ResaleReviewComment[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type ResaleChatLatestComment = {
  commentId: string;
  resaleId: string;
  avatarId: string;
  body: string;
  deleted: boolean;
  isRead: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ResaleChatListItem = {
  resaleId: string;
  status: "listing" | "suspended" | "sold";
  productName: string;
  tokenName: string;
  tokenIcon: string;
  brandName: string;
  imageUrl: string;
  price: number;
  latestComment?: ResaleChatLatestComment;
  commentCount: number;
  unreadCommentCount: number;
  latestActivityAt: string;
};

export type ResaleChatListResponse = {
  items: ResaleChatListItem[];
  totalCount: number;
};

export type ResaleChatBadgeCountResponse = {
  unreadCommentCount: number;
};

export type ResaleCommentsMarkAsReadResponse = {
  ok: boolean;
  resaleId: string;
  markedCount: number;
};