// frontend/src/features/contact/types.ts
import type { MediaUploaderItem } from "../../components/ui/MediaUploader";

export type ContactAttachmentItem = MediaUploaderItem & {
  file: File;
};

export type UploadedContactAttachment = {
  fileName: string;
  contentType: string;
  size: number;
  storagePath: string;
  downloadUrl: string;
};