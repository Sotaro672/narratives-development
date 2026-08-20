// frontend/console/shell/src/features/transportation/application/transportationService.ts

import {
  createTransportationFeeSettingHTTP,
  getTransportationFeeSettingHTTP,
  getTransportationMasterHTTP,
  listTransportationFeeSettingsHTTP,
  updateTransportationFeeSettingHTTP,
} from "../infrastracture/transportationApi";

import {
  PREFECTURE_NAME_BY_CODE,
  REGION_NAME_BY_CODE,
  type IslandCode,
  type PrefectureCode,
  type TransportationFeeSetting,
  type TransportationFeeSettingInput,
  type TransportationIslandRate,
  type TransportationMaster,
  type TransportationRegion,
} from "../../../shared/types/transporation";

// ============================================================
// View model
// ============================================================

export type TransportationPrefectureRateVM = {
  prefectureCode: PrefectureCode;
  prefectureName: string;
  amount: number | null;
};

export type TransportationRegionVM = {
  region: TransportationRegion;
  regionName: string;
  prefectures: TransportationPrefectureRateVM[];
};

export type TransportationIslandRateVM = {
  islandCode: IslandCode;
  islandName: string;
  prefectureCode: PrefectureCode;
  prefectureName: string;
  amount: number | null;
};

export type TransportationVM = {
  id: string;
  companyId: string;
  name: string;
  regions: TransportationRegionVM[];
  islandRates: TransportationIslandRateVM[];
  createdAt: string;
  createdBy: string;
  createdByName?: string;
  updatedAt: string;
  updatedBy: string;
  updatedByName?: string;
};

export type TransportationSaveInput = {
  name: string;
  regions: TransportationRegionVM[];
  islandRates: TransportationIslandRateVM[];
};

export type TransportationListItemVM = {
  id: string;
  companyId: string;
  name: string;
  createdAt: string;
  createdBy: string;
  createdByName?: string;
  updatedAt: string;
  updatedBy: string;
  updatedByName?: string;
};

// ============================================================
// Master
// ============================================================

const PREFECTURE_CODES = Object.keys(PREFECTURE_NAME_BY_CODE) as PrefectureCode[];

// ============================================================
// View model builder
// ============================================================

function buildPrefectureRateMap(setting: TransportationFeeSetting): Map<PrefectureCode, number> {
  const result = new Map<PrefectureCode, number>();

  for (const rate of setting.prefectureRates) {
    result.set(rate.prefectureCode, rate.amount);
  }

  return result;
}

function buildIslandRateMap(setting: TransportationFeeSetting): Map<IslandCode, TransportationIslandRate> {
  const result = new Map<IslandCode, TransportationIslandRate>();

  for (const rate of setting.islandRates) {
    if (result.has(rate.islandCode)) {
      throw new Error(`transportation_duplicate_island_rate:${rate.islandCode}`);
    }

    result.set(rate.islandCode, rate);
  }

  return result;
}

function buildRegionVMs(master: TransportationMaster, setting: TransportationFeeSetting): TransportationRegionVM[] {
  const rateMap = buildPrefectureRateMap(setting);

  return master.regions
    .filter((group) => group.region !== "islands")
    .map((group) => ({
      region: group.region,
      regionName: REGION_NAME_BY_CODE[group.region],
      prefectures: group.prefectureCodes.map((prefectureCode) => {
        const amount = rateMap.get(prefectureCode);

        if (amount === undefined) {
          throw new Error(`transportation_prefecture_rate_not_found:${prefectureCode}`);
        }

        return {
          prefectureCode,
          prefectureName: PREFECTURE_NAME_BY_CODE[prefectureCode],
          amount,
        };
      }),
    }));
}

function buildIslandRateVMs(master: TransportationMaster, setting: TransportationFeeSetting): TransportationIslandRateVM[] {
  const rateMap = buildIslandRateMap(setting);

  return master.islands.map((definition) => {
    const savedRate = rateMap.get(definition.islandCode);

    if (savedRate && savedRate.prefectureCode !== definition.prefectureCode) {
      throw new Error(`transportation_island_prefecture_mismatch:${definition.islandCode}`);
    }

    return {
      islandCode: definition.islandCode,
      islandName: definition.displayName,
      prefectureCode: definition.prefectureCode,
      prefectureName: PREFECTURE_NAME_BY_CODE[definition.prefectureCode],
      amount: savedRate ? savedRate.amount : null,
    };
  });
}

export function buildTransportationVM(master: TransportationMaster, setting: TransportationFeeSetting): TransportationVM {
  return {
    id: setting.id,
    companyId: setting.companyId,
    name: setting.name,
    regions: buildRegionVMs(master, setting),
    islandRates: buildIslandRateVMs(master, setting),
    createdAt: setting.createdAt,
    createdBy: setting.createdBy,
    createdByName: setting.createdByName,
    updatedAt: setting.updatedAt,
    updatedBy: setting.updatedBy,
    updatedByName: setting.updatedByName,
  };
}

function buildTransportationListItemVM(setting: TransportationFeeSetting): TransportationListItemVM {
  return {
    id: setting.id,
    companyId: setting.companyId,
    name: setting.name,
    createdAt: setting.createdAt,
    createdBy: setting.createdBy,
    createdByName: setting.createdByName,
    updatedAt: setting.updatedAt,
    updatedBy: setting.updatedBy,
    updatedByName: setting.updatedByName,
  };
}

// ============================================================
// Empty view model
// ============================================================

export function buildEmptyTransportationVM(master: TransportationMaster): TransportationVM {
  return {
    id: "",
    companyId: "",
    name: "",
    regions: master.regions
      .filter((group) => group.region !== "islands")
      .map((group) => ({
        region: group.region,
        regionName: REGION_NAME_BY_CODE[group.region],
        prefectures: group.prefectureCodes.map((prefectureCode) => ({
          prefectureCode,
          prefectureName: PREFECTURE_NAME_BY_CODE[prefectureCode],
          amount: null,
        })),
      })),
    islandRates: master.islands.map((definition) => ({
      islandCode: definition.islandCode,
      islandName: definition.displayName,
      prefectureCode: definition.prefectureCode,
      prefectureName: PREFECTURE_NAME_BY_CODE[definition.prefectureCode],
      amount: null,
    })),
    createdAt: "",
    createdBy: "",
    createdByName: undefined,
    updatedAt: "",
    updatedBy: "",
    updatedByName: undefined,
  };
}

// ============================================================
// Validation
// ============================================================

function validateName(name: string): void {
  if (name === "") {
    throw new Error("料金設定名を入力してください。");
  }

  if (Array.from(name).length > 100) {
    throw new Error("料金設定名は100文字以内で入力してください。");
  }
}

function validateAmount(amount: number): void {
  if (!Number.isSafeInteger(amount) || amount < 0) {
    throw new Error("送料は0以上の整数で入力してください。");
  }
}

function requirePrefectureAmount(prefecture: TransportationPrefectureRateVM): number {
  if (prefecture.amount === null) {
    throw new Error(`${prefecture.prefectureName}の送料を入力してください。`);
  }

  validateAmount(prefecture.amount);
  return prefecture.amount;
}

function validateRegions(regions: TransportationRegionVM[]): void {
  const seen = new Set<PrefectureCode>();

  for (const region of regions) {
    if (region.region === "islands") {
      continue;
    }

    for (const prefecture of region.prefectures) {
      requirePrefectureAmount(prefecture);

      if (seen.has(prefecture.prefectureCode)) {
        throw new Error(`都道府県が重複しています: ${prefecture.prefectureName}`);
      }

      seen.add(prefecture.prefectureCode);
    }
  }

  if (seen.size !== PREFECTURE_CODES.length) {
    throw new Error("47都道府県すべての送料を設定してください。");
  }

  for (const prefectureCode of PREFECTURE_CODES) {
    if (!seen.has(prefectureCode)) {
      throw new Error(`${PREFECTURE_NAME_BY_CODE[prefectureCode]}の送料が設定されていません。`);
    }
  }
}

function validateIslandRates(islandRates: TransportationIslandRateVM[]): void {
  const seen = new Set<IslandCode>();

  for (const islandRate of islandRates) {
    if (!islandRate.islandCode) {
      throw new Error("離島コードが設定されていません。");
    }

    if (seen.has(islandRate.islandCode)) {
      throw new Error(`離島送料が重複しています: ${islandRate.islandName}`);
    }

    seen.add(islandRate.islandCode);

    if (islandRate.amount !== null) {
      validateAmount(islandRate.amount);
    }
  }
}

function validateSaveInput(input: TransportationSaveInput): void {
  validateName(input.name);
  validateRegions(input.regions);
  validateIslandRates(input.islandRates);
}

// ============================================================
// Request mapper
// ============================================================

function toTransportationFeeSettingInput(input: TransportationSaveInput): TransportationFeeSettingInput {
  validateSaveInput(input);

  return {
    name: input.name,
    prefectureRates: input.regions
      .filter((region) => region.region !== "islands")
      .flatMap((region) =>
        region.prefectures.map((prefecture) => ({
          prefectureCode: prefecture.prefectureCode,
          amount: requirePrefectureAmount(prefecture),
        })),
      ),
    islandRates: input.islandRates.flatMap((islandRate) => {
      if (islandRate.amount === null) {
        return [];
      }

      return [{
        islandCode: islandRate.islandCode,
        prefectureCode: islandRate.prefectureCode,
        amount: islandRate.amount,
      }];
    }),
  };
}

// ============================================================
// Query
// ============================================================

export async function listTransportationVMs(): Promise<TransportationListItemVM[]> {
  const settings = await listTransportationFeeSettingsHTTP();
  return settings.map(buildTransportationListItemVM);
}

export async function fetchTransportationVM(transportationId: string): Promise<TransportationVM> {
  const [master, setting] = await Promise.all([
    getTransportationMasterHTTP(),
    getTransportationFeeSettingHTTP(transportationId),
  ]);

  return buildTransportationVM(master, setting);
}

export async function fetchEmptyTransportationVM(): Promise<TransportationVM> {
  const master = await getTransportationMasterHTTP();
  return buildEmptyTransportationVM(master);
}

// ============================================================
// Command
// ============================================================

export async function createTransportation(input: TransportationSaveInput): Promise<TransportationVM> {
  const payload = toTransportationFeeSettingInput(input);

  const [master, setting] = await Promise.all([
    getTransportationMasterHTTP(),
    createTransportationFeeSettingHTTP(payload),
  ]);

  return buildTransportationVM(master, setting);
}

export async function updateTransportation(
  transportationId: string,
  input: TransportationSaveInput,
): Promise<TransportationVM> {
  const payload = toTransportationFeeSettingInput(input);

  const [master, setting] = await Promise.all([
    getTransportationMasterHTTP(),
    updateTransportationFeeSettingHTTP(transportationId, payload),
  ]);

  return buildTransportationVM(master, setting);
}