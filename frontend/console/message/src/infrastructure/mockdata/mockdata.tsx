// frontend/message/src/infrastructure/mockdata/mockdata.tsx

import type { Message, MessageStatus, ImageRef } from "../../../../shell/src/shared/types/message";

/**
 * モックメッセージデータ
 * backend/internal/domain/message/entity.go と
 * frontend/shell/src/shared/types/message.ts に準拠。
 *
 * - createdAt/updatedAt は ISO8601 形式
 * - status は "draft" | "sent" | "canceled" | "delivered" | "read"
 * - images は空配列（本番では GCS 参照が入る想定）
 */
export const MOCK_MESSAGES: Message[] = [
  {
    id: "msg_001",
    senderId: "system_admin",
    receiverId: "user_001",
    content:
      "2025/11/12(水) 02:00 - 04:00 の間、サーバーメンテナンスを実施します。",
    status: "sent" as MessageStatus,
    images: [] as ImageRef[],
    createdAt: "2025-11-08T09:45:00Z",
    updatedAt: "2025-11-08T09:45:00Z",
    deletedAt: null,
    readAt: null,
    canceledAt: null,
  },
  {
    id: "msg_002",
    senderId: "brand_lumina",
    receiverId: "user_001",
    content:
      "先日アップロードされたデザインデータの確認が完了しました。次の工程へ進めます。",
    status: "read" as MessageStatus,
    images: [] as ImageRef[],
    createdAt: "2025-11-07T16:20:00Z",
    updatedAt: "2025-11-07T17:00:00Z",
    deletedAt: null,
    readAt: "2025-11-07T17:00:00Z",
    canceledAt: null,
  },
  {
    id: "msg_003",
    senderId: "support_team",
    receiverId: "user_001",
    content:
      "お問い合わせいただいた件は解決済みとしてクローズしました。詳細はサポート履歴をご確認ください。",
    status: "read" as MessageStatus,
    images: [] as ImageRef[],
    createdAt: "2025-11-06T10:30:00Z",
    updatedAt: "2025-11-06T11:00:00Z",
    deletedAt: null,
    readAt: "2025-11-06T11:00:00Z",
    canceledAt: null,
  },
];
