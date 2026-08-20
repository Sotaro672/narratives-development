// frontend/console/shell/src/features/transportation/infrastracture/transportationApi.ts

import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON } from "../../../shared/http/fetchJSON";
import {
  isPrefectureCode,
  type PrefectureCode,
  type TransportationFeeSetting,
  type TransportationFeeSettingInput,
  type TransportationIslandRate,
  type TransportationMaster,
  type TransportationPrefectureRate,
  type TransportationRegion,
  type TransportationRegionGroup,
} from "../../../shared/types/transporation";

const transportationPath = "/transportation";
const transportationMasterPath = "/transportation/master";

type TransportationPrefectureRateApiResponse = {
  prefectureCode: string;
  amount: number;
};

type TransportationIslandRateApiResponse = {
  islandCode: string;
  prefectureCode: string;
  amount: number;
};

type TransportationFeeSettingApiResponse = {
  companyId: string;
  prefectureRates: TransportationPrefectureRateApiResponse[] | null;
  islandRates: TransportationIslandRateApiResponse[] | null;
  createdAt: string;
  updatedAt: string;
};

type TransportationRegionGroupApiResponse = {
  region: TransportationRegion;
  prefectureCodes: string[] | null;
};

type TransportationMasterApiResponse = {
  regions: TransportationRegionGroupApiResponse[] | null;
};

function toPrefectureCode(value: string): PrefectureCode {
  if (!isPrefectureCode(value)) {
    throw new Error(`invalid_prefecture_code:${value}`);
  }

  return value;
}

function toTransportationPrefectureRate(
  value: TransportationPrefectureRateApiResponse,
): TransportationPrefectureRate {
  return {
    prefectureCode: toPrefectureCode(value.prefectureCode),
    amount: value.amount,
  };
}

function toTransportationIslandRate(
  value: TransportationIslandRateApiResponse,
): TransportationIslandRate {
  return {
    islandCode: value.islandCode,
    prefectureCode: toPrefectureCode(value.prefectureCode),
    amount: value.amount,
  };
}

function toTransportationRegionGroup(
  value: TransportationRegionGroupApiResponse,
): TransportationRegionGroup {
  return {
    region: value.region,
    prefectureCodes: (value.prefectureCodes ?? []).map(toPrefectureCode),
  };
}

function toTransportationFeeSetting(
  value: TransportationFeeSettingApiResponse,
): TransportationFeeSetting {
  return {
    companyId: value.companyId,
    prefectureRates: (value.prefectureRates ?? []).map(
      toTransportationPrefectureRate,
    ),
    islandRates: (value.islandRates ?? []).map(
      toTransportationIslandRate,
    ),
    createdAt: value.createdAt,
    updatedAt: value.updatedAt,
  };
}

function toTransportationMaster(
  value: TransportationMasterApiResponse,
): TransportationMaster {
  return {
    regions: (value.regions ?? []).map(toTransportationRegionGroup),
  };
}

export async function getTransportationFeeSettingHTTP(): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationPath),
    {
      method: "GET",
      auth: "required",
    },
  );

  return toTransportationFeeSetting(response);
}

export async function getTransportationMasterHTTP(): Promise<TransportationMaster> {
  const response = await fetchJSON<TransportationMasterApiResponse>(
    buildConsoleUrl(transportationMasterPath),
    {
      method: "GET",
      auth: "required",
    },
  );

  return toTransportationMaster(response);
}

export async function createTransportationFeeSettingHTTP(
  input: TransportationFeeSettingInput,
): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationPath),
    {
      method: "POST",
      auth: "required",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        prefectureRates: input.prefectureRates,
        islandRates: input.islandRates,
      }),
    },
  );

  return toTransportationFeeSetting(response);
}

export async function updateTransportationFeeSettingHTTP(
  input: TransportationFeeSettingInput,
): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationPath),
    {
      method: "PUT",
      auth: "required",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        prefectureRates: input.prefectureRates,
        islandRates: input.islandRates,
      }),
    },
  );

  return toTransportationFeeSetting(response);
}