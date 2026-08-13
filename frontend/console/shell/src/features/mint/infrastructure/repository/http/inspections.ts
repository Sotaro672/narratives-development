// frontend/console/shell/src/features/mint/infrastructure/repository/http/inspections.ts 
 
import { API_BASE } from "../../../../../shared/http/apiBase"; 
import { getAuthHeadersOrThrow } from "../../../../../shared/http/authHeaders"; 
 
import type { InspectionBatchDTO } from "../../../../../shared/types/inspections"; 
import type { MintRequestDetailDTO } from "../../dto/mintRequestLocal.dto"; 
 
// =============================== 
// private: detail fetch (/mint/inspections/{productionId}) 
// - public API からは export しない 
// =============================== 
 
async function fetchMintRequestDetailByProductionIdHTTP( 
  productionId: string, 
): Promise<MintRequestDetailDTO | null> { 
  const pid = String(productionId ?? "").trim(); 
 
  if (!pid) { 
    throw new Error("productionId が空です"); 
  } 
 
  const authHeaders = await getAuthHeadersOrThrow(); 
 
  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(pid)}`; 
 
  const res = await fetch(url, { 
    method: "GET", 
    headers: authHeaders, 
  }); 
 
  if (res.status === 404) { 
    return null; 
  } 
 
  if (!res.ok) { 
    const body = await res.text(); 
 
    throw new Error( 
      `Failed to fetch mint request detail: ${res.status} ${res.statusText}${ 
        body ? ` body=${body.slice(0, 400)}` : "" 
      }`, 
    ); 
  } 
 
  return (await res.json()) as MintRequestDetailDTO; 
} 
 
// =============================== 
// single: inspection by productionId 
// =============================== 
 
export async function fetchInspectionByProductionIdHTTP( 
  productionId: string, 
): Promise<InspectionBatchDTO | null> { 
  const pid = String(productionId ?? "").trim(); 
 
  if (!pid) { 
    throw new Error("productionId が空です"); 
  } 
 
  const detail = 
    await fetchMintRequestDetailByProductionIdHTTP( 
      pid, 
    ); 
 
  if (!detail?.inspection) { 
    return null; 
  } 
 
  return { 
    ...detail.inspection, 
    productBlueprintId: 
      detail.productBlueprintId ?? "", 
    productName: 
      detail.productName, 
    modelMeta: 
      detail.modelMeta ?? {}, 
  }; 
} 
 
// =============================== 
// complete: /products/inspections/complete 
// =============================== 
 
export async function completeInspectionHTTP( 
  productionId: string, 
): Promise<InspectionBatchDTO | null> { 
  const pid = String(productionId ?? "").trim(); 
 
  if (!pid) { 
    throw new Error("productionId が空です"); 
  } 
 
  const authHeaders = await getAuthHeadersOrThrow(); 
 
  const url = `${API_BASE}/products/inspections/complete`; 
 
  const headers = { 
    ...authHeaders, 
    "Content-Type": "application/json", 
  }; 
 
  const res = await fetch(url, { 
    method: "PATCH", 
    headers, 
    body: JSON.stringify({ 
      productionId: pid, 
    }), 
  }); 
 
  if (!res.ok) { 
    const body = await res.text(); 
 
    throw new Error( 
      `Failed to complete inspection: ${res.status} ${res.statusText}${ 
        body ? ` body=${body.slice(0, 400)}` : "" 
      }`, 
    ); 
  } 
 
  return (await res.json()) as InspectionBatchDTO; 
}