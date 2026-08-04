// frontend/amol/src/features/inquiry/api/inquiryApi.tsx

export {
  uploadInquiryImage,
  uploadReplyImage,
} from "./inquiryImageApi";

export {
  getUnreadInquiryCount,
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
  GetUnreadInquiryCountParams,
  Inquiry,
  InquiryImage,
  InquiryReply,
  ListMeInquiriesParams,
  ListMeInquiriesResult,
  ReplyInquiryRequest,
  UploadInquiryImageParams,
  UploadReplyImageParams,
} from "../../shared/types/inquiryTypes";