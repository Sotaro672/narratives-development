//frontend\console\list\src\infrastructure\dto\listPriceRowDto.ts
export type ListDetailPriceRowDTO = {
  modelId: string;
  displayOrder?: number | null;
  stock: number;
  size: string;
  color: string;
  rgb?: number | null;
  price: number | null;
};