// frontend/amol/src/features/catalog/application/catalogPageViewModelFactory.ts

import type {
  CatalogListImage,
  CatalogModelVariation,
  CatalogResponse,
  MeasurementTableRow,
  ModelColorOption,
} from "../../shared/types/catalog";
import type {
  ProductBlueprintCategoryRoot,
  ProductCategoryKind,
} from "../../shared/types/category";
import type { ProductBlueprintReviewPage } from "../../shared/types/review";
import {
  createCatalogMeasurementKeys,
  createCatalogMeasurementRows,
  shouldShowCatalogMeasurementTable,
} from "./catalogMeasurementFactory";
import {
  canAddSelectedCatalogItemToCart,
  createCatalogAlcoholOptions,
  createCatalogColorOptions,
  createCatalogSizeOptions,
  hasSelectedCatalogModelStock,
  resolveSelectedCatalogModel,
  resolveSelectedModelPrice,
  type CatalogAlcoholOption,
} from "./catalogSelectionFactory";
import { getAvailableStock } from "../utils/model";

export type CatalogPageViewModel = {
  catalogKind: ProductCategoryKind;
  isAlcoholCatalog: boolean;
  activeImage: CatalogListImage | undefined;
  catalogImages: CatalogListImage[];
  hasMultipleImages: boolean;
  firstPrice: CatalogResponse["list"]["prices"][number] | undefined;
  reviewSummary: CatalogResponse["productReviewSummary"] | undefined;
  reviewItems: ProductBlueprintReviewPage["items"];
  measurementRows: MeasurementTableRow[];
  measurementKeys: string[];
  shouldShowMeasurementTable: boolean;
  alcoholOptions: CatalogAlcoholOption[];
  colorOptions: ModelColorOption[];
  sizeOptions: string[];
  selectedModel: CatalogModelVariation | null;
  selectedModelPrice: CatalogResponse["list"]["prices"][number] | undefined;
  selectedModelStock: number | undefined;
  hasSelectedModelStock: boolean;
  canAddToCart: boolean;
};

function isProductBlueprintCategoryRoot(
  value: string | undefined,
): value is ProductBlueprintCategoryRoot {
  switch (value) {
    case "apparel":
    case "alcohol":
    case "cosmetics":
    case "healthcare":
    case "other":
      return true;
    default:
      return false;
  }
}

function resolveCatalogKind(
  catalog: CatalogResponse | null,
): ProductCategoryKind {
  const root =
    catalog?.productBlueprint.productBlueprintCategoryPath?.[0];

  return isProductBlueprintCategoryRoot(root) ? root : "unknown";
}

export function createCatalogPageViewModel(args: {
  catalog: CatalogResponse | null;
  reviews: ProductBlueprintReviewPage | null;
  selectedColorKey: string;
  selectedSize: string;
  selectedModelId: string;
  activeImageIndex: number;
  isAddingToCart: boolean;
}): CatalogPageViewModel {
  const catalogKind = resolveCatalogKind(args.catalog);
  const isAlcoholCatalog = catalogKind === "alcohol";
  const models = args.catalog?.modelVariations;
  const catalogImages = args.catalog?.listImages ?? [];
  const activeImage = catalogImages[args.activeImageIndex];

  const measurementRows = createCatalogMeasurementRows({
    models,
    isAlcoholCatalog,
  });

  const measurementKeys = createCatalogMeasurementKeys(measurementRows);
  const alcoholOptions = isAlcoholCatalog
    ? createCatalogAlcoholOptions(models)
    : [];
  const colorOptions = isAlcoholCatalog
    ? []
    : createCatalogColorOptions(models);

  const sizeOptions = isAlcoholCatalog
    ? []
    : createCatalogSizeOptions({
        models,
        selectedColorKey: args.selectedColorKey,
      });

  const selectedModel = resolveSelectedCatalogModel({
    models,
    selectedModelId: args.selectedModelId,
    selectedColorKey: args.selectedColorKey,
    selectedSize: args.selectedSize,
    isAlcoholCatalog,
  });

  const selectedModelPrice = resolveSelectedModelPrice({
    prices: args.catalog?.list.prices,
    selectedModel,
  });

  const selectedModelStock = selectedModel
    ? getAvailableStock(args.catalog?.inventory, selectedModel.id)
    : undefined;

  const hasSelectedModelStock =
    hasSelectedCatalogModelStock(selectedModelStock);

  return {
    catalogKind,
    isAlcoholCatalog,
    activeImage,
    catalogImages,
    hasMultipleImages: catalogImages.length > 1,
    firstPrice: args.catalog?.list.prices[0],
    reviewSummary: args.catalog?.productReviewSummary,
    reviewItems: args.reviews?.items ?? [],
    measurementRows,
    measurementKeys,
    shouldShowMeasurementTable: shouldShowCatalogMeasurementTable({
      isAlcoholCatalog,
      measurementRows,
      measurementKeys,
    }),
    alcoholOptions,
    colorOptions,
    sizeOptions,
    selectedModel,
    selectedModelPrice,
    selectedModelStock,
    hasSelectedModelStock,
    canAddToCart: canAddSelectedCatalogItemToCart({
      hasCatalog: Boolean(args.catalog),
      hasSelectedModel: Boolean(selectedModel),
      hasSelectedModelStock,
      isAddingToCart: args.isAddingToCart,
    }),
  };
}