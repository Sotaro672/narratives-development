// frontend/amol/src/features/token-commnet/types/tokenCommentTypes.ts

export type TokenCommentAuthorType = "avatar" | "brand" | string;

export type TokenCommentReactionType = "comment" | "like" | "dislike" | string;

export type TokenBlueprintReviewAggregate = {
  tokenBlueprintId: string;
  likeCount: number;
  dislikeCount: number;
  topLevelCommentCount: number;
  totalCommentCount: number;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type TokenBlueprintReaction = {
  tokenBlueprintId: string;
  actorId: string;
  actorType: TokenCommentAuthorType;
  type: TokenCommentReactionType;
  createdAt?: string | null;
  updatedAt?: string | null;
  authorAvatarName?: string | null;
  authorAvatarIcon?: string | null;
  brandName?: string | null;
  brandIcon?: string | null;
};

export type TokenBlueprintReactionInput = {
  tokenBlueprintId: string;
  type: "like" | "dislike";
};

export type TokenComment = {
  commentId: string;
  tokenBlueprintId: string;
  parentCommentId: string;
  rootCommentId: string;
  depth: number;
  authorId: string;
  authorType: TokenCommentAuthorType;
  isOwnerComment: boolean;
  body: string;
  likeCount: number;
  dislikeCount: number;
  childCount: number;
  deleted: boolean;
  createdAt: string;
  updatedAt: string;
  authorAvatarName?: string | null;
  authorAvatarIcon?: string | null;
  brandName?: string | null;
  brandIcon?: string | null;
};

export type TokenCommentListResponse = {
  items: TokenComment[];
  page: number;
  perPage: number;
  total: number;
  tokenBlueprintName?: string | null;
  brandName?: string | null;
};

export type TokenCommentTreeNode = {
  comment: TokenComment;
  children: TokenCommentTreeNode[];
};

export type TokenCommentPostInput = {
  tokenBlueprintId: string;
  body: string;
};

export type TokenCommentReplyInput = {
  tokenBlueprintId: string;
  parentCommentId: string;
  body: string;
};

export type TokenCommentVoteInput = {
  tokenBlueprintId: string;
  commentId: string;
};

export function getTokenCommentDisplayName(comment: TokenComment): string {
  if (comment.authorType === "brand") {
    const brandName = comment.brandName?.trim();

    if (brandName) {
      return brandName;
    }
  } else {
    const avatarName = comment.authorAvatarName?.trim();

    if (avatarName) {
      return avatarName;
    }
  }

  return comment.authorId || "unknown";
}

export function getTokenCommentDisplayIconUrl(comment: TokenComment): string {
  if (comment.authorType === "brand") {
    return comment.brandIcon?.trim() || "";
  }

  return comment.authorAvatarIcon?.trim() || "";
}

export function isTopLevelTokenComment(comment: TokenComment): boolean {
  return comment.parentCommentId === "" && comment.depth === 0;
}