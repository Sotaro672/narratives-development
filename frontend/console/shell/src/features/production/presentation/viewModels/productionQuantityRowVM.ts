// frontend/console/shell/src/features/production/presentation/viewModels/productionQuantityRowVM.ts

export type ProductionQuantityRowVM = {
  modelId: string;
  kind?: "apparel" | "alcohol" | string;

  modelNumber: string;
  variationLabel?: string;

  // apparel
  size?: string;
  color?: string;
  rgb?: number;

  // alcohol
  volumeValue?: number;
  volumeUnit?: string;

  displayOrder?: number;
  quantity: number;
};