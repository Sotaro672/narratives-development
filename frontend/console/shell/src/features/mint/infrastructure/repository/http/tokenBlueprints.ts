// frontend/console/shell/src/features/mintRequest/infrastructure/repository/http/tokenBlueprints.ts 
 
import { API_BASE } from "../../../../../shared/http/apiBase"; 
import { getAuthHeadersOrThrow } from "../../../../../shared/http/authHeaders"; 
 
import type { 
  TokenBlueprintSummary, 
} from "../../../application/port/MintRequestRepository"; 
 
export async function fetchTokenBlueprintsByBrandHTTP( 
  brandId: string, 
): Promise<TokenBlueprintSummary[]> { 
  const normalizedBrandId = 
    String(brandId ?? "").trim(); 
 
  if (!normalizedBrandId) { 
    return []; 
  } 
 
  const authHeaders = 
    await getAuthHeadersOrThrow(); 
 
  const url = 
    `${API_BASE}/mint/token_blueprints` + 
    `?brandId=${encodeURIComponent( 
      normalizedBrandId, 
    )}`; 
 
  const response = await fetch( 
    url, 
    { 
      method: "GET", 
      headers: authHeaders, 
    }, 
  ); 
 
  if (!response.ok) { 
    const body = 
      await response.text(); 
 
    throw new Error( 
      `Failed to fetch tokenBlueprints (mint): ` + 
        `${response.status} ${response.statusText}` + 
        ( 
          body 
            ? ` body=${body.slice(0, 400)}` 
            : "" 
        ), 
    ); 
  } 
 
  return await response.json() as 
    TokenBlueprintSummary[]; 
}