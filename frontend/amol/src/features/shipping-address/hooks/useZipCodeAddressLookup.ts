//frontend\amol\src\features\shipping-address\hooks\useZipCodeAddressLookup.ts
import { useEffect, useRef, useState } from "react";

import type { ShippingAddressFormValues } from "../types";
import {
  formatZipCode,
  isZipCloudResponse,
  normalizeZipCode,
} from "../utils/zipCode";

type UseZipCodeAddressLookupInput = {
  zipCode: string;
  setForm: React.Dispatch<React.SetStateAction<ShippingAddressFormValues>>;
};

export function useZipCodeAddressLookup({
  zipCode,
  setForm,
}: UseZipCodeAddressLookupInput) {
  const lastLookedUpZipCodeRef = useRef("");

  const [isLookingUpAddress, setIsLookingUpAddress] = useState(false);
  const [zipCodeError, setZipCodeError] = useState("");

  useEffect(() => {
    const normalizedZipCode = normalizeZipCode(zipCode);

    if (normalizedZipCode.length !== 7) {
      setZipCodeError("");
      return;
    }

    if (lastLookedUpZipCodeRef.current === normalizedZipCode) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      const lookupAddress = async () => {
        try {
          setIsLookingUpAddress(true);
          setZipCodeError("");
          lastLookedUpZipCodeRef.current = normalizedZipCode;

          const response = await fetch(
            `https://zipcloud.ibsnet.co.jp/api/search?zipcode=${normalizedZipCode}`
          );

          const responseBody: unknown = await response.json();

          if (!response.ok || !isZipCloudResponse(responseBody)) {
            throw new Error("住所の取得に失敗しました。");
          }

          if (responseBody.status !== 200) {
            throw new Error(responseBody.message || "住所の取得に失敗しました。");
          }

          const address = responseBody.results?.[0];

          if (!address) {
            setZipCodeError("該当する住所が見つかりませんでした。");
            return;
          }

          setForm((current) => ({
            ...current,
            zipCode: formatZipCode(normalizedZipCode),
            state: address.address1,
            city: address.address2,
            street: address.address3,
          }));
        } catch (error) {
          console.error(error);

          if (error instanceof Error) {
            setZipCodeError(error.message);
          } else {
            setZipCodeError("住所の取得に失敗しました。");
          }
        } finally {
          setIsLookingUpAddress(false);
        }
      };

      void lookupAddress();
    }, 400);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [zipCode, setForm]);

  const resetZipCodeLookup = () => {
    lastLookedUpZipCodeRef.current = "";
    setZipCodeError("");
  };

  return {
    isLookingUpAddress,
    zipCodeError,
    resetZipCodeLookup,
  };
}