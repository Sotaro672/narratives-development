// frontend/amol/src/features/catalog/application/catalogSelectionFactory.ts

import { toSafeColorRGB } from "../../../components/utils/color";
import type {
  CatalogInventory,
  CatalogListPrice,
  CatalogModelVariation,
  ModelColorOption,
} from "../../shared/types/catalog";
import { getAvailableStock, getModelColorKey } from "../utils/model";
import {
  formatAlcoholModelLabel,
  formatApparelSizeLabel,
} from "./catalogModelMapper";

export type CatalogAlcoholOption = {
  modelId: string;
  modelNumber: string;
  volumeValue: number | null;
  volumeUnit: string;
  label: string;
};

export function createCatalogAlcoholOptions(
  models: CatalogModelVariation[] | undefined,
): CatalogAlcoholOption[] {
  return (models ?? [])
    .filter((model) => model.kind === "alcohol")
    .map((model) => ({
      modelId: model.id,
      modelNumber: model.modelNumber,
      volumeValue: model.volumeValue ?? null,
      volumeUnit: model.volumeUnit ?? "",
      label: formatAlcoholModelLabel(model),
    }));
}

export function createCatalogColorOptions(
  models: CatalogModelVariation[] | undefined,
): ModelColorOption[] {
  const options = new Map<string, ModelColorOption>();

  for (const model of models ?? []) {
    if (model.kind !== "apparel") continue;

    const key = getModelColorKey(model);
    if (options.has(key)) continue;

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
}): string[] {
  const sizes = new Set<string>();

  for (const model of args.models ?? []) {
    if (model.kind !== "apparel") continue;
    if (args.selectedColorKey && getModelColorKey(model) !== args.selectedColorKey) continue;

    sizes.add(formatApparelSizeLabel(model));
  }

  return Array.from(sizes);
}

function resolveSelectedApparelModels(args: {
  models: CatalogModelVariation[] | undefined;
  selectedColorKey: string;
  selectedSize: string;
}): CatalogModelVariation[] {
  if (!args.selectedColorKey || !args.selectedSize) return [];

  return (args.models ?? []).filter(
    (model) =>
      model.kind === "apparel" &&
      getModelColorKey(model) === args.selectedColorKey &&
      formatApparelSizeLabel(model) === args.selectedSize,
  );
}

export function resolveSelectedCatalogModel(args: {
  models: CatalogModelVariation[] | undefined;
  selectedModelId: string;
  selectedColorKey: string;
  selectedSize: string;
  isAlcoholCatalog: boolean;
}): CatalogModelVariation | null {
  const models = args.models ?? [];

  if (args.isAlcoholCatalog) {
    if (!args.selectedModelId) return null;

    return (
      models.find(
        (model) =>
          model.kind === "alcohol" &&
          model.id === args.selectedModelId,
      ) ?? null
    );
  }

  const matchedModels = resolveSelectedApparelModels({
    models,
    selectedColorKey: args.selectedColorKey,
    selectedSize: args.selectedSize,
  });

  return matchedModels.length === 1 ? matchedModels[0] : null;
}

export function resolveSelectedModelPrice(args: {
  prices: CatalogListPrice[] | undefined;
  selectedModel: CatalogModelVariation | null;
}): CatalogListPrice | undefined {
  const selectedModel = args.selectedModel;
  if (!selectedModel) return undefined;

  return args.prices?.find((price) => price.modelId === selectedModel.id);
}

export function resolveSelectedModelStock(args: {
  inventory: CatalogInventory | undefined;
  selectedModel: CatalogModelVariation | null;
}): number | undefined {
  const selectedModel = args.selectedModel;
  if (!selectedModel) return undefined;

  return getAvailableStock(args.inventory, selectedModel.id);
}

export function hasSelectedCatalogModelStock(selectedModelStock: number | undefined): boolean {
  return typeof selectedModelStock === "number" && selectedModelStock > 0;
}

export function canAddSelectedCatalogItemToCart(args: {
  hasCatalog: boolean;
  hasSelectedModel: boolean;
  hasSelectedModelStock: boolean;
  isAddingToCart: boolean;
}): boolean {
  return args.hasCatalog && args.hasSelectedModel && args.hasSelectedModelStock && !args.isAddingToCart;
}