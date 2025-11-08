// frontend/inquiry/mockdata.tsx

export type InquiryRow = {
  id: string;
  title: string;
  body: string;
  user: string;
  status: "対応中" | "未対応";
  type: "商品説明" | "交換";
  owner: string;
  inquiredAt: string; // YYYY/M/D
  answeredAt: string; // YYYY/M/D or "-"
};

export const INQUIRIES: InquiryRow[] = [
  {
    id: "inquiry_002",
    title: "デニムジャケットの色落ちについて",
    body: "NEXUS Streetのデニムジャケットを洗濯したら色落ちし…",
    user: "Style Yuki",
    status: "対応中",
    type: "商品説明",
    owner: "田中 雄太",
    inquiredAt: "2024/9/24",
    answeredAt: "2024/9/25",
  },
  {
    id: "inquiry_001",
    title: "シルクブラウスのサイズ交換について",
    body: "LUMINA Fashionのプレミアムシルクブラウスを購入しまし…",
    user: "Creator Alice",
    status: "未対応",
    type: "交換",
    owner: "佐藤 美咲",
    inquiredAt: "2024/9/20",
    answeredAt: "2024/9/20",
  },
];
