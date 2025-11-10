// frontend/inquiry/src/infrastructure/mockdata/mockdata.tsx
// frontend/shell/src/shared/types/inquiry.ts を正としてモックデータを更新

import type { Inquiry } from "../../../../shell/src/shared/types/inquiry";

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
    imageId: null,
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
    imageId: null,
    createdAt: "2024-09-20T08:00:00Z",
    updatedAt: "2024-09-20T12:00:00Z",
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
  },
];

/**
 * 説明:
 * - Inquiry 型（backend/internal/domain/inquiry/entity.go の Mirror）に準拠
 * - status は内部値 ("pending" | "in_progress" など) を採用
 * - inquiryType は "exchange" | "product_description" など識別用文字列
 * - 日付は ISO8601 (UTC) 形式で保持
 */
