// frontend/console/shell/src/features/announcement/application/announcement_attachment_service.ts

import {
  getDownloadURL,
  getStorage,
  ref as storageRef,
  uploadBytes,
} from "firebase/storage";

import type {
  AnnouncementAttachmentInput,
} from "../../../shared/types/announcements";

type UploadAnnouncementImageParams = {
  announcementId: string;
  file: File;
  index: number;
};

export type UploadAnnouncementImagesParams = {
  announcementId: string;
  images: File[];
};

// ============================================================
// Client ID
// ============================================================

export function createAnnouncementClientId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random()
    .toString(36)
    .slice(2, 12)}`;
}

// ============================================================
// Validation
// ============================================================

function isImageFile(
  value: unknown,
): value is File {
  return (
    typeof File !== "undefined" &&
    value instanceof File &&
    value.type.startsWith(
      "image/",
    )
  );
}

// ============================================================
// Storage path
// ============================================================

function getFileExtension(
  fileName: string,
): string {
  const normalizedFileName =
    String(
      fileName ?? "",
    ).trim();

  const extensionIndex =
    normalizedFileName.lastIndexOf(
      ".",
    );

  if (
    extensionIndex < 0 ||
    extensionIndex ===
      normalizedFileName.length - 1
  ) {
    return "";
  }

  return normalizedFileName.slice(
    extensionIndex,
  );
}

function buildAnnouncementAttachmentStorageFileName(
  file: File,
  index: number,
): string {
  const extension =
    getFileExtension(
      file.name || "image",
    );

  const attachmentId =
    createAnnouncementClientId();

  const displayOrder =
    String(
      index + 1,
    ).padStart(
      2,
      "0",
    );

  return `${displayOrder}-${attachmentId}${extension}`;
}

function buildAnnouncementAttachmentObjectPath(
  announcementId: string,
  storageFileName: string,
): string {
  return [
    "announcements",
    announcementId,
    "attachments",
    storageFileName,
  ].join(
    "/",
  );
}

// ============================================================
// Upload
// ============================================================

async function uploadAnnouncementImage({
  announcementId,
  file,
  index,
}: UploadAnnouncementImageParams): Promise<
  AnnouncementAttachmentInput
> {
  const storageFileName =
    buildAnnouncementAttachmentStorageFileName(
      file,
      index,
    );

  const objectPath =
    buildAnnouncementAttachmentObjectPath(
      announcementId,
      storageFileName,
    );

  const mimeType =
    file.type ||
    "application/octet-stream";

  const storage =
    getStorage();

  const attachmentRef =
    storageRef(
      storage,
      objectPath,
    );

  await uploadBytes(
    attachmentRef,
    file,
    {
      contentType:
        mimeType,

      customMetadata: {
        announcementId,

        fileName:
          storageFileName,

        originalFileName:
          file.name,
      },
    },
  );

  const fileUrl =
    await getDownloadURL(
      attachmentRef,
    );

  return {
    fileName:
      storageFileName,

    fileUrl,

    fileSize:
      file.size,

    mimeType,

    objectPath,
  };
}

export async function uploadAnnouncementImages({
  announcementId,
  images,
}: UploadAnnouncementImagesParams): Promise<
  AnnouncementAttachmentInput[]
> {
  const validImages =
    Array.isArray(
      images,
    )
      ? images.filter(
          isImageFile,
        )
      : [];

  if (
    validImages.length === 0
  ) {
    return [];
  }

  return Promise.all(
    validImages.map(
      (
        file,
        index,
      ) =>
        uploadAnnouncementImage({
          announcementId,
          file,
          index,
        }),
    ),
  );
}