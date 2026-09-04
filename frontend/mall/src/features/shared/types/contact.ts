// frontend/amol/src/features/shared/types/contact.ts

import type { MediaUploaderItem } from "../../../components/ui/MediaUploader";

export type ContactAttachmentItem = MediaUploaderItem & {
  file: File;
};

export type UploadedContactAttachment = {
  imageId: string;
};