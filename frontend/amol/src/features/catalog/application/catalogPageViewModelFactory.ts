// frontend/amol/src/features/catalog/application/catalogPageViewModelFactory.ts

import type {
  CatalogListImage,
  CatalogModelVariation,
  CatalogResponse,
  MeasurementTableRow,
  ModelColorOption,
} from "../../shared/types/catalog";
import type { ProductCategoryKind } from "../../shared/types/category";
import type { ProductBlueprintReviewPage } from "../../shared/types/review";

import {
  createCatalogImages,
  hasMultipleCatalogImages,
  resolveActiveCatalogImage,
} from "./catalogImageFactory";
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
  resolveSelectedModelStock,
  type CatalogAlcoholOption,
} from "./catalogSelectionFactory";

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

export function createCatalogPageViewModel(args: {
  catalog: CatalogResponse | null;
  reviews: ProductBlueprintReviewPage | null;
  selectedColorKey: string;
  selectedSize: string;
  selectedModelId: string;
  activeImageIndex: number;
  isAddingToCart: boolean;
}): CatalogPageViewModel {
  const catalogKind = args.catalog?.productBlueprint.productBlueprintCategoryKind ?? "unknown";
  const isAlcoholCatalog = catalogKind === "alcohol";
  const models = args.catalog?.modelVariations;
  const catalogImages = createCatalogImages(args.catalog?.listImages);

  const activeImage = resolveActiveCatalogImage({
    images: catalogImages,
    activeImageIndex: args.activeImageIndex,
  });

  const measurementRows = createCatalogMeasurementRows({
    models,
    isAlcoholCatalog,
  });
  const measurementKeys = createCatalogMeasurementKeys(measurementRows);

  const alcoholOptions = isAlcoholCatalog ? createCatalogAlcoholOptions(models) : [];
  const colorOptions = isAlcoholCatalog ? [] : createCatalogColorOptions(models);
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

  const selectedModelStock = resolveSelectedModelStock({
    inventory: args.catalog?.inventory,
    selectedModel,
  });

  const hasSelectedModelStock = hasSelectedCatalogModelStock(selectedModelStock);

  return {
    catalogKind,
    isAlcoholCatalog,
    activeImage,
    catalogImages,
    hasMultipleImages: hasMultipleCatalogImages(catalogImages),
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