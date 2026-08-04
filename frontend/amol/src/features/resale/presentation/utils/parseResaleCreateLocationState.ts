// frontend/amol/src/features/resale/presentation/utils/parseResaleCreateLocationState.ts

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import type {
  ResaleCreatePageLocationState,
  ResaleCreateTarget,
} from "../types/resaleCreatePageTypes";

const RESALE_CREATE_LOCATION_STATE_KEYS = [
  "mintAddress",
  "productId",
  "brandId",
  "brandName",
  "productName",
  "productBlueprintId",
  "tokenBlueprintId",
  "tokenName",
  "tokenIconUrl",
] as const satisfies readonly (
  keyof ResaleCreatePageLocationState
)[];

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function extractLocationState(
  value: unknown,
): ResaleCreatePageLocationState {
  if (!isRecord(value)) {
    return {};
  }

  const state:
    ResaleCreatePageLocationState = {};

  RESALE_CREATE_LOCATION_STATE_KEYS.forEach(
    (key) => {
      const fieldValue =
        value[key];

      if (
        typeof fieldValue ===
        "string"
      ) {
        state[key] =
          fieldValue;
      }
    },
  );

  return state;
}

export function parseResaleCreateLocationState(
  value: unknown,
): ResaleCreateTarget {
  const state =
    extractLocationState(
      value,
    );

  return {
    mintAddress:
      textOrEmpty(
        state.mintAddress,
      ),

    productId:
      textOrEmpty(
        state.productId,
      ),

    brandId:
      textOrEmpty(
        state.brandId,
      ),

    brandName:
      textOrEmpty(
        state.brandName,
      ),

    productName:
      textOrEmpty(
        state.productName,
      ),

    productBlueprintId:
      textOrEmpty(
        state.productBlueprintId,
      ),

    tokenBlueprintId:
      textOrEmpty(
        state.tokenBlueprintId,
      ),

    tokenName:
      textOrEmpty(
        state.tokenName,
      ),

    tokenIconUrl:
      textOrEmpty(
        state.tokenIconUrl,
      ),
  };
}