// frontend/console/order/src/presentation/hooks/useOrderDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  createOrderRepository,
  Order,
} from "../../infrastructure/repostiroty";
import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

import {
  buildOrderDetailFromAllowedItems,
  OrderDetailDTO,
  OrderDetailItemDTO,
} from "../../application/orderDetailBuilder";
import {
  calculateOrderQuantity,
  calculateOrderTotalPrice,
  extractListIds,
  formatJPY,
  hasTransferredItem,
} from "../../application/orderDetailCalculations";

export { formatJPY };
export type { OrderDetailDTO, OrderDetailItemDTO };

export type UseOrderDetailReturn = {
  orderId?: string;
  order: OrderDetailDTO | null;
  loading: boolean;
  error: string | null;

  items: OrderDetailItemDTO[];
  quantity: number;
  totalPrice: number;
  anyTransferred: boolean;
  createdAt: string;
  shipping: OrderDetailDTO["shippingSnapshot"];
  userName: string;
  avatarName: string;
  listIds: string[];
  pageTitle: string;

  onBack: () => void;
};

export function useOrderDetail(): UseOrderDetailReturn {
  const navigate = useNavigate();
  const { orderId } = useParams<{ orderId: string }>();

  const repo = React.useMemo(() => createOrderRepository(), []);

  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [order, setOrder] = React.useState<OrderDetailDTO | null>(null);

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

        // 2) /orders/items?id=... で item 行を取得して detail を組み立て
        const rowsRes = await repo.listItemInventoryRows({
          id,
          page: 1,
          perPage: 500,
        });

        const detail = buildOrderDetailFromAllowedItems(
          base,
          rowsRes.items ?? [],
        );

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

    void run();

    return () => {
      cancelled = true;
    };
  }, [orderId, repo]);

  // 戻るは -1 ではなく、注文一覧（本モジュールのルート絶対）へ
  const onBack = React.useCallback(() => {
    navigate("/order");
  }, [navigate]);

  const items = React.useMemo<OrderDetailItemDTO[]>(
    () => order?.items ?? [],
    [order?.items],
  );

  const quantity = React.useMemo(
    () => calculateOrderQuantity(items),
    [items],
  );

  const totalPrice = React.useMemo(
    () => calculateOrderTotalPrice(items),
    [items],
  );

  const anyTransferred = React.useMemo(
    () => hasTransferredItem(items),
    [items],
  );

  const createdAt = safeDateLabelJa(order?.createdAt, "-");

  const shipping = order?.shippingSnapshot;

  const userName = String(order?.userName ?? "").trim() || "-";
  const avatarName = String(order?.avatarName ?? "").trim() || "-";

  // リストID: 複数itemsがある場合は重複排除してカンマ区切り
  const listIds = React.useMemo(() => extractListIds(items), [items]);

  const pageTitle = `注文詳細：${order?.id ?? orderId ?? "不明ID"}`;

  return {
    orderId,
    order,
    loading,
    error,

    items,
    quantity,
    totalPrice,
    anyTransferred,
    createdAt,
    shipping,
    userName,
    avatarName,
    listIds,
    pageTitle,

    onBack,
  };
}