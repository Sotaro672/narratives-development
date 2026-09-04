// frontend/amol/src/features/payment/utils/order.ts 
 
import type { CartDisplayItem } from "../../shared/types/cart"; 
import type { 
  CreateOrderItemRequest, 
} from "../../shared/types/payment"; 
import type { CardPaymentMethod } from "../../shared/types/paymentMethods"; 
 
export function selectPrimaryPaymentMethod( 
  methods: CardPaymentMethod[], 
  defaultMethod: CardPaymentMethod | null, 
): CardPaymentMethod | null { 
  if (defaultMethod) { 
    return defaultMethod; 
  } 
 
  return methods.find((method) => method.isDefault) ?? methods[0] ?? null; 
} 
 
export function buildOrderItems( 
  cartItems: CartDisplayItem[], 
): CreateOrderItemRequest[] { 
  return cartItems.map((item): CreateOrderItemRequest => { 
    if (item.type === "resale") { 
      return { 
        type: "resale", 
        resaleId: item.resaleId ?? "", 
        qty: 1, 
        isCancelled: false, 
        isDispatched: false, 
      }; 
    } 
 
    return { 
      type: "list", 
      listId: item.listId ?? "", 
      modelId: item.modelId ?? "", 
      qty: item.qty, 
      isCancelled: false, 
      isDispatched: false, 
    }; 
  }); 
} 
 
export function validateOrderItems( 
  items: CreateOrderItemRequest[], 
): string | null { 
  if (items.length === 0) { 
    return "注文対象の商品がありません。"; 
  } 
 
  for (const item of items) { 
    if (item.type === "resale") { 
      const error = validateResaleOrderItem(item); 
 
      if (error) { 
        return error; 
      } 
 
      continue; 
    } 
 
    const error = validateListOrderItem(item); 
 
    if (error) { 
      return error; 
    } 
  } 
 
  return null; 
} 
 
function validateListOrderItem( 
  item: Extract<CreateOrderItemRequest, { type: "list" }>, 
): string | null { 
  if (!item.listId) { 
    return "注文商品の listId を取得できませんでした。"; 
  } 
 
  if (!item.modelId) { 
    return "注文商品の modelId を取得できませんでした。"; 
  } 
 
  if (item.qty <= 0) { 
    return "注文商品の数量が不正です。"; 
  } 
 
  return null; 
} 
 
function validateResaleOrderItem( 
  item: Extract<CreateOrderItemRequest, { type: "resale" }>, 
): string | null { 
  if (!item.resaleId) { 
    return "リセール商品の resaleId を取得できませんでした。"; 
  } 
 
  if (item.qty !== 1) { 
    return "リセール商品の数量が不正です。"; 
  } 
 
  return null; 
} 