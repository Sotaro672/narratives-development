// frontend/order/src/pages/orderDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageHeader from "../../../shell/src/layout/PageHeader/PageHeader";

export default function OrderDetail() {
  const navigate = useNavigate();
  const { orderId } = useParams<{ orderId: string }>();

  // ─────────────────────────────────────────
  // モックデータ
  // ─────────────────────────────────────────
  const [orderNumber] = React.useState("ORD-2024-0001");
  const [customer] = React.useState("山田 花子");
  const [brand] = React.useState("LUMINA Fashion");
  const [product] = React.useState("シルクブラウス プレミアムライン");
  const [quantity] = React.useState(2);
  const [price] = React.useState(24800);
  const [status] = React.useState<"出荷済み" | "処理中" | "キャンセル済み">("出荷済み");
  const [orderedAt] = React.useState("2024/10/10");
  const [shippedAt] = React.useState("2024/10/12");

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <div className="p-6">
      <PageHeader title={`注文詳細：${orderId ?? "不明ID"}`} onBack={onBack} />
    </div>
  );
}
