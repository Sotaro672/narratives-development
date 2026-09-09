// frontend/admin/shell/src/shared/type/contractTokenBlueprintReview.ts

export type ContractTokenBlueprintReviewAuthorType =
  | "avatar"
  | "brand";

export type ContractTokenBlueprintReview = {
  commentId: string;
  tokenBlueprintId: string;
  parentCommentId: string;
  rootCommentId: string;
  depth: number;
  authorId: string;
  authorType: ContractTokenBlueprintReviewAuthorType;
  authorName: string;
  authorIcon: string;
  body: string;
  likeCount: number;
  dislikeCount: number;
  childCount: number;
  deleted: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ContractTokenBlueprintReviewResponse = {
  tokenBlueprintId: string;
  items: ContractTokenBlueprintReview[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};