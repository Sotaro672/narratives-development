// frontend/console/shell/src/shared/types/inquiry.ts    
    
import type { ShippingAddress } from "./shippingAddress";    
    
export type InquiryStatus = "open" | "resolved" | "closed";    
    
export type InquiryReplySenderType = "avatar" | "member";    
    
export const INQUIRY_TYPES = [    
  "product",    
  "return_unopened",    
  "return_opened",    
] as const;    
    
export type InquiryType = (typeof INQUIRY_TYPES)[number];    
    
export function getInquiryTypeLabel(    
  inquiryType: InquiryType,    
): string {    
  switch (inquiryType) {    
    case "product":    
      return "商品説明";    
    
    case "return_unopened":    
      return "未開封返品";    
    
    case "return_opened":    
      return "開封後返品";    
    
    default:    
      return inquiryType;    
  }    
}    
    
export type InquiryImageFile = {    
  inquiryId: string;    
  fileName: string;    
  fileUrl: string;    
  objectPath?: string;    
  fileSize: number;    
  mimeType: string;    
  createdAt: string;    
  createdBy: string;    
  updatedAt?: string;    
  updatedBy?: string;    
  deletedAt?: string;    
  deletedBy?: string;    
};    
    
export type InquiryReply = {    
  id: string;    
  inquiryId: string;    
  senderType: InquiryReplySenderType;    
  senderId: string;    
  content: string;    
  isRead: boolean;    
  images?: InquiryImageFile[];    
  createdAt: string;    
  createdBy: string;    
  updatedAt?: string;    
  updatedBy?: string;    
  deletedAt?: string;    
  deletedBy?: string;    
};    
    
export type Inquiry = {    
  id: string;    
  productId?: string;    
  orderId?: string;    
  orderItemIndex?: number;    
  avatarId: string;    
  subject?: string;    
  content: string;    
  status: InquiryStatus;    
  inquiryType: InquiryType;    
  isRead: boolean;    
  images?: InquiryImageFile[];    
  createdAt: string;    
  updatedAt: string;    
  updatedBy?: string;    
  deletedAt?: string;    
  deletedBy?: string;    
  resolvedAt?: string;    
  resolvedBy?: string;    
  closedAt?: string;    
  closedBy?: string;    
};    
    
export type InquiryOrderItemType = "list" | "resale";    
    
export type InquiryOrderItemSummary = {    
  itemIndex: number;    
  itemType: InquiryOrderItemType;    
  modelId: string;    
  inventoryId: string;    
  listId: string;    
  resaleId: string;    
  productId: string;    
  productBlueprintId: string;    
  tokenBlueprintId: string;    
  tokenName: string;    
  brandId: string;    
  qty: number;    
  price: number;    
  isCancelled: boolean;    
  isDispatched: boolean;    
  isReturnRequested: boolean;    
  returnRequestedAt?: string;    
  tokenTransferVerifiedAt?: string;    
  transferred: boolean;    
  transferredAt?: string;    
};    
    
export type InquiryOrderSummary = {    
  id: string;    
  userId: string;    
  avatarId: string;    
  cartId: string;    
  paid: boolean;    
  items: InquiryOrderItemSummary[];    
  createdAt: string;    
};    
    
export type InquiryManagementItem = {    
  inquiry: Inquiry;    
  modelId: string;    
  productBlueprintId: string;    
  productName: string;    
  brandId: string;    
  brandName: string;    
  userFullName: string;    
  companyId: string;    
};    
    
export type InquiryDetail = {    
  inquiry: Inquiry;    
  replies: InquiryReply[];    
  modelId: string;    
  productBlueprintId: string;    
  productName: string;    
  brandId: string;    
  brandName: string;    
  assetId: string;    
  transferredAt?: string;    
  avatarName: string;    
  userId: string;    
  userFullName: string;    
  shippingAddresses: ShippingAddress[];    
  orders: InquiryOrderSummary[];    
  companyId: string;    
};    
    
export type InquiryActionRequiredCountResult = {    
  count: number;    
};    
    
export type ListInquiriesParams = {    
  companyId: string;    
  searchQuery?: string;    
  productId?: string;    
  orderId?: string;    
  avatarId?: string;    
  status?: InquiryStatus;    
  inquiryType?: InquiryType;    
  updatedBy?: string;    
  deletedBy?: string;    
  resolvedBy?: string;    
  closedBy?: string;    
  imageFileName?: string;    
  deleted?: boolean;    
  resolved?: boolean;    
  closed?: boolean;    
};    
    
export type ReplyInquiryParams = {    
  content: string;    
  images: InquiryImageFile[];    
};