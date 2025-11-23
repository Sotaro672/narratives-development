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

// ★ 商品設計作成 API 呼び出しサービス
import { createProductBlueprint } from "../../application/productBlueprintCreateService";

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

  // サイズ系
  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (
    id: string,
    patch: Partial<Omit<SizeRow, "id">>,
  ) => void;

  // モデルナンバー系
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  // 担当者系
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
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumber[]>([]);

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
      currentMember.fullName || currentMember.email || currentMember.id;

    setAssigneeId(label);
  }, [currentMember, assigneeId]);

  // ───────────────────────
  // バリデーション
  // ───────────────────────
  const validate = React.useCallback((): string[] => {
    const errors: string[] = [];

    // 必須: 商品名
    if (!productName.trim()) {
      errors.push("商品名は必須です。");
    }

    // 必須: ブランド
    if (!brandId) {
      errors.push("ブランドを選択してください。");
    }

    // 必須: アイテム種別
    if (!itemType) {
      errors.push("アイテム種別を選択してください。");
    }

    // 必須: 商品IDタグ種別
    if (!productIdTagType) {
      errors.push("商品IDタグを選択してください。");
    }

    // カラーバリエーションは1件以上
    if (colors.length === 0) {
      errors.push("カラーバリエーションを1つ以上登録してください。");
    }

    // サイズバリエーションは1件以上
    if (sizes.length === 0) {
      errors.push("サイズバリエーションを1つ以上登録してください。");
    }

    // モデルナンバー: 1件以上 & 空欄禁止
    if (modelNumbers.length === 0) {
      errors.push("モデルナンバーを1つ以上登録してください。");
    } else {
      const hasEmpty = modelNumbers.some((mn) => {
        return Object.values(mn as any).some((v) => {
          if (v == null) return true;
          if (typeof v === "string" && v.trim() === "") return true;
          return false;
        });
      });

      if (hasEmpty) {
        errors.push("モデルナンバー欄に空欄があります。すべて入力してください。");
      }
    }

    return errors;
  }, [
    productName,
    brandId,
    itemType,
    productIdTagType,
    colors,
    sizes,
    modelNumbers,
  ]);

  // ───────────────────────
  // アクション
  // ───────────────────────
  const onCreate = React.useCallback(async () => {
    const errors = validate();
    if (errors.length > 0) {
      alert(`入力内容に不備があります。\n\n- ${errors.join("\n- ")}`);
      console.warn("[useProductBlueprintCreate] validation errors:", errors);
      return;
    }

    const payload = {
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
      companyId: effectiveCompanyId,
      // backend で不要なら無視されるが、デバッグ用に含めておく
      measurementOptions,
    };

    console.log("[useProductBlueprintCreate] onCreate payload snapshot", payload);

    try {
      await createProductBlueprint(payload);
      alert("商品設計を作成しました。");
      navigate(-1);
    } catch (e: any) {
      console.error(
        "[useProductBlueprintCreate] failed to create product blueprint:",
        e,
      );
      alert(
        e instanceof Error
          ? e.message
          : "商品設計の作成に失敗しました。時間をおいて再度お試しください。",
      );
    }
  }, [
    validate,
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

  const onAddSize = React.useCallback(() => {
    setSizes((prev) => [
      ...prev,
      {
        id:
          typeof crypto !== "undefined" && "randomUUID" in crypto
            ? crypto.randomUUID()
            : `size-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        sizeLabel: "",
        chest: undefined,
        waist: undefined,
        length: undefined,
        shoulder: undefined,
      },
    ]);
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      setSizes((prev) =>
        prev.map((s) => (s.id === id ? { ...s, ...patch } : s)),
      );
    },
    [],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const idx = prev.findIndex(
          (m) => m.size === sizeLabel && m.color === color,
        );
        const trimmed = nextCode.trim();

        // 空文字の場合はエントリを削除（バリデーションで拾う）
        if (!trimmed) {
          if (idx === -1) return prev;
          const copy = [...prev];
          copy.splice(idx, 1);
          return copy;
        }

        const next: ModelNumber = {
          size: sizeLabel,
          color,
          code: trimmed,
        };

        if (idx === -1) {
          return [...prev, next];
        }

        const copy = [...prev];
        copy[idx] = next;
        return copy;
      });
    },
    [],
  );

  const onEditAssignee = React.useCallback(() => {
    if (currentMember) {
      const label =
        currentMember.fullName || currentMember.email || currentMember.id;
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

    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,

    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,
  };
}
