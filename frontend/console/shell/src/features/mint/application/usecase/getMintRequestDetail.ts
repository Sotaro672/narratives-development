// frontend/console/shell/src/features/mintRequest/application/usecase/getMintRequestDetail.ts 
 
import type { 
  MintRequestRepository, 
} from "../port/MintRequestRepository"; 
 
export async function getMintRequestDetail( 
  repo: MintRequestRepository, 
  productionId: string, 
) { 
  const pid = String(productionId ?? "").trim(); 
 
  if (!pid) { 
    return { 
      inspectionBatch: null, 
      mintRequestRow: null, 
      productBlueprintId: "", 
    }; 
  } 
 
  const [ 
    inspectionBatch, 
    mintRequestRow, 
  ] = await Promise.all([ 
    repo.fetchInspectionByProductionId(pid), 
    repo.fetchMintRequestRowByProductionId(pid), 
  ]); 
 
  return { 
    inspectionBatch, 
    mintRequestRow, 
    productBlueprintId: 
      inspectionBatch?.productBlueprintId ?? "", 
  }; 
}