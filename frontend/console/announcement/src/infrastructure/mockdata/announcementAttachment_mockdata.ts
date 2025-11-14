// frontend/announcement/src/infrastructure/mockdata/announcementAttachment_mockdata.ts

import type {
  AttachmentFile,
} from "../../../../shell/src/shared/types/announcementAttachment";

/**
 * モック用 Announcement Attachment データ
 * frontend/shell/src/shared/types/announcementAttachment.ts に準拠。
 *
 * - announcementId: お知らせID
 * - id: announcementId + fileName から生成された安定ID（簡略化）
 * - fileUrl: 公開アクセス可能なURL（mock用）
 * - bucket / objectPath: GCS上の格納先
 */
export const ANNOUNCEMENT_ATTACHMENTS: AttachmentFile[] = [
  {
    announcementId: "announcement_001",
    id: "attachment_001_pdf",
    fileName: "notice_2024_spring.pdf",
    fileUrl:
      "https://storage.googleapis.com/narratives_development_announcement_attachment/announcements/announcement_001/notice_2024_spring.pdf",
    fileSize: 204800,
    mimeType: "application/pdf",
    bucket: "narratives_development_announcement_attachment",
    objectPath:
      "announcements/announcement_001/notice_2024_spring.pdf",
  },
  {
    announcementId: "announcement_001",
    id: "attachment_002_jpg",
    fileName: "campaign_banner.jpg",
    fileUrl:
      "https://storage.googleapis.com/narratives_development_announcement_attachment/announcements/announcement_001/campaign_banner.jpg",
    fileSize: 512000,
    mimeType: "image/jpeg",
    bucket: "narratives_development_announcement_attachment",
    objectPath:
      "announcements/announcement_001/campaign_banner.jpg",
  },
  {
    announcementId: "announcement_002",
    id: "attachment_003_png",
    fileName: "update_feature_highlight.png",
    fileUrl:
      "https://storage.googleapis.com/narratives_development_announcement_attachment/announcements/announcement_002/update_feature_highlight.png",
    fileSize: 384000,
    mimeType: "image/png",
    bucket: "narratives_development_announcement_attachment",
    objectPath:
      "announcements/announcement_002/update_feature_highlight.png",
  },
  {
    announcementId: "announcement_003",
    id: "attachment_004_webp",
    fileName: "maintenance_schedule.webp",
    fileUrl:
      "https://storage.googleapis.com/narratives_development_announcement_attachment/announcements/announcement_003/maintenance_schedule.webp",
    fileSize: 156000,
    mimeType: "image/webp",
    bucket: "narratives_development_announcement_attachment",
    objectPath:
      "announcements/announcement_003/maintenance_schedule.webp",
  },
  {
    announcementId: "announcement_004",
    id: "attachment_005_txt",
    fileName: "release_notes.txt",
    fileUrl:
      "https://storage.googleapis.com/narratives_development_announcement_attachment/announcements/announcement_004/release_notes.txt",
    fileSize: 9600,
    mimeType: "text/plain",
    bucket: "narratives_development_announcement_attachment",
    objectPath:
      "announcements/announcement_004/release_notes.txt",
  },
];

/**
 * 公開URLをUI表示用に返すヘルパー
 */
export function getAttachmentPublicUrls(): string[] {
  return ANNOUNCEMENT_ATTACHMENTS.map((a) => a.fileUrl);
}

/**
 * AnnouncementID ごとに添付ファイルを取得
 */
export function getAttachmentsByAnnouncementId(
  announcementId: string
): AttachmentFile[] {
  return ANNOUNCEMENT_ATTACHMENTS.filter(
    (a) => a.announcementId === announcementId
  );
}
