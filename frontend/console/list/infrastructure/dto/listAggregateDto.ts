//frontend\console\list\src\infrastructure\dto\listAggregateDto.ts
import type { ListDTO } from "./listDto";

export type ListAggregateDTO = {
  items: ListDTO[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};