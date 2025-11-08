// frontend/transaction/mockdata.tsx

export type Transaction = {
  datetime: string; // "YYYY/M/D HH:mm:ss"
  brand: string;
  type: "受取" | "送金";
  description: string;
  amount: number; // マイナスは出金
  counterparty: string;
};

export const TRANSACTIONS: Transaction[] = [
  {
    datetime: "2024/3/15 23:32:00",
    brand: "LUMINA Fashion",
    type: "受取",
    description: "商品購入代金",
    amount: 125000,
    counterparty: "株式会社○○商事",
  },
  {
    datetime: "2024/3/15 22:15:00",
    brand: "NEXUS Street",
    type: "送金",
    description: "サプライヤー支払い",
    amount: -89000,
    counterparty: "△△サプライヤー",
  },
  {
    datetime: "2024/3/15 20:45:00",
    brand: "LUMINA Fashion",
    type: "受取",
    description: "VIP会員購入",
    amount: 230000,
    counterparty: "VIP会員様",
  },
  {
    datetime: "2024/3/15 19:20:00",
    brand: "NEXUS Street",
    type: "受取",
    description: "トークン販売",
    amount: 54000,
    counterparty: "NFT購入者",
  },
  {
    datetime: "2024/3/15 18:10:00",
    brand: "NEXUS Street",
    type: "受取",
    description: "コラボ商品売上",
    amount: 156000,
    counterparty: "コラボ先企業",
  },
  {
    datetime: "2024/3/15 01:20:00",
    brand: "LUMINA Fashion",
    type: "送金",
    description: "製造委託費用",
    amount: -175000,
    counterparty: "製造パートナー",
  },
  {
    datetime: "2024/3/15 00:10:00",
    brand: "LUMINA Fashion",
    type: "受取",
    description: "EC売上",
    amount: 98000,
    counterparty: "オンラインストア",
  },
  {
    datetime: "2024/3/14 23:30:00",
    brand: "NEXUS Street",
    type: "送金",
    description: "広告宣伝費",
    amount: -65000,
    counterparty: "広告代理店",
  },
  {
    datetime: "2024/3/14 20:25:00",
    brand: "LUMINA Fashion",
    type: "受取",
    description: "店頭売上",
    amount: 315000,
    counterparty: "百貨店",
  },
  {
    datetime: "2024/3/14 19:15:00",
    brand: "NEXUS Street",
    type: "送金",
    description: "材料仕入れ",
    amount: -42000,
    counterparty: "素材サプライヤー",
  },
  {
    datetime: "2024/3/14 18:00:00",
    brand: "LUMINA Fashion",
    type: "受取",
    description: "キャンペーン売上",
    amount: 76000,
    counterparty: "直営店",
  },
  {
    datetime: "2024/3/14 17:40:00",
    brand: "NEXUS Street",
    type: "送金",
    description: "配送費用",
    amount: -12000,
    counterparty: "配送会社",
  },
];
