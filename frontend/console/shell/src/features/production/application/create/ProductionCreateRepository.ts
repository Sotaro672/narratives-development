// frontend/console/shell/src/features/production/application/create/ProductionCreateRepository.ts

import type { Production } from "../../../../shared/types/production";

// ======================================================================
// Port: ProductionRepository
// ======================================================================
// Application 層は I/O の詳細（HTTP等）を知らないため、Portを定義し、
// Infrastructure 側がAdapter（実装）を提供する。
export interface ProductionRepository {
  create(payload: Production): Promise<Production>;
}