// frontend/message/src/pages/messageManagement.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { MOCK_MESSAGES, type Message } from "../../mockdata";

export default function MessageManagementPage() {
  const navigate = useNavigate();
  const [rows] = React.useState<Message[]>(MOCK_MESSAGES);

  // 戻るボタンの処理
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="single"
      title="メッセージ管理"
      onBack={onBack} // ✅ PageHeaderに「戻る」ボタンを追加
    >
      <div className="message-list">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-[0.75rem] text-muted-foreground">
              <th className="text-left py-2 px-2 w-24">ID</th>
              <th className="text-left py-2 px-2 w-40">送信者</th>
              <th className="text-left py-2 px-2">件名</th>
              <th className="text-left py-2 px-2 w-32">状態</th>
              <th className="text-left py-2 px-2 w-48">受信日時</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((m) => (
              <tr
                key={m.id}
                className={`border-b last:border-b-0 ${
                  m.status === "未読" ? "bg-yellow-50" : ""
                }`}
              >
                <td className="py-2 px-2 text-xs text-blue-600">{m.id}</td>
                <td className="py-2 px-2 text-xs">{m.sender}</td>
                <td className="py-2 px-2">
                  <div className="font-medium">{m.subject}</div>
                  <div className="text-xs text-muted-foreground line-clamp-1">
                    {m.body}
                  </div>
                </td>
                <td className="py-2 px-2 text-xs">
                  {m.status === "未読" ? (
                    <span className="inline-flex items-center px-2 py-1 rounded-full bg-yellow-100 text-yellow-800">
                      未読
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2 py-1 rounded-full bg-slate-100 text-slate-600">
                      既読
                    </span>
                  )}
                </td>
                <td className="py-2 px-2 text-xs">{m.receivedAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </PageStyle>
  );
}
