// frontend/order/mockdata.tsx

export type OrderRow = {
  id: string;
  customerName: string;
  productName: string;
  additionalItems: string;
  status: "支払済" | "移譲完了";
  paymentMethod: string;
  amount: string; // e.g. "¥32,700"
  quantityInfo: string; // e.g. "3点"
  purchaseLocation: string;
  orderDate: string; // YYYY/M/D
};

export const ORDERS: OrderRow[] = [
  {
    id: "ORD-2024-0002",
    customerName: "山本 由紀",
    productName: "デニムジャケット ヴィンテージ加工",
    additionalItems: "他1点",
    status: "支払済",
    paymentMethod: "デジタルウォレット",
    amount: "¥32,700",
    quantityInfo: "3点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/21",
  },
  {
    id: "ORD-2024-0001",
    customerName: "Creator Alice",
    productName: "シルクブラウス プレミアムライン",
    additionalItems: "他1点",
    status: "移譲完了",
    paymentMethod: "クレジットカード",
    amount: "¥45,900",
    quantityInfo: "2点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/20",
  },
];
