// frontend/console/shell/src/features/inventory/application/inventoryDetailService.tsx 
 
import { 
  getInventoryDetailRaw, 
  updateInventoryShippingAddressRaw, 
  updateInventoryTransportationRaw, 
} from "../infrastructure/inventoryApi"; 
 
import type { 
  InventoryDetailDTO, 
  InventoryDetailViewModel, 
  TransportationOption, 
} from "../../../shared/types/inventory"; 
 
function buildInventoryDetailViewModel( 
  detail: InventoryDetailDTO, 
): InventoryDetailViewModel { 
  const inventoryId = detail.inventoryId; 
  const productBlueprintId = detail.productBlueprintId; 
  const tokenBlueprintId = detail.tokenBlueprintId; 
  const productBlueprintPatch = detail.productBlueprintPatch; 
  const tokenBlueprintPatch = detail.tokenBlueprintPatch; 
 
  if (!inventoryId) { 
    throw new Error("inventory_detail_missing_inventory_id"); 
  } 
 
  if (!productBlueprintId || !tokenBlueprintId) { 
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id"); 
  } 
 
  const productName = productBlueprintPatch.productName; 
  const tokenName = tokenBlueprintPatch.tokenName; 
 
  if (!productName) { 
    throw new Error("inventory_detail_missing_product_name"); 
  } 
 
  if (!tokenName) { 
    throw new Error("inventory_detail_missing_token_name"); 
  } 
 
  const transportationOption = 
    detail.transportationOption ?? ""; 
 
  const transportationId = 
    transportationOption === "custom" 
      ? detail.transportationId ?? "" 
      : ""; 
 
  return { 
    inventoryId, 
    productBlueprintId, 
    tokenBlueprintId, 
    productName, 
    tokenName, 
    headerTitle: `${productName} / ${tokenName}`, 
    categoryFields: productBlueprintPatch.categoryFields, 
    productBlueprintPatch, 
    tokenBlueprintPatch, 
 
    shippingAddressId: detail.shippingAddressId ?? "", 
    shippingAddress: detail.shippingAddress ?? null, 
    shippingAddressOptions: detail.shippingAddressOptions ?? [], 
 
    transportationOption, 
    transportationId, 
    transportationOptions: detail.transportationOptions ?? [], 
 
    updatedAt: detail.updatedAt, 
    totalStock: detail.totalStock, 
    rows: detail.rows, 
  }; 
} 
 
export async function loadInventoryDetailViewModel( 
  inventoryId: string, 
): Promise<InventoryDetailViewModel> { 
  const id = inventoryId; 
 
  if (!id) { 
    throw new Error("inventoryId is empty"); 
  } 
 
  const detail = await getInventoryDetailRaw(id); 
 
  if (!detail.inventoryId) { 
    throw new Error("inventory_detail_missing_inventory_id"); 
  } 
 
  if (!detail.productBlueprintId || !detail.tokenBlueprintId) { 
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id"); 
  } 
 
  return buildInventoryDetailViewModel(detail); 
} 
 
export async function saveInventoryShippingAddress( 
  inventoryId: string, 
  shippingAddressId: string, 
): Promise<InventoryDetailViewModel> { 
  const id = inventoryId; 
  const addressId = shippingAddressId; 
 
  if (!id) { 
    throw new Error("inventoryId is empty"); 
  } 
 
  if (!addressId) { 
    throw new Error("shippingAddressId is empty"); 
  } 
 
  const detail = await updateInventoryShippingAddressRaw(id, addressId); 
 
  if (!detail.inventoryId) { 
    throw new Error("inventory_detail_missing_inventory_id"); 
  } 
 
  if (!detail.productBlueprintId || !detail.tokenBlueprintId) { 
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id"); 
  } 
 
  return buildInventoryDetailViewModel(detail); 
} 
 
export async function saveInventoryTransportation( 
  inventoryId: string, 
  transportationOption: TransportationOption, 
  transportationId: string, 
): Promise<InventoryDetailViewModel> { 
  const id = inventoryId; 
  const option = transportationOption; 
 
  const settingId = 
    option === "custom" 
      ? transportationId 
      : ""; 
 
  if (!id) { 
    throw new Error("inventoryId is empty"); 
  } 
 
  if (!option) { 
    throw new Error("transportationOption is empty"); 
  } 
 
  if ( 
    option === "custom" && 
    !settingId 
  ) { 
    throw new Error("transportationId is required for custom transportation"); 
  } 
 
  const detail = await updateInventoryTransportationRaw( 
    id, 
    option, 
    settingId, 
  ); 
 
  if (!detail.inventoryId) { 
    throw new Error("inventory_detail_missing_inventory_id"); 
  } 
 
  if (!detail.productBlueprintId || !detail.tokenBlueprintId) { 
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id"); 
  } 
 
  return buildInventoryDetailViewModel(detail); 
}