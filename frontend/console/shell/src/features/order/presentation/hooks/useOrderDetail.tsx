// frontend/console/shell/src/features/order/presentation/hooks/useOrderDetail.tsx      
      
import * as React from "react";      
import { useNavigate, useParams } from "react-router-dom";      
      
import {      
  createOrderRepository,      
  type OrderDetailDTO,      
  type OrderDetailItemDTO,      
} from "../../infrastructure/repository";      
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";      
import {      
  calculateOrderQuantity,      
  calculateOrderTotalPrice,      
  extractListLinks,      
  formatJPY,      
  hasTransferredItem,      
  type OrderDetailListLink,      
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
  subtotal: number;      
  shippingAmount: number;      
  consumptionTax: number;      
  totalPrice: number;      
  anyTransferred: boolean;      
  createdAt: string;      
  shipping: OrderDetailDTO["shippingSnapshot"] | undefined;      
  userName: string;      
  email: string;      
  lists: OrderDetailListLink[];      
  pageTitle: string;      
  onBack: () => void;      
  goListDetail: (listId: string) => void;      
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
      if (!orderId) {      
        setOrder(null);      
        setError("orderId is missing");      
        return;      
      }      
      
      setLoading(true);      
      setError(null);      
      
      try {      
        const detail = await repo.getById(orderId);      
      
        if (!cancelled) {      
          setOrder(detail);      
        }      
      } catch (e) {      
        if (!cancelled) {      
          setOrder(null);      
          setError(e instanceof Error ? e.message : String(e));      
        }      
      } finally {      
        if (!cancelled) {      
          setLoading(false);      
        }      
      }      
    };      
      
    void run();      
      
    return () => {      
      cancelled = true;      
    };      
  }, [orderId, repo]);      
      
  const onBack = React.useCallback(() => {      
    navigate("/order");      
  }, [navigate]);      
      
  const goListDetail = React.useCallback(      
    (listId: string) => {      
      if (!listId) return;      
      
      navigate(`/list/${encodeURIComponent(listId)}`);      
    },      
    [navigate],      
  );      
      
  const items = React.useMemo<OrderDetailItemDTO[]>(      
    () => (order ? order.items : []),      
    [order],      
  );      
      
  const quantity = React.useMemo(      
    () => calculateOrderQuantity(items),      
    [items],      
  );      
      
  const subtotal = React.useMemo(      
    () => calculateOrderTotalPrice(items),      
    [items],      
  );      
      
  const shippingAmount = order?.shippingAmount ?? 0;      
  const consumptionTax = order?.consumptionTax ?? 0;      
      
  const totalPrice = React.useMemo(      
    () => subtotal + shippingAmount + consumptionTax,      
    [subtotal, shippingAmount, consumptionTax],      
  );      
      
  const anyTransferred = React.useMemo(      
    () => hasTransferredItem(items),      
    [items],      
  );      
      
  const createdAt = safeDateTimeLabelJa(order?.createdAt, "-");      
  const shipping = order?.shippingSnapshot;      
  const userName = order?.userName || "-";      
  const email = order?.email || "-";      
      
  const lists = React.useMemo(      
    () => extractListLinks(items),      
    [items],      
  );      
      
  const pageTitle = `注文詳細：${order?.id || orderId || "不明ID"}`;      
      
  return {      
    orderId,      
    order,      
    loading,      
    error,      
    items,      
    quantity,      
    subtotal,      
    shippingAmount,      
    consumptionTax,      
    totalPrice,      
    anyTransferred,      
    createdAt,      
    shipping,      
    userName,      
    email,      
    lists,      
    pageTitle,      
    onBack,      
    goListDetail,      
  };      
}