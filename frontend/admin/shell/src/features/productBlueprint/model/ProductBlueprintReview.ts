//frontend\admin\shell\src\features\productBlueprint\model\productBlueprintReview.ts

export type ContractProductBlueprintReviewStatus =
  | "PUBLISHED"
  | "HIDDEN"
  | "REMOVED";

export type ContractProductBlueprintReview = {
  id: string;
  productBlueprintId: string;
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  rating: number;
  title: string;
  body: string;
  helpfulVotes: number;
  totalVotes: number;
  status: ContractProductBlueprintReviewStatus;
  reviewedAt: string;
  createdAt: string;
  updatedAt: string;
  moderationReason?: string;
};

export type ContractProductBlueprintReviewResponse = {
  productBlueprintId: string;
  status: ContractProductBlueprintReviewStatus;
  items: ContractProductBlueprintReview[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};