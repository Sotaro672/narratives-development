//frontend\console\shell\src\features\production\application\productionQuantityRow.ts
export type ProductionQuantityRow = {
  modelId: string;
  kind?: "apparel" | "alcohol";
  modelNumber: string;
  size?: string;
  color?: string;
  rgb?: number;
  volumeValue?: number;
  volumeUnit?: string;
  displayOrder?: number;
  quantity: number;
};