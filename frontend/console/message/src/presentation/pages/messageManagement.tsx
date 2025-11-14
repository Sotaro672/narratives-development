// frontend/message/src/presentation/pages/messageManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { MOCK_MESSAGES } from "../../infrastructure/mockdata/mockdata"; // ✅ モックデータのみ
import {
  type Message,
  type MessageStatus,
  isValidMessageStatus,
} from "../../../../shell/src/shared/types/message";

/**
 * メッセージ管理ページ
 * - frontend/shell/src/shared/types/message.ts に準拠
 * - 送信者、内容、状態、作成日時を一覧表示
 */
export default function MessageManagementPage() {
  const navigate = useNavigate();
  const [rows] = React.useState<Message[]>(MOCK_MESSAGES);

  // 戻るボタン処理
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ステータスラベル変換
  const statusLabel = (status: MessageStatus) => {
    switch (status) {
      case "draft":
        return { label: "下書き", color: "bg-gray-100 text-gray-600" };
      case "sent":
        return { label: "送信済み", color: "bg-blue-100 text-blue-700" };
      case "delivered":
        return { label: "配信完了", color: "bg-green-100 text-green-700" };
      case "read":
        return { label: "既読", color: "bg-slate-100 text-slate-600" };
      case "canceled":
        return { label: "キャンセル", color: "bg-red-100 text-red-700" };
      default:
        return { label: "不明", color: "bg-gray-100 text-gray-400" };
    }
  };

  // ISO8601 → ローカル日時表示
  const formatDateTime = (iso?: string | null): string => {
    if (!iso) return "-";
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;
    return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    })}`;
  };

  return (
    <PageStyle layout="single" title="メッセージ管理" onBack={onBack}>
      <div className="message-list overflow-x-auto">
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b text-[0.75rem] text-muted-foreground">
              <th className="text-left py-2 px-2 w-24">ID</th>
              <th className="text-left py-2 px-2 w-40">送信者ID</th>
              <th className="text-left py-2 px-2">内容</th>
              <th className="text-left py-2 px-2 w-32">状態</th>
              <th className="text-left py-2 px-2 w-48">作成日時</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((m) => {
              const status: MessageStatus = isValidMessageStatus(m.status)
                ? m.status
                : "draft";
              const st = statusLabel(status);

              return (
                <tr
                  key={m.id}
                  className={`border-b last:border-b-0 hover:bg-slate-50 transition-colors ${
                    status === "draft" ? "bg-yellow-50" : ""
                  }`}
                >
                  <td className="py-2 px-2 text-xs text-blue-600">
                    {m.id}
                  </td>
                  <td className="py-2 px-2 text-xs">{m.senderId}</td>
                  <td className="py-2 px-2">
                    <div className="text-xs text-muted-foreground line-clamp-2">
                      {m.content}
                    </div>
                  </td>
                  <td className="py-2 px-2 text-xs">
                    <span
                      className={`inline-flex items-center px-2 py-1 rounded-full ${st.color}`}
                    >
                      {st.label}
                    </span>
                  </td>
                  <td className="py-2 px-2 text-xs">
                    {formatDateTime(m.createdAt)}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </PageStyle>
  );
}
