// frontend/mall/src/features/catalog/application/catalogSelectionFactory.ts

import { toSafeColorRGB } from "../../../components/utils/color";
import type {
  CatalogListPrice,
  CatalogModelVariation,
  CatalogProductBlueprintModelRef,
  ModelColorOption,
} from "../../shared/types/catalog";
import { getModelColorKey } from "../utils/model";
import { formatAlcoholModelLabel, formatApparelSizeLabel } from "./catalogModelMapper";

export type CatalogAlcoholOption = {
  modelId: string;
  modelNumber: string;
  volumeValue: number | null;
  volumeUnit: string;
  label: string;
};

/**
 * ProductBlueprint.modelRefs.displayOrder を正としてModel variationを並べ替える。
 *
 * - modelRefが存在するModelをdisplayOrder昇順で先に並べる。
 * - 同一displayOrderの場合は元のmodelVariationsの順序を維持する。
 * - modelRefが存在しないModelは末尾へ配置し、元の順序を維持する。
 * - 引数のmodels自体は破壊しない。
 */
export function sortCatalogModelsByDisplayOrder(args: {
  models: CatalogModelVariation[] | undefined;
  modelRefs: CatalogProductBlueprintModelRef[] | undefined;
}): CatalogModelVariation[] {
  const models = args.models ?? [];
  const modelRefs = args.modelRefs ?? [];

  if (models.length <= 1 || modelRefs.length === 0) {
    return [...models];
  }

  const displayOrderByModelId = new Map<string, number>();

  for (const modelRef of modelRefs) {
    const modelId = modelRef.modelId?.trim();
    if (!modelId || !Number.isFinite(modelRef.displayOrder)) continue;

    displayOrderByModelId.set(modelId, modelRef.displayOrder);
  }

  return models
    .map((model, originalIndex) => ({
      model,
      originalIndex,
      displayOrder: displayOrderByModelId.get(model.id),
    }))
    .sort((left, right) => {
      const leftHasDisplayOrder = left.displayOrder !== undefined;
      const rightHasDisplayOrder = right.displayOrder !== undefined;

      if (leftHasDisplayOrder !== rightHasDisplayOrder) {
        return leftHasDisplayOrder ? -1 : 1;
      }

      if (
        left.displayOrder !== undefined &&
        right.displayOrder !== undefined &&
        left.displayOrder !== right.displayOrder
      ) {
        return left.displayOrder - right.displayOrder;
      }

      return left.originalIndex - right.originalIndex;
    })
    .map(({ model }) => model);
}

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

  return matchedModels.length === 1
    ? matchedModels[0]
    : null;
}

export function resolveSelectedModelPrice(args: {
  prices: CatalogListPrice[] | undefined;
  selectedModel: CatalogModelVariation | null;
}): CatalogListPrice | undefined {
  const selectedModel = args.selectedModel;
  if (!selectedModel) return undefined;

  return args.prices?.find(
    (price) => price.modelId === selectedModel.id,
  );
}

export function hasSelectedCatalogModelStock(
  selectedModelStock: number | undefined,
): boolean {
  return (
    typeof selectedModelStock === "number" &&
    selectedModelStock > 0
  );
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