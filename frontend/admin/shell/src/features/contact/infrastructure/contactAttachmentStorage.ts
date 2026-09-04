// frontend/admin/shell/src/features/contact/infrastructure/contactAttachmentStorage.ts
import { getDownloadURL, ref } from "firebase/storage";

import { storage } from "../../../auth/infrastructure/firebaseClient";
import type { ContactAttachmentImage } from "../../../shared/type/contact";

const CONTACT_ATTACHMENT_ROOT_PATH = "contact-attachments";

export async function loadContactAttachmentImages(
  imageIds: string[],
): Promise<ContactAttachmentImage[]> {
  const normalizedImageIds = Array.from(
    new Set(
      imageIds
        .map((imageId) => imageId.trim())
        .filter((imageId) => imageId !== ""),
    ),
  );

  return Promise.all(
    normalizedImageIds.map(async (imageId) => {
      const imageUrl = await getDownloadURL(
        ref(storage, `${CONTACT_ATTACHMENT_ROOT_PATH}/${imageId}`),
      );

      return {
        imageId,
        imageUrl,
      };
    }),
  );
}