// frontend/src/features/contact/utils/upload.ts
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { storage } from "../../../lib/firebase";
import { CONTACT_ATTACHMENT_ROOT_PATH } from "../constants";
import type {
  ContactAttachmentItem,
  UploadedContactAttachment,
} from "../types";

export function createUploadFolderId() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function createSafeFileName(fileName: string) {
  return fileName.replace(/[^\w.-]/g, "_");
}

export async function uploadContactAttachments({
  attachments,
  ownerId,
}: {
  attachments: ContactAttachmentItem[];
  ownerId: string;
}): Promise<UploadedContactAttachment[]> {
  if (attachments.length === 0) {
    return [];
  }

  const folderId = createUploadFolderId();

  return Promise.all(
    attachments.map(async (item) => {
      const safeFileName = createSafeFileName(item.file.name);
      const storagePath = `${CONTACT_ATTACHMENT_ROOT_PATH}/${ownerId}/${folderId}/${safeFileName}`;
      const storageRef = ref(storage, storagePath);

      const snapshot = await uploadBytes(storageRef, item.file, {
        contentType: item.file.type || "application/octet-stream",
        customMetadata: {
          originalFileName: item.file.name,
        },
      });

      const downloadUrl = await getDownloadURL(snapshot.ref);

      return {
        fileName: item.file.name,
        contentType: item.file.type || "application/octet-stream",
        size: item.file.size,
        storagePath,
        downloadUrl,
      };
    })
  );
}