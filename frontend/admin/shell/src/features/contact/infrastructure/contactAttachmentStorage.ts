// frontend/admin/shell/src/features/contact/infrastructure/contactAttachmentStorage.ts
import {
  getBlob,
  getDownloadURL,
  getMetadata,
  ref,
} from "firebase/storage";

import { storage } from "../../../auth/infrastructure/firebaseClient";
import type { ContactAttachmentImage } from "../../../shared/type/contact";

const CONTACT_ATTACHMENT_ROOT_PATH = "contact-attachments";

function createContactAttachmentRef(imageId: string) {
  return ref(storage, `${CONTACT_ATTACHMENT_ROOT_PATH}/${imageId}`);
}

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
      const storageRef = createContactAttachmentRef(imageId);
      const [imageUrl, metadata] = await Promise.all([
        getDownloadURL(storageRef),
        getMetadata(storageRef),
      ]);

      return {
        imageId,
        imageUrl,
        fileName: metadata.customMetadata?.originalFileName || imageId,
        contentType: metadata.contentType || "application/octet-stream",
      };
    }),
  );
}

export async function downloadContactAttachment(
  imageId: string,
  fileName: string,
): Promise<void> {
  const normalizedImageId = imageId.trim();

  if (!normalizedImageId) {
    throw new Error("imageId is required.");
  }

  const storageRef = createContactAttachmentRef(normalizedImageId);
  const blob = await getBlob(storageRef);
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement("a");

  anchor.href = objectUrl;
  anchor.download = fileName || normalizedImageId;
  anchor.style.display = "none";

  document.body.appendChild(anchor);

  try {
    anchor.click();
  } finally {
    anchor.remove();
    URL.revokeObjectURL(objectUrl);
  }
}