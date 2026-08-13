// frontend/console/shell/src/features/production/infrastructure/query/productionQuery.ts

import {
  API_BASE,
} from "../../../../shared/http/apiBase";

import {
  getAuthHeadersOrThrow,
} from "../../../../shared/http/authHeaders";

export type ProductionModelResponse = {
  ModelID: string;
  Quantity: number;
};

export type ProductionListItemResponse = {
  ID: string;
  ProductBlueprintID: string;
  AssigneeID: string;

  Models: ProductionModelResponse[];

  Printed: boolean;
  PrintedAt?: string | null;
  PrintedBy?: string | null;

  CreatedBy?: string | null;
  CreatedAt: string;

  UpdatedBy?: string | null;
  UpdatedAt?: string | null;

  totalQuantity: number;

  productName?: string;
  brandName?: string;
  assigneeName?: string;
  createdByName?: string;
  updatedByName?: string;
  printedByName?: string;
};

export async function listProductionsHTTP(): Promise<
  ProductionListItemResponse[]
> {
  const headers =
    await getAuthHeadersOrThrow();

  const response =
    await fetch(
      `${API_BASE}/productions`,
      {
        method: "GET",
        headers,
      },
    );

  if (!response.ok) {
    const detail =
      await response
        .text()
        .catch(
          () => "",
        );

    throw new Error(
      `生産計画一覧の取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  return response.json() as Promise<
    ProductionListItemResponse[]
  >;
}