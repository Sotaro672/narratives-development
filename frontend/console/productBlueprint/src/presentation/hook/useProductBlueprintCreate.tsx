// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";

import type { ProductIDTagType } from "../../domain/entity/productBlueprint";
import type { Brand } from "../../../../brand/src/domain/entity/brand";
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

import type { SizeRow } from "../../../../model/src/presentation/hook/useModelCard";
import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  ITEM_TYPE_MEASUREMENT_OPTIONS,
} from "../../domain/entity/catalog";
import type {
  Fit,
  ItemType,
  MeasurementOption,
} from "../../domain/entity/catalog";

import { createProductBlueprint } from "../../application/productBlueprintCreateService";

export {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  ITEM_TYPE_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  ITEM_TYPE_MEASUREMENT_OPTIONS,
} from "../../domain/entity/catalog";

export interface UseProductBlueprintCreateResult {
  title: string;

  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  productName: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];
  productIdTagType: ProductIDTagType;

  measurementOptions: MeasurementOption[];

  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  assigneeId: string;
  assigneeName: string;
  createdBy: string;
  createdAt: string;

  onCreate: () => void;
  onBack: () => void;

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
  onChangeColorRgb: (name: string, rgbHex: string) => void;

  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (
    id: string,
    patch: Partial<Omit<SizeRow, "id">>,
  ) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onEditAssignee: () => void;
  onClickAssignee: () => void;
}

export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();
  const { currentMember, user } = useAuth();

  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );

  // ブランド
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<Brand[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      if (!effectiveCompanyId) {
        setBrandOptions([]);
        return;
      }

      setBrandLoading(true);
      setBrandError(null);

      try {
        const items = await fetchAllBrandsForCompany(
          effectiveCompanyId,
          true
        );
        if (!cancelled) {
          setBrandOptions(items);
        }
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e));
        if (!cancelled) {
          setBrandError(err);
        }
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    }

    void loadBrands();
    return () => { cancelled = true };
  }, [effectiveCompanyId]);

  const brandName = React.useMemo(() => {
    const found = brandOptions.find((b) => b.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  // 商品フィールド
  const [productName, setProductName] = React.useState("");
  const [itemType, setItemType] = React.useState<ItemType>("" as ItemType);
  const [fit, setFit] = React.useState<Fit>("" as Fit);
  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);
  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType>("" as ProductIDTagType);

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    {},
  );

  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumber[]>([]);

  const measurementOptions: MeasurementOption[] = React.useMemo(() => {
    if (!itemType) return [];
    return ITEM_TYPE_MEASUREMENT_OPTIONS[itemType] ?? [];
  }, [itemType]);

  const [assigneeId, setAssigneeId] = React.useState("");
  const [assigneeName, setAssigneeName] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  React.useEffect(() => {
    if (!currentMember) return;
    if (assigneeId) return;

    const memberId = currentMember.id;
    const label =
      currentMember.fullName || currentMember.email || currentMember.id;

    setAssigneeId(memberId);
    setAssigneeName(label);
  }, [currentMember, assigneeId]);

  // バリデーション
  const validate = React.useCallback((): string[] => {
    const errors: string[] = [];

    if (!effectiveCompanyId)
      errors.push("companyId が取得できません。ログインし直してください。");

    if (!productName.trim()) errors.push("商品名は必須です。");
    if (!brandId) errors.push("ブランドを選択してください。");
    if (!itemType) errors.push("アイテム種別を選択してください。");
    if (!productIdTagType)
      errors.push("商品IDタグを選択してください。");
    if (weight < 0)
      errors.push("重さは 0 以上の値を入力してください。");
    if (colors.length === 0)
      errors.push("カラーバリエーションを1つ以上登録してください。");
    if (sizes.length === 0)
      errors.push("サイズバリエーションを1つ以上登録してください。");

    // モデルナンバー必須 & 空欄チェック
    if (modelNumbers.length === 0) {
      errors.push("モデルナンバーを1つ以上登録してください。");
    } else {
      const hasEmpty = modelNumbers.some((mn) =>
        Object.values(mn).some((v) => {
          if (v == null) return true;
          if (typeof v === "string" && v.trim() === "") return true;
          return false;
        }),
      );
      if (hasEmpty)
        errors.push("モデルナンバー欄に空欄があります。すべて入力してください。");
    }

    // ✅ モデルナンバーの重複チェック（code 単位でユニーク）
    if (modelNumbers.length > 0) {
      const seenCodes = new Set<string>();
      const dupCodes = new Set<string>();

      modelNumbers.forEach((mn) => {
        const code = mn.code?.trim();
        if (!code) return;
        if (seenCodes.has(code)) {
          dupCodes.add(code);
        } else {
          seenCodes.add(code);
        }
      });

      if (dupCodes.size > 0) {
        errors.push(
          `モデルナンバーが重複しています。（重複コード: ${Array.from(
            dupCodes,
          ).join("、")}）`,
        );
      }
    }

    // ✅ サイズ名（sizeLabel）の重複チェック
    if (sizes.length > 0) {
      const seenSizes = new Set<string>();
      const dupSizes = new Set<string>();

      sizes.forEach((s) => {
        const labelRaw = (s as any).sizeLabel;
        const label =
          typeof labelRaw === "string" ? labelRaw.trim() : String(labelRaw ?? "").trim();
        if (!label) return;
        if (seenSizes.has(label)) {
          dupSizes.add(label);
        } else {
          seenSizes.add(label);
        }
      });

      if (dupSizes.size > 0) {
        errors.push(
          `サイズ名が重複しています。（重複サイズ: ${Array.from(
            dupSizes,
          ).join("、")}）`,
        );
      }
    }

    // ✅ 採寸値: 空欄もエラーにする
    const invalidMeasurementFields: string[] = [];
    sizes.forEach((s, index) => {
      const label = (s as any).sizeLabel || `#${index + 1}`;
      Object.entries(s as Record<string, unknown>).forEach(([key, value]) => {
        if (key === "id" || key === "sizeLabel") return;

        // 空欄（null / undefined / 空文字）も NG
        if (
          value == null ||
          (typeof value === "string" && value.trim() === "")
        ) {
          invalidMeasurementFields.push(`サイズ ${label} の「${key}」`);
          return;
        }

        if (typeof value !== "number" || Number.isNaN(value)) {
          invalidMeasurementFields.push(`サイズ ${label} の「${key}」`);
        }
      });
    });

    if (invalidMeasurementFields.length > 0) {
      errors.push(
        `採寸欄には空欄がないようにし、数値のみ入力してください。（問題のある項目: ${invalidMeasurementFields.join(
          "、",
        )}）`,
      );
    }

    return errors;
  }, [
    effectiveCompanyId,
    productName,
    brandId,
    itemType,
    productIdTagType,
    weight,
    colors,
    sizes,
    modelNumbers,
  ]);

  const onCreate = React.useCallback(async () => {
    const errors = validate();
    if (errors.length > 0) {
      alert(`入力内容に不備があります。\n\n- ${errors.join("\n- ")}`);
      return;
    }

    if (!effectiveCompanyId) {
      alert("companyId が取得できません。ログインし直してください。");
      return;
    }

    const productIdTag = { type: productIdTagType };

    const apiParams = {
      productName,
      brandId,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTag,
      companyId: effectiveCompanyId,
      colors,
      colorRgbMap,
      sizes,
      modelNumbers,
      assigneeId,
      createdBy: currentMember?.id ?? "",
    };

    try {
      await createProductBlueprint(apiParams);
      alert("商品設計を作成しました。");
      navigate("/productBlueprint");
    } catch (e: any) {
      alert(
        e instanceof Error
          ? e.message
          : "商品設計の作成に失敗しました。時間をおいて再度お試しください。",
      );
    }
  }, [
    validate,
    effectiveCompanyId,
    productName,
    brandId,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,
    assigneeId,
    currentMember?.id,
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
    setColorRgbMap((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
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
      } as any,
    ]);
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      const safePatch: Partial<Omit<SizeRow, "id">> = { ...patch };

      const clampField = (key: keyof Omit<SizeRow, "id">) => {
        const v = safePatch[key];
        if (typeof v === "number") {
          safePatch[key] = (v < 0 ? 0 : v) as any;
        }
      };

      clampField("chest");
      clampField("waist");
      clampField("length");
      clampField("shoulder");

      setSizes((prev) =>
        prev.map((s) => (s.id === id ? { ...s, ...safePatch } : s)),
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

        // 空文字の場合はエントリ削除（上のバリデーションで拾う）
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

  const onChangeColorRgb = React.useCallback(
    (name: string, rgbHex: string) => {
      const key = name.trim();
      if (!key) return;
      setColorRgbMap((prev) => ({
        ...prev,
        [key]: rgbHex,
      }));
    },
    [],
  );

  const handleChangeWeight = React.useCallback((v: number) => {
    if (Number.isNaN(v)) {
      setWeight(0);
      return;
    }
    setWeight(v < 0 ? 0 : v);
  }, []);

  const onEditAssignee = React.useCallback(() => {
    if (currentMember) {
      const memberId = currentMember.id;
      const label =
        currentMember.fullName || currentMember.email || currentMember.id;
      setAssigneeId(memberId);
      setAssigneeName(label);
    }
  }, [currentMember]);

  const onClickAssignee = React.useCallback(() => {
    // クリック自体のハンドリングのみ（ログ出力なし）
  }, [assigneeId, assigneeName]);

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
    colorRgbMap,
    sizes,
    modelNumbers,

    assigneeId,
    assigneeName,
    createdBy,
    createdAt,

    onCreate,
    onBack,

    onChangeProductName: setProductName,
    onChangeItemType: setItemType,
    onChangeFit: setFit,
    onChangeMaterial: setMaterial,
    onChangeWeight: handleChangeWeight,
    onChangeQualityAssurance: setQualityAssurance,
    onChangeProductIdTagType: setProductIdTagType,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,

    onEditAssignee,
    onClickAssignee,
  };
}
