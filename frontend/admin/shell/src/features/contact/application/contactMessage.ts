// frontend/admin/shell/src/features/contact/application/contactMessage.ts

const ATTACHMENT_MARKER = "--- 添付ファイル ---";

export type ContactAttachment = {
  fileName: string;
  storagePath: string;
  contentType: string;
  size: number | null;
};

export type ParsedContactMessage = {
  message: string;
  attachments: ContactAttachment[];
};

export function parseContactMessage(value: string): ParsedContactMessage {
  const markerIndex = value.indexOf(ATTACHMENT_MARKER);

  if (markerIndex < 0) {
    return {
      message: value.trim(),
      attachments: [],
    };
  }

  const message = value.slice(0, markerIndex).trim();
  const attachmentSection = value
    .slice(markerIndex + ATTACHMENT_MARKER.length)
    .trim();

  if (!attachmentSection) {
    return {
      message,
      attachments: [],
    };
  }

  const attachments = attachmentSection
    .split(/\n\s*\n/)
    .map(parseAttachmentBlock)
    .filter(
      (attachment): attachment is ContactAttachment =>
        attachment !== null,
    );

  return {
    message,
    attachments,
  };
}

function parseAttachmentBlock(
  block: string,
): ContactAttachment | null {
  const lines = block
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  if (lines.length === 0) {
    return null;
  }

  const fileName = lines[0].replace(/^\d+\.\s*/, "").trim();
  const storagePath = findLineValue(lines, "Storage Path:");

  if (!fileName || !storagePath) {
    return null;
  }

  const contentType =
    findLineValue(lines, "Content Type:") ||
    "application/octet-stream";

  const rawSize = findLineValue(lines, "Size:");
  const parsedSize = Number(rawSize);

  return {
    fileName,
    storagePath,
    contentType,
    size:
      rawSize !== "" &&
      Number.isFinite(parsedSize) &&
      parsedSize >= 0
        ? parsedSize
        : null,
  };
}

function findLineValue(
  lines: string[],
  prefix: string,
): string {
  const line = lines.find((candidate) =>
    candidate.startsWith(prefix),
  );

  return line ? line.slice(prefix.length).trim() : "";
}