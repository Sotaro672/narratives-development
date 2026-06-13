// frontend/inquiry/src/infrastructure/mockdata/mockdata.tsx
// frontend/shell/src/shared/types/inquiry.ts および inquiryImage.ts を正としてモックデータを更新

import type { Inquiry } from "../../../../shell/src/shared/types/inquiry";
import type { InquiryImage } from "../../../../shell/src/shared/types/inquiryImage";

export const INQUIRIES: Inquiry[] = [
  {
    id: "inq_002",
    avatarId: "avatar_002",
    subject: "デニムジャケットの色落ちについて",
    content:
      "NEXUS Streetのデニムジャケットを洗濯したところ、色落ちが予想以上に強く出ました。洗濯方法やケア方法について教えてください。",
    status: "in_progress",
    inquiryType: "product_description",
    productBlueprintId: "pb_002",
    tokenBlueprintId: "tb_002",
    assigneeId: "member_tanaka",
    imageId: "inqimg_002",
    createdAt: "2024-09-24T10:00:00Z",
    updatedAt: "2024-09-25T09:30:00Z",
    updatedBy: "member_tanaka",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "inq_001",
    avatarId: "avatar_001",
    subject: "シルクブラウスのサイズ交換について",
    content:
      "LUMINA Fashionのプレミアムシルクブラウスを購入しましたが、サイズが合わないため交換を希望します。どのような手順が必要でしょうか？",
    status: "pending",
    inquiryType: "exchange",
    productBlueprintId: "pb_001",
    tokenBlueprintId: "tb_001",
    assigneeId: "member_sato",
    imageId: "inqimg_001",
    createdAt: "2024-09-20T08:00:00Z",
    updatedAt: "2024-09-20T12:00:00Z",
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
  },
];

/**
 * 問い合わせに紐づく画像モックデータ
 * frontend/shell/src/shared/types/inquiryImage.ts に準拠
 */
export const INQUIRY_IMAGES: InquiryImage[] = [
  {
    id: "inqimg_001",
    images: [
      {
        inquiryId: "inq_001",
        fileName: "silk_blouse_size.jpg",
        fileUrl:
          "https://storage.googleapis.com/narratives_development_inquiry_image/silk_blouse_size.jpg",
        fileSize: 245678,
        mimeType: "image/jpeg",
        width: 1024,
        height: 768,
        createdAt: "2024-09-20T08:05:00Z",
        createdBy: "avatar_001",
        updatedAt: null,
        updatedBy: null,
        deletedAt: null,
        deletedBy: null,
      },
    ],
  },
  {
    id: "inqimg_002",
    images: [
      {
        inquiryId: "inq_002",
        fileName: "denim_fade_1.png",
        fileUrl:
          "https://storage.googleapis.com/narratives_development_inquiry_image/denim_fade_1.png",
        fileSize: 385432,
        mimeType: "image/png",
        width: 800,
        height: 600,
        createdAt: "2024-09-24T10:10:00Z",
        createdBy: "avatar_002",
        updatedAt: null,
        updatedBy: null,
        deletedAt: null,
        deletedBy: null,
      },
      {
        inquiryId: "inq_002",
        fileName: "denim_fade_2.png",
        fileUrl:
          "https://storage.googleapis.com/narratives_development_inquiry_image/denim_fade_2.png",
        fileSize: 402345,
        mimeType: "image/png",
        width: 800,
        height: 600,
        createdAt: "2024-09-24T10:12:00Z",
        createdBy: "avatar_002",
        updatedAt: null,
        updatedBy: null,
        deletedAt: null,
        deletedBy: null,
      },
    ],
  },
];

/**
 * 説明:
 * - Inquiry 型および InquiryImage 型に準拠。
 * - inquiryId に紐づいた画像リストを INQUIRY_IMAGES として定義。
 * - 各 fileUrl は GCS 署名付きURL想定 (モックとして公開URLを利用)。
 * - 各 createdAt は ISO8601 形式 (UTC)。
 */
