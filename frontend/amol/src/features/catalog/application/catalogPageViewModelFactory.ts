// frontend/amol/src/features/catalog/application/catalogPageViewModelFactory.ts

import type {
  CatalogListImage,
  CatalogModelVariation,
  CatalogProductBlueprintReviewPage,
  CatalogResponse,
  MeasurementTableRow,
  ModelColorOption,
} from "../types";
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
  mapCatalogKind,
  type CatalogModelKind,
} from "./catalogModelMapper";
import {
  canAddSelectedCatalogItemToCart,
  createCatalogColorOptions,
  createCatalogSizeOptions,
  hasSelectedCatalogModelStock,
  resolveSelectedCatalogModel,
  resolveSelectedModelPrice,
  resolveSelectedModelStock,
} from "./catalogSelectionFactory";

export type CatalogPageViewModel = {
  catalogKind: CatalogModelKind;
  isAlcoholCatalog: boolean;

  activeImage: CatalogListImage | undefined;
  catalogImages: CatalogListImage[];
  hasMultipleImages: boolean;

  firstPrice: CatalogResponse["list"]["prices"][number] | undefined;
  reviewSummary: CatalogResponse["productReviewSummary"] | undefined;
  reviewItems: CatalogProductBlueprintReviewPage["items"];

  measurementRows: MeasurementTableRow[];
  measurementKeys: string[];
  shouldShowMeasurementTable: boolean;

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
  reviews: CatalogProductBlueprintReviewPage | null;
  selectedColorKey: string;
  selectedSize: string;
  activeImageIndex: number;
  isAddingToCart: boolean;
}): CatalogPageViewModel {
  const catalogKind = mapCatalogKind(args.catalog?.modelVariations);
  const isAlcoholCatalog = catalogKind === "alcohol";

  const catalogImages = createCatalogImages(args.catalog?.listImages);
  const activeImage = resolveActiveCatalogImage({
    images: catalogImages,
    activeImageIndex: args.activeImageIndex,
  });

  const measurementRows = createCatalogMeasurementRows({
    models: args.catalog?.modelVariations,
    isAlcoholCatalog,
  });

  const measurementKeys = createCatalogMeasurementKeys(measurementRows);

  const colorOptions = createCatalogColorOptions({
    models: args.catalog?.modelVariations,
    isAlcoholCatalog,
  });

  const sizeOptions = createCatalogSizeOptions({
    models: args.catalog?.modelVariations,
    selectedColorKey: args.selectedColorKey,
    isAlcoholCatalog,
  });

  const selectedModel = resolveSelectedCatalogModel({
    models: args.catalog?.modelVariations,
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

  const hasSelectedModelStock =
    hasSelectedCatalogModelStock(selectedModelStock);

  return {
    catalogKind,
    isAlcoholCatalog,

    activeImage,
    catalogImages,
    hasMultipleImages: hasMultipleCatalogImages(catalogImages),

    firstPrice: args.catalog?.list.prices?.[0],
    reviewSummary: args.catalog?.productReviewSummary,
    reviewItems: args.reviews?.items ?? [],

    measurementRows,
    measurementKeys,
    shouldShowMeasurementTable: shouldShowCatalogMeasurementTable({
      isAlcoholCatalog,
      measurementRows,
      measurementKeys,
    }),

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