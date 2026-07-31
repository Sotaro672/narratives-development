// frontend/console/shell/src/features/announcement/application/announcement_attachment_service.ts

import {
  getDownloadURL,
  getStorage,
  ref as storageRef,
  uploadBytes,
} from "firebase/storage";

import type { AnnouncementAttachmentInput } from "../infrastructure/announcement_repository_http";

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

function normalizeAnnouncementId(
  announcementId: string,
): string {
  const normalizedId = String(
    announcementId ?? "",
  ).trim();

  if (!normalizedId) {
    throw new Error(
      "announcementId is required",
    );
  }

  return normalizedId;
}

function isImageFile(
  value: unknown,
): value is File {
  return (
    typeof File !== "undefined" &&
    value instanceof File &&
    value.type.startsWith("image/")
  );
}

// ============================================================
// Storage path
// ============================================================

function sanitizeStoragePathSegment(
  value: string,
): string {
  const normalizedValue = String(
    value ?? "",
  ).trim();

  if (!normalizedValue) {
    return "file";
  }

  return normalizedValue
    .replace(/[\\/#?[\]*]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/_+/g, "_");
}

function getFileExtension(
  fileName: string,
): string {
  const normalizedFileName = String(
    fileName ?? "",
  ).trim();

  const extensionIndex =
    normalizedFileName.lastIndexOf(".");

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
  const extension = getFileExtension(
    file.name || "image",
  );

  const attachmentId =
    createAnnouncementClientId();

  const displayOrder = String(
    index + 1,
  ).padStart(2, "0");

  return sanitizeStoragePathSegment(
    `${displayOrder}-${attachmentId}${extension}`,
  );
}

function buildAnnouncementAttachmentObjectPath(
  announcementId: string,
  storageFileName: string,
): string {
  return [
    "announcements",
    sanitizeStoragePathSegment(
      announcementId,
    ),
    "attachments",
    sanitizeStoragePathSegment(
      storageFileName,
    ),
  ].join("/");
}

// ============================================================
// Upload
// ============================================================

async function uploadAnnouncementImage({
  announcementId,
  file,
  index,
}: UploadAnnouncementImageParams): Promise<AnnouncementAttachmentInput> {
  const normalizedAnnouncementId =
    normalizeAnnouncementId(
      announcementId,
    );

  const storageFileName =
    buildAnnouncementAttachmentStorageFileName(
      file,
      index,
    );

  const objectPath =
    buildAnnouncementAttachmentObjectPath(
      normalizedAnnouncementId,
      storageFileName,
    );

  const mimeType =
    file.type ||
    "application/octet-stream";

  const storage = getStorage();

  const attachmentRef = storageRef(
    storage,
    objectPath,
  );

  await uploadBytes(
    attachmentRef,
    file,
    {
      contentType: mimeType,
      customMetadata: {
        announcementId:
          normalizedAnnouncementId,
        fileName: storageFileName,
        originalFileName: file.name,
      },
    },
  );

  const fileUrl = await getDownloadURL(
    attachmentRef,
  );

  return {
    fileName: storageFileName,
    fileUrl,
    fileSize: file.size,
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
  const normalizedAnnouncementId =
    normalizeAnnouncementId(
      announcementId,
    );

  const validImages = Array.isArray(images)
    ? images.filter(isImageFile)
    : [];

  if (validImages.length === 0) {
    return [];
  }

  return Promise.all(
    validImages.map((file, index) =>
      uploadAnnouncementImage({
        announcementId:
          normalizedAnnouncementId,
        file,
        index,
      }),
    ),
  );
}