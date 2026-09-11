// frontend/admin/shell/src/shared/type/resaleReview.ts

export type ResaleReviewCommentKind = "user" | "purchase";

export type ResaleReviewComment = {
  commentId: string;
  resaleId: string;
  avatarId: string;
  kind: ResaleReviewCommentKind;
  body: string;
  deleted: boolean;
  isRead: boolean;
  reportCount: number;
  createdAt: string;
  updatedAt: string;
  avatarName: string;
  avatarIcon: string;
};

export type ResaleReviewResponse = {
  items: ResaleReviewComment[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};