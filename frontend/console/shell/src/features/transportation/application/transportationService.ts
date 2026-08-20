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
  amount: number;
};

export type TransportationRegionVM = {
  region: TransportationRegion;
  regionName: string;
  prefectures: TransportationPrefectureRateVM[];
};

export type TransportationIslandRateVM = {
  islandCode: string;
  prefectureCode: PrefectureCode;
  prefectureName: string;
  amount: number;
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
// Prefecture master
// ============================================================

const PREFECTURE_CODES = Object.keys(
  PREFECTURE_NAME_BY_CODE,
) as PrefectureCode[];

// ============================================================
// View model builder
// ============================================================

function buildPrefectureRateMap(
  setting: TransportationFeeSetting,
): Map<PrefectureCode, number> {
  const result = new Map<PrefectureCode, number>();

  for (const rate of setting.prefectureRates) {
    result.set(rate.prefectureCode, rate.amount);
  }

  return result;
}

function buildRegionVMs(
  master: TransportationMaster,
  setting: TransportationFeeSetting,
): TransportationRegionVM[] {
  const rateMap = buildPrefectureRateMap(setting);

  return master.regions.map((group) => ({
    region: group.region,
    regionName: REGION_NAME_BY_CODE[group.region],
    prefectures: group.prefectureCodes.map((prefectureCode) => {
      const amount = rateMap.get(prefectureCode);

      if (amount === undefined) {
        throw new Error(
          `transportation_prefecture_rate_not_found:${prefectureCode}`,
        );
      }

      return {
        prefectureCode,
        prefectureName: PREFECTURE_NAME_BY_CODE[prefectureCode],
        amount,
      };
    }),
  }));
}

function buildIslandRateVMs(
  islandRates: TransportationIslandRate[],
): TransportationIslandRateVM[] {
  return islandRates.map((rate) => ({
    islandCode: rate.islandCode,
    prefectureCode: rate.prefectureCode,
    prefectureName: PREFECTURE_NAME_BY_CODE[rate.prefectureCode],
    amount: rate.amount,
  }));
}

export function buildTransportationVM(
  master: TransportationMaster,
  setting: TransportationFeeSetting,
): TransportationVM {
  return {
    companyId: setting.companyId,
    regions: buildRegionVMs(master, setting),
    islandRates: buildIslandRateVMs(setting.islandRates),
    createdAt: setting.createdAt,
    updatedAt: setting.updatedAt,
  };
}

// ============================================================
// Empty view model
// ============================================================

export function buildEmptyTransportationVM(
  master: TransportationMaster,
): TransportationVM {
  return {
    companyId: "",
    regions: master.regions.map((group) => ({
      region: group.region,
      regionName: REGION_NAME_BY_CODE[group.region],
      prefectures: group.prefectureCodes.map((prefectureCode) => ({
        prefectureCode,
        prefectureName: PREFECTURE_NAME_BY_CODE[prefectureCode],
        amount: 0,
      })),
    })),
    islandRates: [],
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

function validateRegions(
  regions: TransportationRegionVM[],
): void {
  const seen = new Set<PrefectureCode>();

  for (const region of regions) {
    for (const prefecture of region.prefectures) {
      validateAmount(prefecture.amount);

      if (seen.has(prefecture.prefectureCode)) {
        throw new Error(
          `都道府県が重複しています: ${prefecture.prefectureName}`,
        );
      }

      seen.add(prefecture.prefectureCode);
    }
  }

  if (seen.size !== PREFECTURE_CODES.length) {
    throw new Error("47都道府県すべての送料を設定してください。");
  }

  for (const prefectureCode of PREFECTURE_CODES) {
    if (!seen.has(prefectureCode)) {
      throw new Error(
        `${PREFECTURE_NAME_BY_CODE[prefectureCode]}の送料が設定されていません。`,
      );
    }
  }
}

function validateIslandRates(
  islandRates: TransportationIslandRateVM[],
): void {
  const seen = new Set<string>();

  for (const islandRate of islandRates) {
    if (!islandRate.islandCode) {
      throw new Error("離島コードが設定されていません。");
    }

    validateAmount(islandRate.amount);

    const key =
      `${islandRate.prefectureCode}:${islandRate.islandCode}`;

    if (seen.has(key)) {
      throw new Error(
        `離島送料が重複しています: ${islandRate.islandCode}`,
      );
    }

    seen.add(key);
  }
}

function validateSaveInput(
  input: TransportationSaveInput,
): void {
  validateRegions(input.regions);
  validateIslandRates(input.islandRates);
}

// ============================================================
// Request mapper
// ============================================================

function toTransportationFeeSettingInput(
  input: TransportationSaveInput,
): TransportationFeeSettingInput {
  validateSaveInput(input);

  return {
    prefectureRates: input.regions.flatMap((region) =>
      region.prefectures.map((prefecture) => ({
        prefectureCode: prefecture.prefectureCode,
        amount: prefecture.amount,
      })),
    ),
    islandRates: input.islandRates.map((islandRate) => ({
      islandCode: islandRate.islandCode,
      prefectureCode: islandRate.prefectureCode,
      amount: islandRate.amount,
    })),
  };
}

function toMasterFromRegions(
  regions: TransportationRegionVM[],
): TransportationMaster {
  return {
    regions: regions.map((region) => ({
      region: region.region,
      prefectureCodes: region.prefectures.map(
        (prefecture) => prefecture.prefectureCode,
      ),
    })),
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

export async function createTransportation(
  input: TransportationSaveInput,
): Promise<TransportationVM> {
  const payload = toTransportationFeeSettingInput(input);
  const setting = await createTransportationFeeSettingHTTP(payload);

  return buildTransportationVM(
    toMasterFromRegions(input.regions),
    setting,
  );
}

export async function updateTransportation(
  input: TransportationSaveInput,
): Promise<TransportationVM> {
  const payload = toTransportationFeeSettingInput(input);
  const setting = await updateTransportationFeeSettingHTTP(payload);

  return buildTransportationVM(
    toMasterFromRegions(input.regions),
    setting,
  );
}