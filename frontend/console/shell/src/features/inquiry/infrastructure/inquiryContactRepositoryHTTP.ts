// frontend/console/shell/src/features/inquiry/infrastructure/inquiryContactRepositoryHTTP.ts

import { ref, uploadBytesResumable } from "firebase/storage";

import { storage } from "../../../auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../shared/http/authHeaders";
import {
  INQUIRY_CONTACT_ATTACHMENT_ROOT_PATH,
  INQUIRY_CONTACT_SOURCE,
} from "../constants/inquiryCreate";

export type InquiryContactUploadProgress = {
  fileIndex: number;
  fileCount: number;
  fileProgress: number;
  totalProgress: number;
};

export type CreateInquiryContactInput = {
  name: string;
  email: string;
  company: string;
  message: string;
  attachmentImageIds: string[];
};

type ContactErrorResponse = {
  error?: string;
};

function createAttachmentImageId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

async function readJsonSafe(
  response: Response,
): Promise<ContactErrorResponse | null> {
  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  try {
    return (await response.json()) as ContactErrorResponse;
  } catch {
    return null;
  }
}

export async function uploadInquiryContactAttachments(
  files: File[],
  onProgress?: (progress: InquiryContactUploadProgress) => void,
): Promise<string[]> {
  if (files.length === 0) {
    return [];
  }

  const fileCount = files.length;
  const totalBytes = files.reduce((total, file) => total + file.size, 0);

  let completedBytes = 0;
  const attachmentImageIds: string[] = [];

  for (let index = 0; index < files.length; index += 1) {
    const file = files[index];
    const imageId = createAttachmentImageId();
    const storageRef = ref(
      storage,
      `${INQUIRY_CONTACT_ATTACHMENT_ROOT_PATH}/${imageId}`,
    );

    await new Promise<void>((resolve, reject) => {
      const uploadTask = uploadBytesResumable(storageRef, file, {
        contentType: file.type || "application/octet-stream",
        customMetadata: {
          originalFileName: file.name,
        },
      });

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
        reject,
        resolve,
      );
    });

    completedBytes += file.size;
    attachmentImageIds.push(imageId);

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

  return attachmentImageIds;
}

export async function createInquiryContactHTTP(
  input: CreateInquiryContactInput,
): Promise<void> {
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${API_BASE}/introduction/contacts`, {
    method: "POST",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({
      name: input.name,
      email: input.email,
      company: input.company,
      message: input.message,
      attachmentImageIds: input.attachmentImageIds,
      source: INQUIRY_CONTACT_SOURCE,
    }),
  });

  const responseBody = await readJsonSafe(response);

  if (!response.ok) {
    throw new Error(
      responseBody?.error || "お問い合わせの送信に失敗しました。",
    );
  }
}