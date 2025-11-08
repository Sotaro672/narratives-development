// frontend/order/src/pages/orderDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

export default function OrderDetail() {
  const navigate = useNavigate();
  const { orderId } = useParams<{ orderId: string }>();

  const [orderNumber] = React.useState("ORD-2024-0001");
  const [customer] = React.useState("山田 花子");
  const [brand] = React.useState("LUMINA Fashion");
  const [product] = React.useState("シルクブラウス プレミアムライン");
  const [quantity] = React.useState(2);
  const [price] = React.useState(24800);
  const [status] =
    React.useState<"出荷済み" | "処理中" | "キャンセル済み">("出荷済み");
  const [orderedAt] = React.useState("2024/10/10");
  const [shippedAt] = React.useState("2024/10/12");

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="single"
      title={`注文詳細：${orderId ?? "不明ID"}`}
      onBack={onBack}
    >
      <Card className="mt-4">
        <CardHeader>
          <CardTitle>注文情報</CardTitle>
        </CardHeader>
        <CardContent>
          <table className="w-full text-sm">
            <tbody>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  注文番号
                </th>
                <td className="py-2">{orderNumber}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  お客様
                </th>
                <td className="py-2">{customer}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  ブランド
                </th>
                <td className="py-2">{brand}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  商品
                </th>
                <td className="py-2">{product}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  数量
                </th>
                <td className="py-2">{quantity}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  合計金額
                </th>
                <td className="py-2">
                  ¥{(price * quantity).toLocaleString()}
                </td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  ステータス
                </th>
                <td className="py-2">
                  {status === "出荷済み" && (
                    <span className="order-badge is-transferred">出荷済み</span>
                  )}
                  {status === "処理中" && (
                    <span className="order-badge is-paid">処理中</span>
                  )}
                  {status === "キャンセル済み" && (
                    <span className="order-badge is-cancelled">キャンセル済み</span>
                  )}
                </td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  注文日
                </th>
                <td className="py-2">{orderedAt}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  出荷日
                </th>
                <td className="py-2">{shippedAt}</td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
    </PageStyle>
  );
}
