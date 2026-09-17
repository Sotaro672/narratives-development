// frontend/console/shell/src/features/announcement/application/announcement_attachment_service.ts

import {
  getDownloadURL,
  getStorage,
  ref as storageRef,
  uploadBytesResumable,
} from "firebase/storage";

import type {
  AnnouncementAttachmentInput,
} from "../../../shared/types/announcements";

export type AnnouncementImageUploadProgress = {
  fileName: string;
  transferredBytes: number;
  totalBytes: number;
  percentage: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

export type AnnouncementImageUploadProgressHandler = (
  progress: AnnouncementImageUploadProgress,
) => void;

type UploadAnnouncementImageParams = {
  announcementId: string;
  file: File;
  index: number;
  onProgress?: (transferredBytes: number, totalBytes: number) => void;
};

export type UploadAnnouncementImagesParams = {
  announcementId: string;
  images: File[];
  onProgress?: AnnouncementImageUploadProgressHandler;
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

function isImageFile(value: unknown): value is File {
  return (
    typeof File !== "undefined" &&
    value instanceof File &&
    value.type.startsWith("image/")
  );
}

// ============================================================
// Storage path
// ============================================================

function getFileExtension(fileName: string): string {
  const normalizedFileName = String(fileName ?? "").trim();
  const extensionIndex = normalizedFileName.lastIndexOf(".");

  if (
    extensionIndex < 0 ||
    extensionIndex === normalizedFileName.length - 1
  ) {
    return "";
  }

  return normalizedFileName.slice(extensionIndex);
}

function buildAnnouncementAttachmentStorageFileName(
  file: File,
  index: number,
): string {
  const extension = getFileExtension(file.name || "image");
  const attachmentId = createAnnouncementClientId();
  const displayOrder = String(index + 1).padStart(2, "0");

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
  ].join("/");
}

// ============================================================
// Progress
// ============================================================

function calculateUploadPercentage(
  transferredBytes: number,
  totalBytes: number,
): number {
  if (totalBytes <= 0) {
    return 0;
  }

  return Math.min(
    100,
    Math.max(
      0,
      Math.round((transferredBytes / totalBytes) * 100),
    ),
  );
}

// ============================================================
// Upload
// ============================================================

async function uploadAnnouncementImage({
  announcementId,
  file,
  index,
  onProgress,
}: UploadAnnouncementImageParams): Promise<AnnouncementAttachmentInput> {
  const storageFileName =
    buildAnnouncementAttachmentStorageFileName(file, index);

  const objectPath =
    buildAnnouncementAttachmentObjectPath(
      announcementId,
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

  onProgress?.(0, file.size);

  const uploadTask = uploadBytesResumable(
    attachmentRef,
    file,
    {
      contentType: mimeType,
      customMetadata: {
        announcementId,
        fileName: storageFileName,
        originalFileName: file.name,
      },
    },
  );

  await new Promise<void>((resolve, reject) => {
    uploadTask.on(
      "state_changed",
      (snapshot) => {
        onProgress?.(
          snapshot.bytesTransferred,
          snapshot.totalBytes,
        );
      },
      (error) => {
        reject(error);
      },
      () => {
        const snapshot = uploadTask.snapshot;

        onProgress?.(
          snapshot.totalBytes,
          snapshot.totalBytes,
        );

        resolve();
      },
    );
  });

  const snapshot = uploadTask.snapshot;
  const fileUrl = await getDownloadURL(snapshot.ref);

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
  onProgress,
}: UploadAnnouncementImagesParams): Promise<
  AnnouncementAttachmentInput[]
> {
  const validImages = Array.isArray(images)
    ? images.filter(isImageFile)
    : [];

  if (validImages.length === 0) {
    return [];
  }

  const totalBytes = validImages.reduce(
    (total, file) => total + file.size,
    0,
  );

  const expectedUploadCount = validImages.length;
  const attachments: AnnouncementAttachmentInput[] = [];

  let completedBytes = 0;
  let completedUploadCount = 0;

  for (
    let index = 0;
    index < validImages.length;
    index += 1
  ) {
    const file = validImages[index];

    if (!file) {
      continue;
    }

    const attachment = await uploadAnnouncementImage({
      announcementId,
      file,
      index,
      onProgress: (fileTransferredBytes) => {
        const transferredBytes = Math.min(
          totalBytes,
          completedBytes + fileTransferredBytes,
        );

        onProgress?.({
          fileName: file.name,
          transferredBytes,
          totalBytes,
          percentage: calculateUploadPercentage(
            transferredBytes,
            totalBytes,
          ),
          completedUploadCount,
          expectedUploadCount,
        });
      },
    });

    attachments.push(attachment);

    completedBytes = Math.min(
      totalBytes,
      completedBytes + file.size,
    );

    completedUploadCount += 1;

    onProgress?.({
      fileName: file.name,
      transferredBytes: completedBytes,
      totalBytes,
      percentage: calculateUploadPercentage(
        completedBytes,
        totalBytes,
      ),
      completedUploadCount,
      expectedUploadCount,
    });
  }

  return attachments;
}