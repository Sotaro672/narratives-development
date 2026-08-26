// frontend/console/shell/src/features/order/presentation/hooks/useOrderDetail.tsx 
 
import * as React from "react"; 
import { useNavigate, useParams } from "react-router-dom"; 
 
import { 
  listInquiriesHTTP, 
} from "../../../inquiry/infrastructure/inquiryRepositoryHTTP"; 
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
 
const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current"; 
 
export type UseOrderDetailReturn = { 
  orderId?: string; 
  order: OrderDetailDTO | null; 
  loading: boolean; 
  error: string | null; 
  dispatching: boolean; 
  dispatchError: string | null; 
  canDispatch: boolean; 
  returnInquiryId: string | null; 
  hasReturnInProgress: boolean; 
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
  goReturnInquiryDetail: () => void; 
  onDispatch: () => Promise<void>; 
}; 
 
export function useOrderDetail(): UseOrderDetailReturn { 
  const navigate = useNavigate(); 
  const { orderId } = useParams<{ orderId: string }>(); 
  const repo = React.useMemo(() => createOrderRepository(), []); 
 
  const [loading, setLoading] = React.useState(false); 
  const [error, setError] = React.useState<string | null>(null); 
  const [order, setOrder] = React.useState<OrderDetailDTO | null>(null); 
  const [dispatching, setDispatching] = React.useState(false); 
  const [dispatchError, setDispatchError] = React.useState<string | null>(null); 
  const [returnInquiryId, setReturnInquiryId] = React.useState<string | null>(null); 
 
  React.useEffect(() => { 
    let cancelled = false; 
 
    const run = async () => { 
      if (!orderId) { 
        setOrder(null); 
        setReturnInquiryId(null); 
        setError("orderId is missing"); 
        return; 
      } 
 
      setLoading(true); 
      setError(null); 
      setDispatchError(null); 
      setReturnInquiryId(null); 
 
      try { 
        const detail = await repo.getById(orderId); 
 
        if (cancelled) { 
          return; 
        } 
 
        setOrder(detail); 
 
        try { 
          const inquiryResult = await listInquiriesHTTP({ 
            companyId: CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER, 
            orderId, 
            status: "open", 
          }); 
 
          if (cancelled) { 
            return; 
          } 
 
          const returnInquiry = inquiryResult.items.find( 
            (item) => 
              item.inquiry.orderId === orderId && 
              ( 
                item.inquiry.inquiryType === "return_unopened" || 
                item.inquiry.inquiryType === "return_opened" 
              ), 
          ); 
 
          setReturnInquiryId( 
            returnInquiry?.inquiry.id ?? null, 
          ); 
        } catch { 
          if (!cancelled) { 
            setReturnInquiryId(null); 
          } 
        } 
      } catch (e) { 
        if (!cancelled) { 
          setOrder(null); 
          setReturnInquiryId(null); 
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
 
  const goReturnInquiryDetail = React.useCallback( 
    () => { 
      if (!returnInquiryId) { 
        return; 
      } 
 
      navigate( 
        `/inquiry/${encodeURIComponent(returnInquiryId)}`, 
      ); 
    }, 
    [navigate, returnInquiryId], 
  ); 
 
  const items = React.useMemo<OrderDetailItemDTO[]>( 
    () => (order ? order.items : []), 
    [order], 
  ); 
 
  const canDispatch = React.useMemo( 
    () => 
      items.some( 
        (item) => 
          !item.isCancelled && 
          !item.isDispatched, 
      ), 
    [items], 
  ); 
 
  const hasReturnInProgress = React.useMemo( 
    () => Boolean(returnInquiryId), 
    [returnInquiryId], 
  ); 
 
  const onDispatch = React.useCallback(async () => { 
    if (!orderId || dispatching || !canDispatch) { 
      return; 
    } 
 
    try { 
      setDispatching(true); 
      setDispatchError(null); 
 
      const updated = await repo.dispatch(orderId); 
 
      setOrder(updated); 
 
      window.dispatchEvent( 
        new Event("order:dispatch-state-changed"), 
      ); 
    } catch (e) { 
      setDispatchError( 
        e instanceof Error 
          ? e.message 
          : String(e), 
      ); 
    } finally { 
      setDispatching(false); 
    } 
  }, [ 
    orderId, 
    dispatching, 
    canDispatch, 
    repo, 
  ]); 
 
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
    dispatching, 
    dispatchError, 
    canDispatch, 
    returnInquiryId, 
    hasReturnInProgress, 
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
    goReturnInquiryDetail, 
    onDispatch, 
  }; 
}