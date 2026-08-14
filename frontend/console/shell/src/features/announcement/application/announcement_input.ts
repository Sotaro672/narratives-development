// frontend/console/shell/src/features/announcement/application/announcement_input.ts

import type { AnnouncementAttachmentFile } from "../../../shared/types/announcements";

export type AnnouncementInputAttachment =
  | (AnnouncementAttachmentFile & {
      type: "existing";
    })
  | {
      type: "new";
      file: File;
    };

export type AnnouncementInputPayload = {
  title: string;
  text: string;
  attachments: AnnouncementInputAttachment[];
};