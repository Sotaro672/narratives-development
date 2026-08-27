// frontend/amol/src/features/inquiry/api/inquiryApi.tsx

export {
  uploadInquiryImage,
  uploadReplyImage,
} from "./inquiryImageApi";

export {
  getInquiryBadgeCount,
  listMeInquiries,
} from "./inquiryListApi";

export {
  getInquiry,
  listInquiryReplies,
} from "./inquiryThreadApi";

export {
  closeInquiry,
  createInquiry,
  markInquiryAsRead,
  replyInquiry,
} from "./inquiryMutationApi";

export type {
  CreateInquiryRequest,
  GetInquiryBadgeCountParams,
  Inquiry,
  InquiryBadgeCountResponse,
  InquiryImage,
  InquiryImageUpload,
  InquiryListItem,
  InquiryReply,
  InquiryReplySenderType,
  InquiryStatus,
  InquiryType,
  ListMeInquiriesParams,
  ListMeInquiriesResult,
  ReplyInquiryRequest,
  UploadInquiryImageParams,
  UploadReplyImageParams,
} from "../../shared/types/inquiryTypes";