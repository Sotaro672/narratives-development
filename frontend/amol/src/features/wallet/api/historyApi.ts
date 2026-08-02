// frontend/amol/src/features/wallet/api/historyApi.ts

import {
  readJsonResponse,
  unwrapApiData,
} from "../../../components/utils/apiResponse";

import type {
  FetchWalletOrdersInput,
  WalletOrder,
  WalletOrderColor,
  WalletOrderItemKind,
  WalletOrderItemSnapshot,
  WalletOrderMeasurements,
  WalletOrdersPage,
} from "../types/orderTypes";

function getString(
  record: Record<string, unknown>,
  key: string,
): string {
  const value = record[key];

  return typeof value === "string"
    ? value
    : "";
}

function getOptionalString(
  record: Record<string, unknown>,
  key: string,
): string | undefined {
  const value = record[key];

  return typeof value === "string"
    ? value
    : undefined;
}

function getNumber(
  record: Record<string, unknown>,
  key: string,
): number {
  const value = record[key];

  return (
    typeof value === "number" &&
    Number.isFinite(value)
  )
    ? value
    : 0;
}

function getOptionalNumber(
  record: Record<string, unknown>,
  key: string,
): number | undefined {
  const value = record[key];

  return (
    typeof value === "number" &&
    Number.isFinite(value)
  )
    ? value
    : undefined;
}

function getBoolean(
  record: Record<string, unknown>,
  key: string,
): boolean {
  const value = record[key];

  return typeof value === "boolean"
    ? value
    : false;
}

function getOptionalBoolean(
  record: Record<string, unknown>,
  key: string,
): boolean | undefined {
  const value = record[key];

  return typeof value === "boolean"
    ? value
    : undefined;
}

function getOptionalWalletOrderItemKind(
  record: Record<string, unknown>,
  key: string,
): WalletOrderItemKind | undefined {
  const value = record[key];

  return typeof value === "string"
    ? value
    : undefined;
}

function toWalletOrderColor(
  value: unknown,
): WalletOrderColor | undefined {
  if (
    !value ||
    typeof value !== "object" ||
    Array.isArray(value)
  ) {
    return undefined;
  }

  const record =
    value as Record<string, unknown>;

  const color: WalletOrderColor = {
    name: getOptionalString(
      record,
      "name",
    ),
    hex: getOptionalString(
      record,
      "hex",
    ),
    rgb: getOptionalNumber(
      record,
      "rgb",
    ),
  };

  if (
    !color.name &&
    !color.hex &&
    typeof color.rgb !== "number"
  ) {
    return undefined;
  }

  return color;
}

function toWalletOrderMeasurements(
  value: unknown,
): WalletOrderMeasurements | undefined {
  if (
    !value ||
    typeof value !== "object" ||
    Array.isArray(value)
  ) {
    return undefined;
  }

  const record =
    value as Record<string, unknown>;

  const measurements:
    WalletOrderMeasurements = {};

  Object.entries(record).forEach(
    ([key, rawValue]) => {
      if (
        typeof rawValue === "number" &&
        Number.isFinite(rawValue)
      ) {
        measurements[key] =
          rawValue;

        return;
      }

      if (
        typeof rawValue === "string"
      ) {
        const parsed =
          Number(rawValue);

        if (
          Number.isFinite(parsed)
        ) {
          measurements[key] =
            parsed;
        }
      }
    },
  );

  return Object.keys(
    measurements,
  ).length > 0
    ? measurements
    : undefined;
}

function toWalletOrderItemSnapshot(
  value: unknown,
): WalletOrderItemSnapshot | null {
  if (
    !value ||
    typeof value !== "object" ||
    Array.isArray(value)
  ) {
    return null;
  }

  const record =
    value as Record<string, unknown>;

  return {
    modelId: getString(
      record,
      "modelId",
    ),
    inventoryId: getString(
      record,
      "inventoryId",
    ),
    listId: getString(
      record,
      "listId",
    ),

    productBlueprintId:
      getOptionalString(
        record,
        "productBlueprintId",
      ),
    tokenBlueprintId:
      getOptionalString(
        record,
        "tokenBlueprintId",
      ),

    productName:
      getOptionalString(
        record,
        "productName",
      ),

    brandId: getOptionalString(
      record,
      "brandId",
    ),
    brandName:
      getOptionalString(
        record,
        "brandName",
      ),
    brandIcon:
      getOptionalString(
        record,
        "brandIcon",
      ),

    kind:
      getOptionalWalletOrderItemKind(
        record,
        "kind",
      ),
    modelNumber:
      getOptionalString(
        record,
        "modelNumber",
      ),

    size: getOptionalString(
      record,
      "size",
    ),
    color: toWalletOrderColor(
      record.color,
    ),
    measurements:
      toWalletOrderMeasurements(
        record.measurements,
      ),

    volumeValue:
      getOptionalNumber(
        record,
        "volumeValue",
      ),
    volumeUnit:
      getOptionalString(
        record,
        "volumeUnit",
      ),

    tokenName:
      getOptionalString(
        record,
        "tokenName",
      ),
    tokenIcon:
      getOptionalString(
        record,
        "tokenIcon",
      ),

    qty: getNumber(
      record,
      "qty",
    ),
    price: getNumber(
      record,
      "price",
    ),

    isCanceled: getBoolean(
      record,
      "isCanceled",
    ),
    isDispatched: getBoolean(
      record,
      "isDispatched",
    ),

    transferred:
      getOptionalBoolean(
        record,
        "transferred",
      ),
    transferredAt:
      getOptionalString(
        record,
        "transferredAt",
      ),
  };
}

function toWalletOrder(
  value: unknown,
): WalletOrder | null {
  if (
    !value ||
    typeof value !== "object" ||
    Array.isArray(value)
  ) {
    return null;
  }

  const record =
    value as Record<string, unknown>;

  const rawItems =
    Array.isArray(record.items)
      ? record.items
      : [];

  const items = rawItems
    .map((item) =>
      toWalletOrderItemSnapshot(
        item,
      ),
    )
    .filter(
      (
        item,
      ): item is WalletOrderItemSnapshot =>
        item !== null,
    );

  return {
    id: getString(
      record,
      "id",
    ),
    userId: getString(
      record,
      "userId",
    ),
    avatarId: getString(
      record,
      "avatarId",
    ),
    cartId: getString(
      record,
      "cartId",
    ),
    paid: getOptionalBoolean(
      record,
      "paid",
    ),
    items,
    createdAt:
      getOptionalString(
        record,
        "createdAt",
      ),
    updatedAt:
      getOptionalString(
        record,
        "updatedAt",
      ),
  };
}

function toWalletOrdersPage(
  value: unknown,
): WalletOrdersPage {
  const body =
    unwrapApiData<unknown>(value);

  if (
    !body ||
    typeof body !== "object" ||
    Array.isArray(body)
  ) {
    throw new Error(
      "注文履歴APIのレスポンス形式が不正です。",
    );
  }

  const record =
    body as Record<string, unknown>;

  const rawItems =
    record.items;

  if (!Array.isArray(rawItems)) {
    throw new Error(
      "注文履歴APIのレスポンス形式が不正です。",
    );
  }

  return {
    items: rawItems
      .map((item) =>
        toWalletOrder(item),
      )
      .filter(
        (
          item,
        ): item is WalletOrder =>
          item !== null,
      ),

    totalCount:
      getOptionalNumber(
        record,
        "totalCount",
      ),
    totalPages:
      getOptionalNumber(
        record,
        "totalPages",
      ),
    page: getOptionalNumber(
      record,
      "page",
    ),
    perPage:
      getOptionalNumber(
        record,
        "perPage",
      ),
  };
}

export async function fetchWalletOrders({
  backendUrl,
  idToken,
  page = 1,
  perPage = 20,
  sort = "createdAt",
  order = "desc",
}: FetchWalletOrdersInput): Promise<WalletOrdersPage> {
  const url = new URL(
    `${backendUrl}/mall/me/orders`,
  );

  url.searchParams.set(
    "page",
    String(page),
  );
  url.searchParams.set(
    "perPage",
    String(perPage),
  );
  url.searchParams.set(
    "sort",
    sort,
  );
  url.searchParams.set(
    "order",
    order,
  );

  const response = await fetch(
    url.toString(),
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization:
          `Bearer ${idToken}`,
      },
    },
  );

  const responseBody =
    await readJsonResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "注文履歴の取得に失敗しました。",
        nonJsonErrorMessage:
          "注文履歴APIがJSON以外を返しました。",
        invalidJsonErrorMessage:
          "注文履歴APIのJSON形式が不正です。",
      },
    );

  return toWalletOrdersPage(
    responseBody,
  );
}