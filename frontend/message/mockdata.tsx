// frontend/message/mockdata.tsx
export type Message = {
  id: string;
  sender: string;
  subject: string;
  body: string;
  receivedAt: string; // YYYY/MM/DD HH:mm
  status: "未読" | "既読";
};

export const MOCK_MESSAGES: Message[] = [
  {
    id: "msg_001",
    sender: "システム管理部",
    subject: "定期メンテナンスのお知らせ",
    body: "2025/11/12(水) 02:00 - 04:00 の間、サーバーメンテナンスを実施します。",
    receivedAt: "2025/11/08 09:45",
    status: "未読",
  },
  {
    id: "msg_002",
    sender: "LUMINA Fashion",
    subject: "商品登録に関するご確認",
    body: "先日アップロードされたデザインデータの確認が完了しました。次の工程へ進めます。",
    receivedAt: "2025/11/07 16:20",
    status: "既読",
  },
  {
    id: "msg_003",
    sender: "サポートチーム",
    subject: "問い合わせ対応完了",
    body: "お問い合わせいただいた件は解決済みとしてクローズしました。詳細はサポート履歴をご確認ください。",
    receivedAt: "2025/11/06 10:30",
    status: "既読",
  },
];
