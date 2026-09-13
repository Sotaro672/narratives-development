// frontend/console/shell/src/features/company/application/postalCodeSearchService.ts

const ZIP_CLOUD_SEARCH_URL = "https://zipcloud.ibsnet.co.jp/api/search";

export type PostalCodeAddress = {
  state: string;
  city: string;
  street: string;
};

type ZipCloudResult = {
  zipcode: string;
  prefcode: string;
  address1: string;
  address2: string;
  address3: string;
  kana1: string;
  kana2: string;
  kana3: string;
};

type ZipCloudResponse = {
  status: number;
  message: string | null;
  results: ZipCloudResult[] | null;
};

function normalizePostalCode(zipCode: string): string {
  return zipCode.replace(/-/g, "").trim();
}

function isValidPostalCode(zipCode: string): boolean {
  return /^[0-9]{7}$/.test(zipCode);
}

function isZipCloudResponse(value: unknown): value is ZipCloudResponse {
  if (typeof value !== "object" || value === null) {
    return false;
  }

  const response = value as Record<string, unknown>;

  return (
    typeof response.status === "number" &&
    (response.message === null || typeof response.message === "string") &&
    (response.results === null || Array.isArray(response.results))
  );
}

/**
 * 郵便番号から住所を検索する。
 *
 * - 郵便番号は123-4567 / 1234567の両方を受け付ける。
 * - 該当住所が存在しない場合はnullを返す。
 * - APIエラーや通信エラーは例外として呼び出し元へ伝播する。
 *
 * ZipCloudでは同一郵便番号に複数住所が紐づく場合があるため、
 * 自動入力には先頭の検索結果を使用する。
 */
export async function searchAddressByPostalCode(
  zipCode: string,
): Promise<PostalCodeAddress | null> {
  const normalizedZipCode = normalizePostalCode(zipCode);

  if (!isValidPostalCode(normalizedZipCode)) {
    return null;
  }

  const url = new URL(ZIP_CLOUD_SEARCH_URL);
  url.searchParams.set("zipcode", normalizedZipCode);
  url.searchParams.set("limit", "1");

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(
      `郵便番号検索に失敗しました。（HTTP ${response.status}）`,
    );
  }

  const body: unknown = await response.json();

  if (!isZipCloudResponse(body)) {
    throw new Error("郵便番号検索APIから不正なレスポンスが返されました。");
  }

  if (body.status !== 200) {
    throw new Error(
      body.message?.trim() || "郵便番号検索に失敗しました。",
    );
  }

  const result = body.results?.[0];

  if (!result) {
    return null;
  }

  return {
    state: result.address1?.trim() ?? "",
    city: result.address2?.trim() ?? "",
    street: result.address3?.trim() ?? "",
  };
}