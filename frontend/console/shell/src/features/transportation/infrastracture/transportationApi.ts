// frontend/console/shell/src/features/transportation/infrastracture/transportationApi.ts

import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON } from "../../../shared/http/fetchJSON";
import {
  isPrefectureCode,
  isTransportationRegion,
  type IslandCode,
  type PrefectureCode,
  type TransportationFeeSetting,
  type TransportationFeeSettingInput,
  type TransportationIslandDefinition,
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
  id: string;
  companyId: string;
  name: string;
  prefectureRates: TransportationPrefectureRateApiResponse[] | null;
  islandRates: TransportationIslandRateApiResponse[] | null;
  createdAt: string;
  updatedAt: string;
};

type TransportationRegionGroupApiResponse = {
  region: string;
  prefectureCodes: string[] | null;
  islandCodes: string[] | null;
};

type TransportationIslandDefinitionApiResponse = {
  islandCode: string;
  prefectureCode: string;
  displayName: string;
};

type TransportationMasterApiResponse = {
  regions: TransportationRegionGroupApiResponse[] | null;
  islands: TransportationIslandDefinitionApiResponse[] | null;
};

function toPrefectureCode(value: string): PrefectureCode {
  if (!isPrefectureCode(value)) {
    throw new Error(`invalid_prefecture_code:${value}`);
  }
  return value;
}

function toTransportationRegion(value: string): TransportationRegion {
  if (!isTransportationRegion(value)) {
    throw new Error(`invalid_transportation_region:${value}`);
  }
  return value;
}

function toIslandCode(value: string): IslandCode {
  if (!value) {
    throw new Error("invalid_island_code");
  }
  return value;
}

function toTransportationPrefectureRate(value: TransportationPrefectureRateApiResponse): TransportationPrefectureRate {
  return {
    prefectureCode: toPrefectureCode(value.prefectureCode),
    amount: value.amount,
  };
}

function toTransportationIslandRate(value: TransportationIslandRateApiResponse): TransportationIslandRate {
  return {
    islandCode: toIslandCode(value.islandCode),
    prefectureCode: toPrefectureCode(value.prefectureCode),
    amount: value.amount,
  };
}

function toTransportationRegionGroup(value: TransportationRegionGroupApiResponse): TransportationRegionGroup {
  return {
    region: toTransportationRegion(value.region),
    prefectureCodes: (value.prefectureCodes ?? []).map(toPrefectureCode),
    islandCodes: (value.islandCodes ?? []).map(toIslandCode),
  };
}

function toTransportationIslandDefinition(value: TransportationIslandDefinitionApiResponse): TransportationIslandDefinition {
  return {
    islandCode: toIslandCode(value.islandCode),
    prefectureCode: toPrefectureCode(value.prefectureCode),
    displayName: value.displayName,
  };
}

function toTransportationFeeSetting(value: TransportationFeeSettingApiResponse): TransportationFeeSetting {
  if (!value.id) {
    throw new Error("invalid_transportation_id");
  }
  if (!value.companyId) {
    throw new Error("invalid_transportation_company_id");
  }
  if (!value.name) {
    throw new Error("invalid_transportation_name");
  }

  return {
    id: value.id,
    companyId: value.companyId,
    name: value.name,
    prefectureRates: (value.prefectureRates ?? []).map(toTransportationPrefectureRate),
    islandRates: (value.islandRates ?? []).map(toTransportationIslandRate),
    createdAt: value.createdAt,
    updatedAt: value.updatedAt,
  };
}

function toTransportationMaster(value: TransportationMasterApiResponse): TransportationMaster {
  return {
    regions: (value.regions ?? []).map(toTransportationRegionGroup),
    islands: (value.islands ?? []).map(toTransportationIslandDefinition),
  };
}

function transportationDetailPath(transportationId: string): string {
  if (!transportationId) {
    throw new Error("transportationId is required");
  }
  return `${transportationPath}/${encodeURIComponent(transportationId)}`;
}

export async function listTransportationFeeSettingsHTTP(): Promise<TransportationFeeSetting[]> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse[]>(
    buildConsoleUrl(transportationPath),
    {
      method: "GET",
      auth: "required",
    },
  );

  return (response ?? []).map(toTransportationFeeSetting);
}

export async function getTransportationFeeSettingHTTP(transportationId: string): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationDetailPath(transportationId)),
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

export async function createTransportationFeeSettingHTTP(input: TransportationFeeSettingInput): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationPath),
    {
      method: "POST",
      auth: "required",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: input.name,
        prefectureRates: input.prefectureRates,
        islandRates: input.islandRates,
      }),
    },
  );

  return toTransportationFeeSetting(response);
}

export async function updateTransportationFeeSettingHTTP(
  transportationId: string,
  input: TransportationFeeSettingInput,
): Promise<TransportationFeeSetting> {
  const response = await fetchJSON<TransportationFeeSettingApiResponse>(
    buildConsoleUrl(transportationDetailPath(transportationId)),
    {
      method: "PUT",
      auth: "required",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: input.name,
        prefectureRates: input.prefectureRates,
        islandRates: input.islandRates,
      }),
    },
  );

  return toTransportationFeeSetting(response);
}