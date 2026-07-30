// frontend/console/inventory/src/infrastructure/http/inventoryRepositoryHTTP.types.ts

import type {
CategoryFieldValues,
ProductBlueprintCategorySnapshot,
} from "../../../productBlueprint/domain/productBlueprintCategory";

// ---------------------------------------------------------
// Inventory 一覧DTO（管理一覧）
// GET /inventory
// ---------------------------------------------------------

export type InventoryListRowDTO = {
productBlueprintId: string;
productName: string;

tokenBlueprintId: string;
tokenName: string;

modelNumber: string;
availableStock: number;
reservedCount: number;
};

// ---------------------------------------------------------
// ProductBlueprint の modelRefs
//
// backend raw:
// - ModelID
// - DisplayOrder
//
// frontend DTO:
// - modelId
// - displayOrder
// ---------------------------------------------------------

export type ProductBlueprintModelRefDTO = {
modelId: string;
displayOrder: number;
};

// ---------------------------------------------------------
// ProductBlueprint patch
//
// productBlueprintCategory は ProductBlueprintCard が期待する
// ProductBlueprintCategorySnapshot と同じ型を使う。
// backend raw の ID / Code / NameJa / NameEn / Kind / Path は
// inventoryRepositoryHTTP.mappers.ts で
// id / code / nameJa / nameEn / kind / path へ変換する。
// ---------------------------------------------------------

export type ProductBlueprintPatchDTO = {
productName?: string | null;
description?: string | null;

brandId?: string | null;
brandName?: string | null;
companyId?: string | null;

productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;
categoryFields?: CategoryFieldValues | null;

fit?: string | null;
material?: string | null;
weight?: number | null;
qualityAssurance?: string[] | null;

productIdTag?: string | {
type?: string;
} | null;

modelRefs?: ProductBlueprintModelRefDTO[] | null;
};

// ---------------------------------------------------------
// TokenBlueprint patch（Inventory 詳細で使用）
// ---------------------------------------------------------

export type TokenBlueprintPatchDTO = {
tokenName?: string | null;
symbol?: string | null;
brandId?: string | null;
brandName?: string | null;
description?: string | null;
iconUrl?: string | null;
};

// ---------------------------------------------------------
// Inventory Detail Row
//
// GET /inventory/{inventoryId} の rows を唯一の正とする。
// /models/by-blueprint/{productBlueprintId}/variations の response は使わない。
//
// apparel:
// - modelId
// - kind
// - modelNumber
// - size
// - color
// - rgb
// - stock
//
// alcohol:
// - modelId
// - kind
// - modelNumber
// - volumeValue
// - volumeUnit
// - stock
// ---------------------------------------------------------

export type InventoryDetailRowDTO = {
modelId: string;
kind?: string | null;

modelNumber: string;
stock: number;

// apparel
size?: string | null;
color?: string | null;
rgb?: number | null;

// alcohol
volumeValue?: number | null;
volumeUnit?: string | null;
};

// ---------------------------------------------------------
// Inventory Detail DTO
// GET /inventory/{inventoryId}
// ---------------------------------------------------------

export type InventoryDetailDTO = {
inventoryId: string;

tokenBlueprintId: string;
productBlueprintId: string;

productBlueprintPatch: ProductBlueprintPatchDTO;
tokenBlueprintPatch?: TokenBlueprintPatchDTO;

rows: InventoryDetailRowDTO[];
totalStock: number;

updatedAt?: string;
};
