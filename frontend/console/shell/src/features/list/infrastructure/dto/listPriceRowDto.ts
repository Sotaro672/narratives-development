// frontend/console/shell/src/features/list/infrastructure/dto/listPriceRowDto.ts

export type ListDetailPriceRowDTO = {
  modelId: string;
  kind: string;
  modelNumber: string;
  displayOrder?: number | null;
  stock: number;
  size?: string;
  color?: string;
  rgb?: number | null;
  volumeValue?: number | null;
  volumeUnit?: string;
  price?: number | null;
};