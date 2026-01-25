// frontend\console\inventory\src\application\listCreate\listCreateService.tsx

export type {
  ImageInputRef,
  ListCreateRouteParams,
  ResolvedListCreateParams,
  PriceRow,
  PriceRowEx,
  CreateListPriceRow,
} from "./listCreate.types";

export {
  resolveListCreateParams,
  computeListCreateTitle,
  canFetchListCreate,
  buildListCreateFetchInput,
  getInventoryIdFromDTO,
  shouldRedirectToInventoryIdRoute,
  buildInventoryDetailPath,
  buildInventoryListCreatePath,
  buildBackPath,
  buildAfterCreatePath,
} from "./listCreate.routing";

export {
  extractDisplayStrings,
  mapDTOToPriceRows,
  attachModelIdsFromDTO,
  initPriceRowsFromDTO,
} from "./listCreate.dto";

export {
  normalizeCreateListPriceRows,
  buildCreateListInput,
  validateCreateListInput,
} from "./listCreate.input";

export { dedupeFiles, uploadListImagesPolicyA } from "./listCreate.images";

export { loadListCreateDTOFromParams, createListWithImages } from "./listCreate.usecase";
