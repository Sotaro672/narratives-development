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

import {
  createOrderRepository,
  Order,
  OrderItemInventoryRowDTO,
} from "../../infrastructure/repostiroty";
import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

// 金額フォーマット
const formatJPY = (n: number | null | undefined): string => {
  const v = typeof n === "number" && Number.isFinite(n) ? n : 0;
  return `¥${v.toLocaleString()}`;
};

type OrderDetailDTO = {
  id: string;
  userId?: string;

  // ✅ avatarId ではなく avatarName を持つ
  avatarName?: string;

  cartId?: string;
  paid: boolean;
  createdAt?: string;

  shippingSnapshot?: {
    ZipCode?: string;
    State?: string;
    City?: string;
    Street?: string;
    Street2?: string;
    // ✅ remove: Country
    [k: string]: any;
  };

  billingSnapshot?: {
    [k: string]: any;
  };

  items?: Array<{
    modelId?: string;

    // ✅ ID ではなく name
    productName?: string;
    tokenName?: string;

    // ✅ keep field name listId, but store readableId value
    listId?: string;

    qty?: number;
    price?: number;
    transferred: boolean;
    transferredAt?: string;
    [k: string]: any;
  }>;
};

// 文字列 best-effort で拾う
function pickString(obj: any, keys: string[]): string {
  if (!obj || typeof obj !== "object") return "";
  for (const k of keys) {
    const v = obj?.[k];
    if (typeof v === "string" && v.trim() !== "") return v.trim();
  }
  return "";
}

// Order(= /orders/{id}) をベースに、/orders/items の “許可された items” だけで items を作り直す
function buildDetailFromAllowedItems(
  base: Order,
  allowedRows: OrderItemInventoryRowDTO[],
): OrderDetailDTO {
  const byOrder = allowedRows.filter(
    (r) => String((r as any).orderId ?? "") === String((base as any).id ?? ""),
  );

  const items = byOrder.map((r) => ({
    modelId: (r as any).modelId ?? "",

    // ✅ ID ではなく name を採用
    productName: (r as any).productName ?? "",
    tokenName: (r as any).tokenName ?? "",

    // ✅ listId の値を readableId に置き換える（列名/フィールド名は listId のまま）
    listId: String(
      (r as any).listReadableId ??
        (r as any).listReadableID ??
        (r as any).readableId ??
        (r as any).readableID ??
        "",
    ),

    qty: typeof (r as any).qty === "number" ? (r as any).qty : Number((r as any).qty ?? 0) || 0,
    price:
      typeof (r as any).price === "number" ? (r as any).price : Number((r as any).price ?? 0) || 0,
    transferred: Boolean((r as any).transferred),
    transferredAt: (r as any).transferredAt ?? "",
  }));

  // ✅ avatarName は /orders/items の行DTOから取る（最優先）
  //    fallback: /orders/{id} が返していればそこからも拾う（将来互換）
  const avatarNameFromRows = pickString(byOrder?.[0], [
    "avatarName",
    "avatar_name",
  ]);

  const avatarNameFromBase = pickString(base as any, [
    "avatarName",
    "avatar_name",
  ]);

  const avatarName = avatarNameFromRows || avatarNameFromBase || "";

  return {
    id: (base as any).id,
    userId: (base as any).userId,

    // ✅ avatarId を avatarName に置き換え
    avatarName,

    cartId: (base as any).cartId,
    paid: Boolean((base as any).paid),
    createdAt: (base as any).createdAt,
    shippingSnapshot: (base as any).shippingSnapshot,
    billingSnapshot: (base as any).billingSnapshot,
    items,
  };
}

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
        // 1) /orders/{id} でベース情報（配送先や課金情報など）を取得
        const base = (await repo.getById(id)) as unknown as Order;

        // 2) /orders/items?id=... で “許可された item 行” だけ取得して detail を組み立て
        const rowsRes = await repo.listItemInventoryRows({
          id,
          page: 1,
          perPage: 500,
        });

        const detail = buildDetailFromAllowedItems(base, rowsRes.items ?? []);

        if (cancelled) return;
        setOrder(detail);
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
  const quantity = items.reduce((sum, it) => sum + (Number(it?.qty ?? 0) || 0), 0);
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

  // ✅ アバター名（/orders/items から拾えるようになった）
  const avatarName = String(order?.avatarName ?? "").trim() || "-";

  // ✅ readableId は right column の「関連情報」直下に表示する
  //    複数itemsがある場合は重複排除してカンマ区切り
  const readableIds = React.useMemo(() => {
    const set = new Set<string>();
    for (const it of items) {
      const v = String(it?.listId ?? "").trim();
      if (v) set.add(v);
    }
    return Array.from(set);
  }, [items]);

  const left = (
    <Card className="mt-4">
      <CardHeader>
        <CardTitle>注文情報</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="text-sm text-muted-foreground text-left">読み込み中...</div>
        ) : error ? (
          <div className="text-sm text-red-600 whitespace-pre-wrap text-left">{error}</div>
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
                </tbody>
              </table>
            </div>

            {/* =======================
                items
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
                                  productName
                                </th>
                                <td className="py-2 text-left">{it.productName ?? "-"}</td>
                              </tr>
                              <tr>
                                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                                  tokenName
                                </th>
                                <td className="py-2 text-left">{it.tokenName ?? "-"}</td>
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
                                    <span className="order-badge is-paid">{tokenLabel}</span>
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
            <div className="text-sm text-red-600 whitespace-pre-wrap text-left">{error}</div>
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

                {/* ✅ avatarId → avatarName */}
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    アバター名
                  </th>
                  <td className="py-2 text-left">{avatarName}</td>
                </tr>

                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap text-left">
                    readableId
                  </th>
                  <td className="py-2 text-left">
                    {readableIds.length > 0 ? readableIds.join(", ") : "-"}
                  </td>
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
