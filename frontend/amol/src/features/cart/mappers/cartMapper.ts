// frontend/amol/src/features/cart/mappers/cartMapper.ts

import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";

import type {
  CartDisplayItem,
  CartDTO,
  CartItemDTO,
  CartItemType,
  CartModelKind,
} from "../types/cart";

function asNonEmptyString(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

function asOptionalNumber(
  value: unknown,
): number | undefined {
  if (!isFiniteNumber(value)) {
    return undefined;
  }

  return value;
}

function asNullableString(
  value: unknown,
): string | null {
  if (value === null) {
    return null;
  }

  const normalizedValue =
    asNonEmptyString(value);

  return normalizedValue || null;
}

function normalizeCartItemType(
  value: unknown,
): CartItemType | null {
  if (
    value === "list" ||
    value === "resale"
  ) {
    return value;
  }

  return null;
}

function normalizeCartModelKind(
  value: unknown,
): CartModelKind | undefined {
  if (
    value === "apparel" ||
    value === "alcohol" ||
    value === "unknown"
  ) {
    return value;
  }

  return undefined;
}

function normalizeQty(
  value: unknown,
): number {
  if (
    !isFiniteNumber(value) ||
    value <= 0
  ) {
    return 1;
  }

  return Math.floor(value);
}

function normalizePrice(
  value: unknown,
): number | undefined {
  if (
    !isFiniteNumber(value) ||
    value < 0
  ) {
    return undefined;
  }

  return Math.floor(value);
}

function unwrapCartResponse(
  raw: unknown,
): Record<string, unknown> {
  if (
    !isRecord(raw) ||
    Array.isArray(raw)
  ) {
    return {};
  }

  const data = raw.data;

  if (
    isRecord(data) &&
    !Array.isArray(data)
  ) {
    return data;
  }

  return raw;
}

function normalizeCartItem(
  raw: unknown,
): CartItemDTO | null {
  if (
    !isRecord(raw) ||
    Array.isArray(raw)
  ) {
    return null;
  }

  const type =
    normalizeCartItemType(
      raw.type,
    );

  if (!type) {
    return null;
  }

  const inventoryId =
    asNonEmptyString(
      raw.inventoryId,
    );

  const listId =
    asNonEmptyString(
      raw.listId,
    );

  const modelId =
    asNonEmptyString(
      raw.modelId,
    );

  const resaleId =
    asNonEmptyString(
      raw.resaleId,
    );

  const productId =
    asNonEmptyString(
      raw.productId,
    );

  const productBlueprintId =
    asNonEmptyString(
      raw.productBlueprintId,
    );

  const tokenBlueprintId =
    asNonEmptyString(
      raw.tokenBlueprintId,
    );

  const brandId =
    asNonEmptyString(
      raw.brandId,
    );

  const title =
    asNonEmptyString(
      raw.title,
    );

  const listImage =
    asNonEmptyString(
      raw.listImage,
    );

  const imageUrl =
    asNonEmptyString(
      raw.imageUrl,
    );

  const brandName =
    asNonEmptyString(
      raw.brandName,
    );

  const productName =
    asNonEmptyString(
      raw.productName,
    );

  const modelNumber =
    asNonEmptyString(
      raw.modelNumber,
    );

  const modelLabel =
    asNonEmptyString(
      raw.modelLabel,
    );

  const size =
    asNonEmptyString(
      raw.size,
    );

  const color =
    asNonEmptyString(
      raw.color,
    );

  const colorName =
    asNonEmptyString(
      raw.colorName,
    );

  const volumeUnit =
    asNonEmptyString(
      raw.volumeUnit,
    );

  const modelKind =
    normalizeCartModelKind(
      raw.modelKind,
    );

  const price =
    normalizePrice(
      raw.price,
    );

  const colorRGB =
    asOptionalNumber(
      raw.colorRGB,
    );

  const volumeValue =
    asOptionalNumber(
      raw.volumeValue,
    );

  return {
    type,

    qty:
      normalizeQty(
        raw.qty,
      ),

    ...(inventoryId
      ? {
          inventoryId,
        }
      : {}),

    ...(listId
      ? {
          listId,
        }
      : {}),

    ...(modelId
      ? {
          modelId,
        }
      : {}),

    ...(resaleId
      ? {
          resaleId,
        }
      : {}),

    ...(productId
      ? {
          productId,
        }
      : {}),

    ...(productBlueprintId
      ? {
          productBlueprintId,
        }
      : {}),

    ...(tokenBlueprintId
      ? {
          tokenBlueprintId,
        }
      : {}),

    ...(brandId
      ? {
          brandId,
        }
      : {}),

    ...(title
      ? {
          title,
        }
      : {}),

    ...(listImage
      ? {
          listImage,
        }
      : {}),

    ...(imageUrl
      ? {
          imageUrl,
        }
      : {}),

    ...(price !== undefined
      ? {
          price,
        }
      : {}),

    ...(brandName
      ? {
          brandName,
        }
      : {}),

    ...(productName
      ? {
          productName,
        }
      : {}),

    ...(modelKind
      ? {
          modelKind,
        }
      : {}),

    ...(modelNumber
      ? {
          modelNumber,
        }
      : {}),

    ...(modelLabel
      ? {
          modelLabel,
        }
      : {}),

    ...(size
      ? {
          size,
        }
      : {}),

    ...(color
      ? {
          color,
        }
      : {}),

    ...(colorName
      ? {
          colorName,
        }
      : {}),

    ...(colorRGB !== undefined
      ? {
          colorRGB,
        }
      : {}),

    ...(volumeValue !== undefined
      ? {
          volumeValue,
        }
      : {}),

    ...(volumeUnit
      ? {
          volumeUnit,
        }
      : {}),
  };
}

function normalizeCartItems(
  value: unknown,
): Record<string, CartItemDTO> {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return {};
  }

  return Object.entries(value).reduce<
    Record<string, CartItemDTO>
  >(
    (
      normalizedItems,
      [
        rawItemKey,
        rawItem,
      ],
    ) => {
      const itemKey =
        rawItemKey.trim();

      if (!itemKey) {
        return normalizedItems;
      }

      const item =
        normalizeCartItem(
          rawItem,
        );

      if (!item) {
        return normalizedItems;
      }

      normalizedItems[itemKey] =
        item;

      return normalizedItems;
    },
    {},
  );
}

export function cartResponseToDTO(
  raw: unknown,
): CartDTO {
  const data =
    unwrapCartResponse(raw);

  return {
    avatarId:
      asNonEmptyString(
        data.avatarId,
      ),

    items:
      normalizeCartItems(
        data.items,
      ),

    createdAt:
      asNullableString(
        data.createdAt,
      ),

    updatedAt:
      asNullableString(
        data.updatedAt,
      ),

    expiresAt:
      asNullableString(
        data.expiresAt,
      ),
  };
}

function buildDisplayFields(
  item: CartItemDTO,
): Partial<CartDisplayItem> {
  return {
    ...(item.productBlueprintId
      ? {
          productBlueprintId:
            item.productBlueprintId,
        }
      : {}),

    ...(item.tokenBlueprintId
      ? {
          tokenBlueprintId:
            item.tokenBlueprintId,
        }
      : {}),

    ...(item.brandId
      ? {
          brandId:
            item.brandId,
        }
      : {}),

    ...(item.title
      ? {
          title:
            item.title,
        }
      : {}),

    ...(item.listImage
      ? {
          listImage:
            item.listImage,
        }
      : {}),

    ...(item.imageUrl
      ? {
          imageUrl:
            item.imageUrl,
        }
      : {}),

    ...(item.price !== undefined
      ? {
          price:
            item.price,
        }
      : {}),

    ...(item.brandName
      ? {
          brandName:
            item.brandName,
        }
      : {}),

    ...(item.productName
      ? {
          productName:
            item.productName,
        }
      : {}),

    ...(item.modelKind
      ? {
          modelKind:
            item.modelKind,
        }
      : {}),

    ...(item.modelNumber
      ? {
          modelNumber:
            item.modelNumber,
        }
      : {}),

    ...(item.modelLabel
      ? {
          modelLabel:
            item.modelLabel,
        }
      : {}),

    ...(item.size
      ? {
          size:
            item.size,
        }
      : {}),

    ...(item.color
      ? {
          color:
            item.color,
        }
      : {}),

    ...(item.colorName
      ? {
          colorName:
            item.colorName,
        }
      : {}),

    ...(item.colorRGB !== undefined
      ? {
          colorRGB:
            item.colorRGB,
        }
      : {}),

    ...(item.volumeValue !== undefined
      ? {
          volumeValue:
            item.volumeValue,
        }
      : {}),

    ...(item.volumeUnit
      ? {
          volumeUnit:
            item.volumeUnit,
        }
      : {}),
  };
}

function listCartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  const {
    avatarId,
    itemKey,
    item,
  } = args;

  const inventoryId =
    asNonEmptyString(
      item.inventoryId,
    );

  const listId =
    asNonEmptyString(
      item.listId,
    );

  const modelId =
    asNonEmptyString(
      item.modelId,
    );

  if (
    !inventoryId ||
    !listId ||
    !modelId
  ) {
    return null;
  }

  return {
    itemKey,
    avatarId,

    type: "list",

    inventoryId,
    listId,
    modelId,

    ...(item.productId
      ? {
          productId:
            item.productId,
        }
      : {}),

    qty:
      normalizeQty(
        item.qty,
      ),

    ...buildDisplayFields(
      item,
    ),

    catalog: null,
  };
}

function resaleCartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  const {
    avatarId,
    itemKey,
    item,
  } = args;

  const resaleId =
    asNonEmptyString(
      item.resaleId,
    );

  const productId =
    asNonEmptyString(
      item.productId,
    );

  if (
    !resaleId ||
    !productId
  ) {
    return null;
  }

  return {
    itemKey,
    avatarId,

    type: "resale",

    resaleId,
    productId,

    qty: 1,

    ...buildDisplayFields(
      item,
    ),

    catalog: null,
  };
}

function cartItemToDisplayItem(
  args: {
    avatarId: string;
    itemKey: string;
    item: CartItemDTO;
  },
): CartDisplayItem | null {
  if (
    args.item.type === "list"
  ) {
    return listCartItemToDisplayItem(
      args,
    );
  }

  if (
    args.item.type === "resale"
  ) {
    return resaleCartItemToDisplayItem(
      args,
    );
  }

  return null;
}

export function cartDTOToDisplayItems(
  cart: CartDTO,
): CartDisplayItem[] {
  const avatarId =
    cart.avatarId.trim();

  if (
    !isRecord(cart.items) ||
    Array.isArray(cart.items)
  ) {
    return [];
  }

  return Object.entries(
    cart.items,
  )
    .map(
      ([
        rawItemKey,
        item,
      ]) => {
        const itemKey =
          rawItemKey.trim();

        if (!itemKey) {
          return null;
        }

        return cartItemToDisplayItem({
          avatarId,
          itemKey,
          item,
        });
      },
    )
    .filter(
      (
        item,
      ): item is CartDisplayItem =>
        item !== null,
    );
}