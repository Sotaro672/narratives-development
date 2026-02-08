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
import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

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
    [k: string]: any;
  };

  billingSnapshot?: {
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

  // ✅ 戻るは -1 ではなく、注文一覧（本モジュールのルート絶対）へ
  const onBack = React.useCallback(() => {
    navigate("/order");
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
  const createdAt = safeDateLabelJa(order?.createdAt, "-");

  const shipping = order?.shippingSnapshot;

  // right column
  const userId = order?.userId ?? "-";
  const avatarId = order?.avatarId ?? "-";

  const left = (
    <Card className="mt-4">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-sm text-muted-foreground text-left">読み込み中...</div>
        ) : error ? (
          <div className="text-sm text-red-600 whitespace-pre-wrap text-left">
            {error}
          </div>
        ) : !order ? (
          <div className="text-sm text-muted-foreground text-left">データがありません。</div>
        ) : (
          <div className="space-y-8 text-left">
            {/* =======================
                基本情報（注文ID/カートIDは削除済）
                ======================= */}
            <div>
              <div className="text-sm font-semibold mb-2 text-left">基本情報</div>
              <table className="w-full text-sm text-left">
                <tbody>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      支払
                    </th>
                    <td className="py-2 text-left">
                      {order.paid ? (
                        <span className="order-badge is-paid">支払済</span>
                      ) : (
                        <span className="order-badge is-cancelled">未払い</span>
                      )}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      注文日
                    </th>
                    <td className="py-2 text-left">{createdAt}</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      トークン
                    </th>
                    <td className="py-2 text-left">
                      {anyTransferred ? (
                        <span className="order-badge is-transferred">移譲済み</span>
                      ) : (
                        <span className="order-badge is-paid">未移譲</span>
                      )}
                    </td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      アイテム数
                    </th>
                    <td className="py-2 text-left">{items.length} 点</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      数量合計
                    </th>
                    <td className="py-2 text-left">{quantity} 点</td>
                  </tr>

                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      合計金額
                    </th>
                    <td className="py-2 text-left">{formatJPY(totalPrice)}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* =======================
                配送先（日本語ラベル）
                ======================= */}
            <div>
              <div className="text-sm font-semibold mb-2 text-left">配送先</div>
              <table className="w-full text-sm text-left">
                <tbody>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      郵便番号
                    </th>
                    <td className="py-2 text-left">{shipping?.ZipCode ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      都道府県
                    </th>
                    <td className="py-2 text-left">{shipping?.State ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      市区町村
                    </th>
                    <td className="py-2 text-left">{shipping?.City ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所1
                    </th>
                    <td className="py-2 text-left">{shipping?.Street ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      住所2
                    </th>
                    <td className="py-2 text-left">{shipping?.Street2 ?? "-"}</td>
                  </tr>
                  <tr>
                    <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                      国
                    </th>
                    <td className="py-2 text-left">{shipping?.Country ?? "-"}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            {/* =======================
                items（縦1行：1アイテム=縦に項目改行）
                ======================= */}
            <div>
              <div className="text-sm font-semibold mb-2 text-left">アイテム</div>

              {items.length === 0 ? (
                <div className="text-sm text-muted-foreground text-left">
                  アイテムがありません。
                </div>
              ) : (
                <div className="space-y-4">
                  {items.map((it, idx) => {
                    const transferredAt = safeDateLabelJa(it.transferredAt, "-");

                    const qty = Number(it.qty ?? 0) || 0;
                    const price = Number(it.price ?? 0) || 0;

                    const tokenLabel = it.transferred ? "移譲済" : "未移譲";

                    return (
                      <Card key={idx}>
                        <CardHeader className="py-3">
                          <CardTitle className="text-base text-left">
                            アイテム {idx + 1}
                          </CardTitle>
                        </CardHeader>
                        <CardContent className="pt-0">
                          <table className="w-full text-sm text-left">
                            <tbody>
                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  modelId
                                </th>
                                <td className="py-2 text-left">{it.modelId ?? "-"}</td>
                              </tr>
                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  inventoryId
                                </th>
                                <td className="py-2 text-left">
                                  {it.inventoryId ?? "-"}
                                </td>
                              </tr>
                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  listId
                                </th>
                                <td className="py-2 text-left">{it.listId ?? "-"}</td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  数量
                                </th>
                                <td className="py-2 text-left">{qty}</td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  金額
                                </th>
                                <td className="py-2 text-left">{formatJPY(price)}</td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  トークン
                                </th>
                                <td className="py-2 text-left">
                                  {it.transferred ? (
                                    <span className="order-badge is-transferred">
                                      {tokenLabel}
                                    </span>
                                  ) : (
                                    <span className="order-badge is-paid">
                                      {tokenLabel}
                                    </span>
                                  )}
                                </td>
                              </tr>

                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  移譲日
                                </th>
                                <td className="py-2 text-left">{transferredAt}</td>
                              </tr>
                            </tbody>
                          </table>
                        </CardContent>
                      </Card>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );

  const right = (
    <div className="mt-4 space-y-4 text-left">
      <Card>
        <CardHeader>
          <CardTitle className="text-left">関連情報</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-sm text-muted-foreground text-left">読み込み中...</div>
          ) : error ? (
            <div className="text-sm text-red-600 whitespace-pre-wrap text-left">
              {error}
            </div>
          ) : !order ? (
            <div className="text-sm text-muted-foreground text-left">-</div>
          ) : (
            <table className="w-full text-sm text-left">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    ユーザーID
                  </th>
                  <td className="py-2 text-left">{userId}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    アバターID
                  </th>
                  <td className="py-2 text-left">{avatarId}</td>
                </tr>
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title={`注文詳細：${order?.id ?? orderId ?? "不明ID"}`}
      onBack={onBack}
    >
      {[left, right]}
    </PageStyle>
  );
}
