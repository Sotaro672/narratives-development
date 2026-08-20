// frontend/console/shell/src/features/transportation/application/transportationService.ts

import {
  createTransportationFeeSettingHTTP,
  getTransportationFeeSettingHTTP,
  getTransportationMasterHTTP,
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
  companyId: string;
  regions: TransportationRegionVM[];
  islandRates: TransportationIslandRateVM[];
  createdAt: string;
  updatedAt: string;
};

export type TransportationSaveInput = {
  regions: TransportationRegionVM[];
  islandRates: TransportationIslandRateVM[];
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
    companyId: setting.companyId,
    regions: buildRegionVMs(master, setting),
    islandRates: buildIslandRateVMs(master, setting),
    createdAt: setting.createdAt,
    updatedAt: setting.updatedAt,
  };
}

// ============================================================
// Empty view model
// ============================================================

export function buildEmptyTransportationVM(master: TransportationMaster): TransportationVM {
  return {
    companyId: "",
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
    updatedAt: "",
  };
}

// ============================================================
// Validation
// ============================================================

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
  validateRegions(input.regions);
  validateIslandRates(input.islandRates);
}

// ============================================================
// Request mapper
// ============================================================

function toTransportationFeeSettingInput(input: TransportationSaveInput): TransportationFeeSettingInput {
  validateSaveInput(input);

  return {
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

export async function fetchTransportationVM(): Promise<TransportationVM> {
  const [master, setting] = await Promise.all([
    getTransportationMasterHTTP(),
    getTransportationFeeSettingHTTP(),
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

export async function updateTransportation(input: TransportationSaveInput): Promise<TransportationVM> {
  const payload = toTransportationFeeSettingInput(input);

  const [master, setting] = await Promise.all([
    getTransportationMasterHTTP(),
    updateTransportationFeeSettingHTTP(payload),
  ]);

  return buildTransportationVM(master, setting);
}