// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintCard.tsx

import * as React from "react";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

import type {
  TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlers,
} from "../components/tokenBlueprintCard";

// Service（アプリケーションロジック）
import {
  loadBrandsForCompany,
  resolveBrandName,
} from "../../application/tokenBlueprintCreateService";

/**
 * TokenBlueprintCard 用のロジックフック
 * - UI 状態管理のみ担当
 */
export function useTokenBlueprintCard(params: {
  initialTokenBlueprint?: Partial<TokenBlueprint> & { brandName?: string };
  initialBurnAt?: string;
  initialIconUrl?: string;
  initialEditMode?: boolean;
}) {
  const tb = params.initialTokenBlueprint ?? {};

  // -------------------------
  // Local UI states
  // -------------------------
  const [id, setId] = React.useState(tb.id ?? "");
  const [name, setName] = React.useState(tb.name ?? "");
  const [symbol, setSymbol] = React.useState(tb.symbol ?? "");

  const [brandId, setBrandId] = React.useState(tb.brandId ?? "");
  const [brandName, setBrandName] = React.useState(
    (tb as any).brandName ?? "",
  );

  const [description, setDescription] = React.useState(tb.description ?? "");
  const [burnAt, setBurnAt] = React.useState(params.initialBurnAt ?? "");
  const [iconUrl] = React.useState(params.initialIconUrl ?? "");

  // ⭐ 編集モード切り替え可能に変更
  const [isEditMode, setIsEditMode] = React.useState(
    params.initialEditMode ?? false,
  );

  const [brandOptions, setBrandOptions] = React.useState<
    { id: string; name: string }[]
  >([]);

  // リセット用に「バックエンドからもらった元データ」を保持
  const initialRef = React.useRef<
    (Partial<TokenBlueprint> & { brandName?: string }) | null
  >(tb);

  // -------------------------
  // Brand 一覧読み込み（Service に委譲）
  // -------------------------
  React.useEffect(() => {
    let cancelled = false;

    loadBrandsForCompany().then((brands) => {
      if (!cancelled) setBrandOptions(brands);
    });

    return () => {
      cancelled = true;
    };
  }, []);

  // backend から initialTokenBlueprint が更新されたら（= 詳細取得完了したら）state に反映
  React.useEffect(() => {
    const src = params.initialTokenBlueprint;
    if (!src) return;

    // リセット用に保持
    initialRef.current = src;

    // 編集中はユーザー入力を壊さない
    if (isEditMode) return;

    setId(src.id ?? "");
    setName(src.name ?? "");
    setSymbol(src.symbol ?? "");
    setBrandId(src.brandId ?? "");
    setBrandName((src as any).brandName ?? "");
    setDescription(src.description ?? "");
    // burnAt / iconUrl は今のところ別初期値を優先
  }, [params.initialTokenBlueprint, isEditMode]);

  // brandId しか無い場合、brandName を backend から解決
  React.useEffect(() => {
    let cancelled = false;

    if (!brandId || brandName) return;

    resolveBrandName(brandId).then((name) => {
      if (!cancelled && name) setBrandName(name);
    });

    return () => {
      cancelled = true;
    };
  }, [brandId, brandName]);

  // -------------------------
  // ViewModel
  // -------------------------
  const vm: TokenBlueprintCardViewModel = {
    id,
    name,
    symbol,
    brandId,
    brandName,
    description,
    iconUrl,
    isEditMode,
    brandOptions,
  };

  // -------------------------
  // Handlers（UI のみ）
  // -------------------------
  const handlers: TokenBlueprintCardHandlers = {
    onChangeName: (v) => setName(v),
    onChangeSymbol: (v) => setSymbol(v.toUpperCase()),

    onChangeBrand: (id, name) => {
      setBrandId(id);
      setBrandName(name);
    },

    onChangeDescription: (v) => setDescription(v),

    onUploadIcon: () => {
      if (!isEditMode) return;
      alert("トークンアイコンのアップロード（モック）");
    },

    onPreview: () => {
      alert("プレビュー画面を開きます（モック）");
    },

    // ⭐ 既存：トグル
    onToggleEditMode: () => {
      setIsEditMode((prev) => !prev);
    },

    // ⭐ 追加：外側から直接モード指定
    setEditMode: (edit: boolean) => {
      setIsEditMode(edit);
    },

    // ⭐ 追加：元データにリセット
    reset: () => {
      const src = initialRef.current;
      if (!src) return;

      setId(src.id ?? "");
      setName(src.name ?? "");
      setSymbol(src.symbol ?? "");
      setBrandId(src.brandId ?? "");
      setBrandName((src as any).brandName ?? "");
      setDescription(src.description ?? "");
      // burnAt は今のところそのまま
    },
  };

  return { vm, handlers };
}
