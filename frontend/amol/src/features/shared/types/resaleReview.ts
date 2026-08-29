//frontend\amol\src\features\shared\types\resaleReview.ts
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