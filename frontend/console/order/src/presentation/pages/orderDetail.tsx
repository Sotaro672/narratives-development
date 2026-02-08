// frontend/order/src/pages/orderDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import { createOrderRepository } from "../../infrastructure/repostiroty";

// 日付フォーマット (YYYY/MM/DD)
const formatDate = (iso: string | null | undefined): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// 金額フォーマット
const formatJPY = (n: number | null | undefined): string => {
  const v = typeof n === "number" && Number.isFinite(n) ? n : 0;
  return `¥${v.toLocaleString()}`;
};

type OrderDetailDTO = {
  id: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  paid: boolean;
  createdAt?: string;

  shippingSnapshot?: {
    ZipCode?: string;
    State?: string;
    City?: string;
    Street?: string;
    Street2?: string;
    Country?: string;
    // もし将来フィールド追加されても壊れないように
    [k: string]: any;
  };

  billingSnapshot?: {
    Last4?: string;
    CardHolderName?: string;
    [k: string]: any;
  };

  items?: Array<{
    modelId?: string;
    inventoryId?: string;
    listId?: string;
    qty?: number;
    price?: number;
    transferred: boolean;
    transferredAt?: string;
    [k: string]: any;
  }>;
};

export default function OrderDetail() {
  const navigate = useNavigate();
  const { orderId } = useParams<{ orderId: string }>();

  const repo = React.useMemo(() => createOrderRepository(), []);

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [order, setOrder] = React.useState<OrderDetailDTO | null>(null);

  // fetch order
  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      const id = String(orderId ?? "").trim();
      if (!id) {
        setError("orderId is missing");
        return;
      }

      setLoading(true);
      setError(null);

      try {
        // repository の Order 型は「一覧用/汎用」と差が出やすいので、
        // ここではレスポンスを DTO として受ける（camelCase前提）
        const o = (await repo.getById(id)) as unknown as OrderDetailDTO;
        if (cancelled) return;
        setOrder(o);
      } catch (e) {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (cancelled) return;
        setLoading(false);
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, [orderId, repo]);

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // derived
  const items = order?.items ?? [];
  const quantity = items.reduce(
    (sum, it) => sum + (Number(it?.qty ?? 0) || 0),
    0,
  );
  const totalPrice = items.reduce(
    (sum, it) =>
      sum + (Number(it?.price ?? 0) || 0) * (Number(it?.qty ?? 0) || 0),
    0,
  );

  const anyTransferred = items.some((it) => Boolean(it?.transferred));
  const createdAt = formatDate(order?.createdAt);

  const shipping = order?.shippingSnapshot;
  const billing = order?.billingSnapshot;

  return (
    <PageStyle
      layout="single"
      title={`注文詳細：${order?.id ?? orderId ?? "不明ID"}`}
      onBack={onBack}
    >
      <Card className="mt-4">
        <CardHeader>
          <CardTitle>注文情報</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground">読み込み中...</div>
          ) : error ? (
            <div className="text-sm text-red-600 whitespace-pre-wrap">
              {error}
            </div>
          ) : !order ? (
            <div className="text-sm text-muted-foreground">
              データがありません。
            </div>
          ) : (
            <div className="space-y-8">
              {/* =======================
                  基本情報（レスポンス直表示）
                  ======================= */}
              <div>
                <div className="text-sm font-semibold mb-2">基本情報</div>
                <table className="w-full text-sm">
                  <tbody>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        注文ID
                      </th>
                      <td className="py-2">{order.id ?? "-"}</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        ユーザーID
                      </th>
                      <td className="py-2">{order.userId ?? "-"}</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        アバターID
                      </th>
                      <td className="py-2">{order.avatarId ?? "-"}</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        カートID
                      </th>
                      <td className="py-2">{order.cartId ?? "-"}</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        支払
                      </th>
                      <td className="py-2">
                        {order.paid ? (
                          <span className="order-badge is-paid">支払済</span>
                        ) : (
                          <span className="order-badge is-cancelled">未払い</span>
                        )}
                      </td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        注文日
                      </th>
                      <td className="py-2">{createdAt}</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        移譲済み（注文内いずれか）
                      </th>
                      <td className="py-2">
                        {anyTransferred ? (
                          <span className="order-badge is-transferred">
                            移譲済み
                          </span>
                        ) : (
                          <span className="order-badge is-paid">未移譲</span>
                        )}
                      </td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        アイテム数
                      </th>
                      <td className="py-2">{items.length} 点</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        数量合計
                      </th>
                      <td className="py-2">{quantity} 点</td>
                    </tr>

                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        合計金額
                      </th>
                      <td className="py-2">{formatJPY(totalPrice)}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* =======================
                  住所（shippingSnapshot）
                  ======================= */}
              <div>
                <div className="text-sm font-semibold mb-2">配送先</div>
                <table className="w-full text-sm">
                  <tbody>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        ZipCode
                      </th>
                      <td className="py-2">{shipping?.ZipCode ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        State
                      </th>
                      <td className="py-2">{shipping?.State ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        City
                      </th>
                      <td className="py-2">{shipping?.City ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        Street
                      </th>
                      <td className="py-2">{shipping?.Street ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        Street2
                      </th>
                      <td className="py-2">{shipping?.Street2 ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        Country
                      </th>
                      <td className="py-2">{shipping?.Country ?? "-"}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* =======================
                  請求情報（billingSnapshot）
                  ======================= */}
              <div>
                <div className="text-sm font-semibold mb-2">請求情報</div>
                <table className="w-full text-sm">
                  <tbody>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        Last4
                      </th>
                      <td className="py-2">{billing?.Last4 ?? "-"}</td>
                    </tr>
                    <tr>
                      <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                        CardHolderName
                      </th>
                      <td className="py-2">{billing?.CardHolderName ?? "-"}</td>
                    </tr>
                  </tbody>
                </table>
              </div>

              {/* =======================
                  items（全表示）
                  ======================= */}
              <div>
                <div className="text-sm font-semibold mb-2">アイテム</div>
                {items.length === 0 ? (
                  <div className="text-sm text-muted-foreground">
                    アイテムがありません。
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b">
                          <th className="text-left font-medium py-2 pr-4 whitespace-nowrap">
                            modelId
                          </th>
                          <th className="text-left font-medium py-2 pr-4 whitespace-nowrap">
                            inventoryId
                          </th>
                          <th className="text-left font-medium py-2 pr-4 whitespace-nowrap">
                            listId
                          </th>
                          <th className="text-right font-medium py-2 pr-4 whitespace-nowrap">
                            qty
                          </th>
                          <th className="text-right font-medium py-2 pr-4 whitespace-nowrap">
                            price
                          </th>
                          <th className="text-left font-medium py-2 pr-4 whitespace-nowrap">
                            transferred
                          </th>
                          <th className="text-left font-medium py-2 whitespace-nowrap">
                            transferredAt
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {items.map((it, idx) => {
                          const transferredAt = it.transferredAt
                            ? formatDate(it.transferredAt)
                            : "-";
                          return (
                            <tr key={idx} className="border-b">
                              <td className="py-2 pr-4">
                                {it.modelId ?? "-"}
                              </td>
                              <td className="py-2 pr-4">
                                {it.inventoryId ?? "-"}
                              </td>
                              <td className="py-2 pr-4">
                                {it.listId ?? "-"}
                              </td>
                              <td className="py-2 pr-4 text-right">
                                {Number(it.qty ?? 0) || 0}
                              </td>
                              <td className="py-2 pr-4 text-right">
                                {formatJPY(Number(it.price ?? 0) || 0)}
                              </td>
                              <td className="py-2 pr-4">
                                {it.transferred ? (
                                  <span className="order-badge is-transferred">
                                    true
                                  </span>
                                ) : (
                                  <span className="order-badge is-paid">
                                    false
                                  </span>
                                )}
                              </td>
                              <td className="py-2">{transferredAt}</td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </PageStyle>
  );
}
