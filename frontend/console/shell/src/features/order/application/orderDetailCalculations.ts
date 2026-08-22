// frontend/console/shell/src/features/order/application/orderDetailCalculations.ts 
 
import type { OrderDetailItemDTO } from "../infrastructure/repository"; 
 
export type OrderDetailListLink = { 
  id: string; 
  readableId: string; 
}; 
 
export function formatJPY(value: number): string { 
  return `¥${value.toLocaleString()}`; 
} 
 
export function calculateOrderQuantity(items: OrderDetailItemDTO[]): number { 
  return items.reduce((total, item) => total + item.qty, 0); 
} 
 
export function calculateOrderTotalPrice(items: OrderDetailItemDTO[]): number { 
  return items.reduce((total, item) => total + item.price * item.qty, 0); 
} 
 
export function hasTransferredItem(items: OrderDetailItemDTO[]): boolean { 
  return items.some((item) => item.transferred); 
} 
 
export function extractListLinks( 
  items: OrderDetailItemDTO[], 
): OrderDetailListLink[] { 
  const lists = new Map<string, OrderDetailListLink>(); 
 
  for (const item of items) { 
    if (!item.listId) { 
      continue; 
    } 
 
    lists.set(item.listId, { 
      id: item.listId, 
      readableId: item.listReadableId || item.listId, 
    }); 
  } 
 
  return Array.from(lists.values()); 
}