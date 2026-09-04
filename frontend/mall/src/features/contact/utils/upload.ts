// frontend/mall/src/features/contact/utils/upload.ts
import {
  ref,
  uploadBytesResumable,
} from "firebase/storage";

import { storage } from "../../../lib/firebase";
import { CONTACT_ATTACHMENT_ROOT_PATH } from "../constants";
import type {
  ContactAttachmentItem,
  UploadedContactAttachment,
} from "../../shared/types/contact";

export type ContactAttachmentUploadProgress = {
  fileIndex: number;
  fileCount: number;
  fileProgress: number;
  totalProgress: number;
};

type UploadContactAttachmentsParams = {
  attachments: ContactAttachmentItem[];
  onProgress?: (progress: ContactAttachmentUploadProgress) => void;
};

export function createContactAttachmentImageId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export async function uploadContactAttachments({
  attachments,
  onProgress,
}: UploadContactAttachmentsParams): Promise<UploadedContactAttachment[]> {
  if (attachments.length === 0) {
    return [];
  }

  const fileCount = attachments.length;
  const totalBytes = attachments.reduce(
    (total, attachment) => total + attachment.file.size,
    0,
  );

  let completedBytes = 0;
  const uploadedAttachments: UploadedContactAttachment[] = [];

  onProgress?.({
    fileIndex: 1,
    fileCount,
    fileProgress: 0,
    totalProgress: 0,
  });

  for (let index = 0; index < attachments.length; index += 1) {
    const attachment = attachments[index];
    const imageId = createContactAttachmentImageId();
    const storageRef = ref(
      storage,
      `${CONTACT_ATTACHMENT_ROOT_PATH}/${imageId}`,
    );

    await new Promise<void>((resolve, reject) => {
      const uploadTask = uploadBytesResumable(
        storageRef,
        attachment.file,
        {
          contentType:
            attachment.file.type || "application/octet-stream",
          customMetadata: {
            originalFileName: attachment.file.name,
          },
        },
      );

      uploadTask.on(
        "state_changed",
        (snapshot) => {
          const fileProgress =
            snapshot.totalBytes > 0
              ? Math.round(
                  (snapshot.bytesTransferred / snapshot.totalBytes) * 100,
                )
              : 100;

          const totalTransferred =
            completedBytes + snapshot.bytesTransferred;

          const totalProgress =
            totalBytes > 0
              ? Math.round((totalTransferred / totalBytes) * 100)
              : 100;

          onProgress?.({
            fileIndex: index + 1,
            fileCount,
            fileProgress,
            totalProgress,
          });
        },
        (error) => {
          reject(error);
        },
        () => {
          resolve();
        },
      );
    });

    completedBytes += attachment.file.size;

    uploadedAttachments.push({
      imageId,
    });

    onProgress?.({
      fileIndex: index + 1,
      fileCount,
      fileProgress: 100,
      totalProgress:
        totalBytes > 0
          ? Math.round((completedBytes / totalBytes) * 100)
          : 100,
    });
  }

  onProgress?.({
    fileIndex: fileCount,
    fileCount,
    fileProgress: 100,
    totalProgress: 100,
  });

  return uploadedAttachments;
}