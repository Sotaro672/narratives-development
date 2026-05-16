// frontend/amol/src/features/catalog/application/catalogSelectionFactory.ts

import { toSafeColorRGB } from "../../../components/utils/color";
import type {
  CatalogInventory,
  CatalogListPrice,
  CatalogModelVariation,
  ModelColorOption,
} from "../types";
import { getAvailableStock, getModelColorKey } from "../utils/model";
import {
  createAlcoholSelectionKey,
  formatAlcoholModelLabel,
  formatAlcoholSizeLabel,
  formatApparelSizeLabel,
  mapCatalogModelKind,
} from "./catalogModelMapper";

export function createCatalogColorOptions(args: {
  models: CatalogModelVariation[] | undefined;
  isAlcoholCatalog: boolean;
}): ModelColorOption[] {
  const options = new Map<string, ModelColorOption>();

  for (const model of args.models ?? []) {
    if (args.isAlcoholCatalog || mapCatalogModelKind(model) === "alcohol") {
      const key = createAlcoholSelectionKey(model);

      if (!key || options.has(key)) {
        continue;
      }

      options.set(key, {
        key,
        colorName: formatAlcoholModelLabel(model),
        colorRGB: 0,
      });

      continue;
    }

    const key = getModelColorKey(model);

    if (options.has(key)) {
      continue;
    }

    options.set(key, {
      key,
      colorName: model.colorName?.trim() || "-",
      colorRGB: toSafeColorRGB(model.colorRGB),
    });
  }

  return Array.from(options.values());
}

export function createCatalogSizeOptions(args: {
  models: CatalogModelVariation[] | undefined;
  selectedColorKey: string;
  isAlcoholCatalog: boolean;
}): string[] {
  const sizes = new Set<string>();

  for (const model of args.models ?? []) {
    if (args.isAlcoholCatalog || mapCatalogModelKind(model) === "alcohol") {
      if (
        args.selectedColorKey &&
        createAlcoholSelectionKey(model) !== args.selectedColorKey
      ) {
        continue;
      }

      sizes.add(formatAlcoholSizeLabel(model));
      continue;
    }

    if (
      args.selectedColorKey &&
      getModelColorKey(model) !== args.selectedColorKey
    ) {
      continue;
    }

    sizes.add(formatApparelSizeLabel(model));
  }

  return Array.from(sizes);
}

export function resolveSelectedCatalogModels(args: {
  models: CatalogModelVariation[] | undefined;
  selectedColorKey: string;
  selectedSize: string;
  isAlcoholCatalog: boolean;
}): CatalogModelVariation[] {
  if (!args.selectedColorKey || !args.selectedSize) {
    return [];
  }

  return (args.models ?? []).filter((model) => {
    if (args.isAlcoholCatalog || mapCatalogModelKind(model) === "alcohol") {
      return (
        createAlcoholSelectionKey(model) === args.selectedColorKey &&
        formatAlcoholSizeLabel(model) === args.selectedSize
      );
    }

    return (
      getModelColorKey(model) === args.selectedColorKey &&
      formatApparelSizeLabel(model) === args.selectedSize
    );
  });
}

export function resolveSelectedCatalogModel(args: {
  models: CatalogModelVariation[] | undefined;
  selectedColorKey: string;
  selectedSize: string;
  isAlcoholCatalog: boolean;
}): CatalogModelVariation | null {
  const matchedModels = resolveSelectedCatalogModels(args);

  return matchedModels.length === 1 ? matchedModels[0] : null;
}

export function resolveSelectedModelPrice(args: {
  prices: CatalogListPrice[] | undefined;
  selectedModel: CatalogModelVariation | null;
}): CatalogListPrice | undefined {
  if (!args.selectedModel) {
    return undefined;
  }

  return args.prices?.find((price) => price.modelId === args.selectedModel?.id);
}

export function resolveSelectedModelStock(args: {
  inventory: CatalogInventory | undefined;
  selectedModel: CatalogModelVariation | null;
}): number | undefined {
  if (!args.selectedModel) {
    return undefined;
  }

  return getAvailableStock(args.inventory, args.selectedModel.id);
}

export function hasSelectedCatalogModelStock(
  selectedModelStock: number | undefined,
): boolean {
  return typeof selectedModelStock === "number" && selectedModelStock > 0;
}

export function canAddSelectedCatalogItemToCart(args: {
  hasCatalog: boolean;
  hasSelectedModel: boolean;
  hasSelectedModelStock: boolean;
  isAddingToCart: boolean;
}): boolean {
  return (
    args.hasCatalog &&
    args.hasSelectedModel &&
    args.hasSelectedModelStock &&
    !args.isAddingToCart
  );
}