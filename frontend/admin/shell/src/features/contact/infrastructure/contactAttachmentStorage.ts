// frontend/admin/shell/src/features/contact/infrastructure/contactAttachmentStorage.ts
import { getDownloadURL, ref } from "firebase/storage";

import { storage } from "../../../auth/infrastructure/firebaseClient";
import type { ContactAttachment } from "../application/contactMessage";

export type ContactAttachmentImage = ContactAttachment & {
  imageUrl: string;
};

export async function loadContactAttachmentImages(
  attachments: ContactAttachment[],
): Promise<ContactAttachmentImage[]> {
  const imageAttachments = attachments.filter((attachment) =>
    attachment.contentType.startsWith("image/"),
  );

  return Promise.all(
    imageAttachments.map(async (attachment) => {
      const imageUrl = await getDownloadURL(
        ref(storage, attachment.storagePath),
      );

      return {
        ...attachment,
        imageUrl,
      };
    }),
  );
}