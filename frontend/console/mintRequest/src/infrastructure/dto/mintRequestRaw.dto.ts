// frontend/console/mintRequest/src/infrastructure/dto/mintRequestRaw.dto.ts

// ===============================
// ✅ /mint/requests response raw types
// ===============================

export type MintRequestRowRaw = {
  id?: string | null;
  productionId?: string | null;
  inspectionId?: string | null;

  // “mint が埋め込まれて返る” 想定
  mint?: any | null;
  Mint?: any | null;

  // “list row 的に平坦化されて返る”可能性もある
  tokenName?: string | null;
  createdByName?: string | null;
  mintedAt?: string | null;
  minted?: boolean | null;

  [k: string]: any;
};

export type MintRequestsPayloadRaw =
  | {
      rows?: MintRequestRowRaw[] | null;
      Rows?: MintRequestRowRaw[] | null;
      items?: MintRequestRowRaw[] | null;
      Items?: MintRequestRowRaw[] | null;
      data?: MintRequestRowRaw[] | null;
      Data?: MintRequestRowRaw[] | null;
      [k: string]: any;
    }
  | MintRequestRowRaw[];

// ===============================
// ✅ /mint/brands raw types
// ===============================

export type BrandRecordRaw = {
  id?: string;
  name?: string;
  ID?: string;
  Name?: string;
};

export type BrandPageResultDTO = {
  items?: BrandRecordRaw[];
  Items?: BrandRecordRaw[];
};

// ===============================
// ✅ /mint/token_blueprints raw types
// ===============================

export type TokenBlueprintRecordRaw = {
  id?: string;
  name?: string;
  symbol?: string;
  iconUrl?: string;

  ID?: string;
  Name?: string;
  Symbol?: string;
  IconUrl?: string;
};

export type TokenBlueprintPageResultDTO = {
  items?: TokenBlueprintRecordRaw[];
  Items?: TokenBlueprintRecordRaw[];
};
