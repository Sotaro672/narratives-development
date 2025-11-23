import * as React from "react";
import { useNavigate } from "react-router-dom";

import type {
  ItemType,
  ProductIDTagType,
} from "../../domain/entity/productBlueprint";

// ★ Firestore brands（backend）から取得
import type { Brand } from "../../../../brand/src/domain/entity/brand";
import { brandRepositoryHTTP } from "../../../../brand/src/infrastructure/http/brandRepositoryHTTP";

import type { SizeRow } from "../../../../model/src/presentation/components/SizeVariationCard";
import type { ModelNumber } from "../../../../model/src/presentation/components/ModelNumberCard";

export type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export interface UseProductBlueprintCreateResult {
  // ブランド関連
  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  // 商品設計フィールド
  productName: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];
  productIdTagType: ProductIDTagType;

  // バリエーション
  colorInput: string;
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  onChangeProductName: (v: string) => void;
  onChangeItemType: (v: ItemType) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterial: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeQualityAssurance: (v: string[]) => void;
  onChangeProductIdTagType: (v: ProductIDTagType) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onRemoveSize: (id: string) => void;

  // 管理情報
  assigneeId: string;
  createdBy: string;
  createdAt: string;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onClickCreatedBy: () => void;

  // 画面アクション
  onCreate: () => void;
  onBack: () => void;
}

/**
 * 商品設計作成画面のロジック・状態管理用カスタムフック
 * ページコンポーネント側にはスタイル/構造のみを残す。
 */
export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();

  // ───────────────────────
  // ブランド一覧を取得（Firestore backend）
  // ───────────────────────
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<Brand[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    console.log(
      "[ProductBlueprintCreate] useEffect(start) brand fetch. initial brandId:",
      brandId,
    );

    const loadBrands = async () => {
      setBrandLoading(true);
      setBrandError(null);

      try {
        const filter = { isActive: true as const };
        const page = 1;
        const perPage = 100;

        console.log(
          "[ProductBlueprintCreate] calling brandRepositoryHTTP.list",
          {
            filter,
            page,
            perPage,
          },
        );

        const result = await brandRepositoryHTTP.list({
          filter,
          page,
          perPage,
        });

        if (cancelled) {
          console.log(
            "[ProductBlueprintCreate] brand fetch result ignored (effect cancelled)",
          );
          return;
        }

        console.log(
          "[ProductBlueprintCreate] brandRepositoryHTTP.list result snapshot",
          {
            totalCount: result.totalCount,
            totalPages: result.totalPages,
            page: result.page,
            perPage: result.perPage,
            itemsSample: result.items?.slice(0, 3) ?? [],
          },
        );

        setBrandOptions(result.items ?? []);

        // brandId 未設定なら先頭要素を自動選択
        if (!brandId && result.items && result.items.length > 0) {
          console.log(
            "[ProductBlueprintCreate] brandId is empty. auto-select first brand:",
            result.items[0],
          );
          setBrandId(result.items[0].id);
        }
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e));
        if (!cancelled) {
          setBrandError(err);
        }
        console.error(
          "[ProductBlueprintCreate] failed to load brands via brandRepositoryHTTP.list",
          err,
        );
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
        console.log(
          "[ProductBlueprintCreate] useEffect(end) brand fetch. cancelled:",
          cancelled,
        );
      }
    };

    void loadBrands();

    return () => {
      cancelled = true;
      console.log("[ProductBlueprintCreate] cleanup: cancel brand fetch effect");
    };
    // brandId は初期値ログ用に参照しているだけなので依存に含めない（初回マウント時のみ実行）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 選択中ブランド名
  const brandName = React.useMemo(() => {
    const found = brandOptions.find((b) => b.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  // ログ：brand 状態のスナップショット
  React.useEffect(() => {
    console.log("[ProductBlueprintCreate] state snapshot (brand)", {
      brandId,
      brandName,
      brandOptionsCount: brandOptions.length,
      brandOptionsSample: brandOptions.slice(0, 3),
      brandLoading,
      brandError,
    });
  }, [brandId, brandName, brandOptions, brandLoading, brandError]);

  // ───────────────────────
  // 商品設計フィールド
  // ───────────────────────
  const [productName, setProductName] = React.useState("");
  const [itemType, setItemType] = React.useState<ItemType>("tops");
  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);
  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType>("qr");

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers] = React.useState<ModelNumber[]>([]);

  const [assigneeId, setAssigneeId] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  // 作成処理（ダミー）
  const onCreate = React.useCallback(() => {
    console.log("[ProductBlueprintCreate] onCreate payload snapshot", {
      productName,
      brandId,
      brandName,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTagType,
      colors,
      sizes,
      modelNumbers,
      assigneeId,
      createdBy,
      createdAt,
    });

    alert("商品設計を作成しました（ダミー）");
    navigate(-1);
  }, [
    productName,
    brandId,
    brandName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,
    colors,
    sizes,
    modelNumbers,
    assigneeId,
    createdBy,
    createdAt,
    navigate,
  ]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  // カラー追加・削除
  const onAddColor = React.useCallback(() => {
    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    setColors((prev) => prev.filter((c) => c !== name));
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const onEditAssignee = React.useCallback(() => {
    setAssigneeId("担当者A");
  }, []);

  const onClickAssignee = React.useCallback(() => {
    console.log("assigneeId clicked:", assigneeId);
  }, [assigneeId]);

  const onClickCreatedBy = React.useCallback(() => {
    console.log("createdBy clicked:", createdBy);
  }, [createdBy]);

  return {
    // ブランド
    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId: (id: string) => {
      console.log(
        "[ProductBlueprintCreate] ProductBlueprintCard onChangeBrandId",
        id,
      );
      setBrandId(id);
    },

    // 商品設計フィールド
    productName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,

    // バリエーション
    colorInput,
    colors,
    sizes,
    modelNumbers,

    onChangeProductName: setProductName,
    onChangeItemType: setItemType,
    onChangeFit: setFit,
    onChangeMaterial: setMaterial,
    onChangeWeight: setWeight,
    onChangeQualityAssurance: setQualityAssurance,
    onChangeProductIdTagType: setProductIdTagType,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onRemoveSize,

    // 管理情報
    assigneeId,
    createdBy,
    createdAt,
    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,

    // 画面アクション
    onCreate,
    onBack,
  };
}
