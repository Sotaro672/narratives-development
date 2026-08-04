// frontend/amol/src/features/inquiry/api/inquiryApi.tsx

export {
  uploadInquiryImage,
  uploadReplyImage,
} from "./inquiryImageApi";

export {
  fetchMeInquiries,
  getUnreadInquiryCount,
  listMeInquiries,
} from "./inquiryListApi";

export {
  getInquiry,
  getInquiryThread,
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
  InquiryThread,
  ListMeInquiriesParams,
  ListMeInquiriesResult,
  ReplyInquiryRequest,
  UploadInquiryImageParams,
  UploadReplyImageParams,
} from "../types/inquiryTypes";