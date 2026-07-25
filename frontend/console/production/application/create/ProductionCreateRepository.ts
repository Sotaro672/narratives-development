//frontend\console\production\src\application\create\ProductionCreateRepository.ts
import type { Production } from "./ProductionCreateTypes";

// ======================================================================
// Port: ProductionRepository
// ======================================================================
// Application 層は I/O の詳細(HTTP等)を知らないため、Port を定義し、
// Infrastructure 側が Adapter(実装)を提供する。
export interface ProductionRepository {
  create(payload: Production): Promise<Production>;
}
