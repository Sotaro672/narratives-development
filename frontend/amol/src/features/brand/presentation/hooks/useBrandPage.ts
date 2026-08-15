// frontend/amol/src/features/brand/presentation/hooks/useBrandPage.ts

import { useCallback, useEffect, useRef, useState } from "react";
import { loadBrandPage } from "../../application/loadBrandPage";
import type { BrandDetail, BrandListItem } from "../../../shared/types/brand";

type BrandPageIdleState = {
  status: "idle";
  brand: null;
  listItems: BrandListItem[];
  error: "";
};

type BrandPageLoadingState = {
  status: "loading";
  brand: null;
  listItems: BrandListItem[];
  error: "";
};

type BrandPageSuccessState = {
  status: "success";
  brand: BrandDetail;
  listItems: BrandListItem[];
  error: "";
};

type BrandPageErrorState = {
  status: "error";
  brand: null;
  listItems: BrandListItem[];
  error: string;
};

type BrandPageState =
  | BrandPageIdleState
  | BrandPageLoadingState
  | BrandPageSuccessState
  | BrandPageErrorState;

const initialState: BrandPageState = {
  status: "idle",
  brand: null,
  listItems: [],
  error: "",
};

function getErrorMessage(caught: unknown): string {
  if (caught instanceof Error && caught.message.trim()) {
    return caught.message;
  }

  return "ブランド情報の取得に失敗しました。";
}

export function useBrandPage(brandId: string) {
  const normalizedBrandId = brandId.trim();
  const [state, setState] = useState<BrandPageState>(initialState);
  const mountedRef = useRef(false);
  const requestIdRef = useRef(0);

  const reload = useCallback(async (): Promise<void> => {
    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;

    if (!normalizedBrandId) {
      if (mountedRef.current && requestId === requestIdRef.current) {
        setState({
          status: "error",
          brand: null,
          listItems: [],
          error: "brandIdが指定されていません。",
        });
      }

      return;
    }

    if (mountedRef.current) {
      setState({
        status: "loading",
        brand: null,
        listItems: [],
        error: "",
      });
    }

    try {
      const result = await loadBrandPage(normalizedBrandId);

      if (!mountedRef.current || requestId !== requestIdRef.current) {
        return;
      }

      setState({
        status: "success",
        brand: result.brand,
        listItems: result.listItems,
        error: "",
      });
    } catch (caught) {
      if (!mountedRef.current || requestId !== requestIdRef.current) {
        return;
      }

      setState({
        status: "error",
        brand: null,
        listItems: [],
        error: getErrorMessage(caught),
      });
    }
  }, [normalizedBrandId]);

  useEffect(() => {
    mountedRef.current = true;
    void reload();

    return () => {
      mountedRef.current = false;
      requestIdRef.current += 1;
    };
  }, [reload]);

  return {
    brand: state.brand,
    listItems: state.listItems,
    loading: state.status === "idle" || state.status === "loading",
    error: state.error,
    reload,
  };
}