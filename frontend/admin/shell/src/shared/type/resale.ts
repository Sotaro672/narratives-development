// frontend/admin/shell/src/shared/type/resale.ts

export type ResaleImage = {
  id: string;
  url: string;
  displayOrder: number;
};

export type Resale = {
  id: string;
  companyId: string;
  productBlueprintId: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  reportCaseId: string;
  status: string;
  price: number;
  condition: string;
  description: string;
  images: ResaleImage[];
  reportCount: number;
  createdAt: string;
  updatedAt?: string | null;
};

export type ResaleListResponse = {
  items: Resale[];
};