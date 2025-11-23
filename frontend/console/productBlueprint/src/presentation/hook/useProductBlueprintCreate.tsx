// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

// ProductIDTagType だけ productBlueprint のエンティティから使う
import type { ProductIDTagType } from "../../domain/entity/productBlueprint";

// Brand (domain)
import type { Brand } from "../../../../brand/src/domain/entity/brand";
// ★ companyId フィルタ付きの安全なクエリを利用
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

// Size / ModelNumber の型だけ借りる
import type { SizeRow } from "../../../../model/src/presentation/components/SizeVariationCard";
import type { ModelNumber } from "../../../../model/src/presentation/components/ModelNumberCard";

// Auth / currentMember
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// catalog.ts から ItemType / Fit / measurement 系を集約して利用
import {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  ITEM_TYPE_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  ITEM_TYPE_MEASUREMENT_OPTIONS,
} from "../../domain/entity/catalog";
import type {
  Fit,
  ItemType,
  MeasurementOption,
} from "../../domain/entity/catalog";

// 他プレゼン層からも使いやすいように再エクスポート
export {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  ITEM_TYPE_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  ITEM_TYPE_MEASUREMENT_OPTIONS,
} from "../../domain/entity/catalog";

// -------------------------------
// Hook が外に公開する ViewModel
// -------------------------------
export interface UseProductBlueprintCreateResult {
  // Meta
  title: string;

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

  // アイテム種別から導出された採寸項目
  measurementOptions: MeasurementOption[];

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  assigneeId: string;
  createdBy: string;
  createdAt: string;

  // 画面全体アクション
  onCreate: () => void;
  onBack: () => void;

  // 入力変更ハンドラ
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

  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onClickCreatedBy: () => void;
}

/**
 * 商品設計作成画面用のロジック・状態をまとめたカスタムフック
 */
export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();

  // Auth / currentMember から companyId を取得
  const { currentMember, user } = useAuth();
  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );

  // ───────────────────────
  // ブランド一覧（companyId で必ず絞る）
  // ───────────────────────
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<Brand[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      if (!effectiveCompanyId) {
        console.log(
          "[useProductBlueprintCreate] companyId is empty; skip brand fetch.",
        );
        setBrandOptions([]);
        return;
      }

      console.log(
        "[useProductBlueprintCreate] start fetchAllBrandsForCompany",
        { companyId: effectiveCompanyId },
      );

      setBrandLoading(true);
      setBrandError(null);

      try {
        const items = await fetchAllBrandsForCompany(
          effectiveCompanyId,
          true, // isActiveOnly
        );
        if (cancelled) {
          console.log(
            "[useProductBlueprintCreate] brand fetch result ignored (cancelled)",
          );
          return;
        }

        console.log(
          "[useProductBlueprintCreate] fetchAllBrandsForCompany result",
          {
            count: items.length,
            sample: items.slice(0, 3),
          },
        );

        setBrandOptions(items);
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e));
        if (!cancelled) {
          setBrandError(err);
        }
        console.error(
          "[useProductBlueprintCreate] failed to fetch brands for company:",
          err,
        );
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, [effectiveCompanyId]);

  const brandName = React.useMemo(() => {
    const found = brandOptions.find((b) => b.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  // ───────────────────────
  // 商品設計フィールド
  // ───────────────────────
  const [productName, setProductName] = React.useState("");

  // アイテム種別は空（未選択）から
  const [itemType, setItemType] = React.useState<ItemType>("" as ItemType);

  // フィットは空（未選択）から
  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);

  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType>("" as ProductIDTagType);

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers] = React.useState<ModelNumber[]>([]);

  // ───────────────────────
  // アイテム種別 → 採寸項目
  // ───────────────────────
  const measurementOptions: MeasurementOption[] = React.useMemo(() => {
    if (!itemType) return [];
    // ★ ここで catalog.ts の ItemType と完全に一致していれば 7053 は発生しない
    return ITEM_TYPE_MEASUREMENT_OPTIONS[itemType] ?? [];
  }, [itemType]);

  // ───────────────────────
  // 管理情報
  // ───────────────────────
  const [assigneeId, setAssigneeId] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  React.useEffect(() => {
    if (!currentMember) return;
    if (assigneeId) return;

    const label =
      currentMember.fullName ||
      currentMember.email ||
      currentMember.id;

    setAssigneeId(label);
  }, [currentMember, assigneeId]);

  // ───────────────────────
  // アクション
  // ───────────────────────
  const onCreate = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] onCreate payload snapshot", {
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
      companyId: effectiveCompanyId,
      measurementOptions,
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
    effectiveCompanyId,
    measurementOptions,
    navigate,
  ]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

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
    if (currentMember) {
      const label =
        currentMember.fullName ||
        currentMember.email ||
        currentMember.id;
      setAssigneeId(label);
    }
  }, [currentMember]);

  const onClickAssignee = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] assigneeId clicked:", assigneeId);
  }, [assigneeId]);

  const onClickCreatedBy = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] createdBy clicked:", createdBy);
  }, [createdBy]);

  // -------------------------------
  // 返却する ViewModel
  // -------------------------------
  return {
    title: "商品設計を作成",

    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId: (id: string) => setBrandId(id),

    productName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,

    measurementOptions,

    colors,
    colorInput,
    sizes,
    modelNumbers,

    assigneeId,
    createdBy,
    createdAt,

    onCreate,
    onBack,

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

    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,
  };
}
