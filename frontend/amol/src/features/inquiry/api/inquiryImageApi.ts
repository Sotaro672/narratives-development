// frontend/amol/src/features/inquiry/api/inquiryImageApi.ts

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import {
  storage,
} from "../../../lib/firebase";

import type {
  InquiryImage,
  UploadInquiryImageParams,
  UploadReplyImageParams,
} from "../../shared/types/inquiryTypes";

type UploadInquiryImageFileParams = {
  directoryPath: string;
  file: File;
};

function createUploadImageId(
  file: File,
): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  ) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(
  fileName: string,
): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(
    /[^\w.\-()]/g,
    "_",
  );
}

async function uploadInquiryImageFile({
  directoryPath,
  file,
}: UploadInquiryImageFileParams): Promise<InquiryImage> {
  const imageId =
    createUploadImageId(file);

  const safeFileName =
    sanitizeStorageFileName(
      file.name,
    );

  const objectPath =
    `${directoryPath}/${imageId}/${safeFileName}`;

  const storageRef = ref(
    storage,
    objectPath,
  );

  const mimeType =
    file.type ||
    "application/octet-stream";

  await uploadBytes(
    storageRef,
    file,
    {
      contentType: mimeType,
    },
  );

  const fileUrl =
    await getDownloadURL(
      storageRef,
    );

  return {
    fileName: file.name,
    fileUrl,
    objectPath,
    fileSize: file.size,
    mimeType,
    createdAt:
      new Date().toISOString(),
  };
}

export async function uploadInquiryImage(
  params: UploadInquiryImageParams,
): Promise<InquiryImage> {
  return uploadInquiryImageFile({
    directoryPath:
      `inquiry-images/${params.productId}`,
    file: params.file,
  });
}

export async function uploadReplyImage(
  params: UploadReplyImageParams,
): Promise<InquiryImage> {
  return uploadInquiryImageFile({
    directoryPath:
      `inquiry-replies/${params.inquiryId}`,
    file: params.file,
  });
}