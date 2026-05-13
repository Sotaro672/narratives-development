// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type { Brand } from "../../../../brand/src/domain/entity/brand";
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  isApparelCategoryCode,
  type ApparelMeasurementOption,
  type Fit,
} from "../../domain/entity/apparel";

import type { ProductBlueprintCategorySnapshot } from "../../domain/entity/productBlueprintCategory";

import { createProductBlueprint } from "../../application/productBlueprintCreateService";

import type { ProductBlueprintSizeRow as SizeRow } from "../../infrastructure/api/productBlueprintApi";

export {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/apparel";

export interface UseProductBlueprintCreateResult {
  title: string;

  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  productName: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryLabel: string;
  isApparelCategory: boolean;

  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];

  measurementOptions: ApparelMeasurementOption[];

  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  assigneeId: string;
  assigneeName: string;
  createdBy: string;
  createdAt: string;

  onCreate: () => Promise<void>;
  onBack: () => void;

  onChangeProductName: (v: string) => void;
  onChangeProductBlueprintCategory: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterial: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeQualityAssurance: (v: string[]) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, rgbHex: string) => void;

  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
}

function newSizeRow(): SizeRow {
  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `size-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    sizeLabel: "",

    // tops / dress
    shoulderWidth: undefined,
    bodyWidth: undefined,
    bodyLength: undefined,
    sleeveLength: undefined,
    neckWidth: undefined,

    // bottoms / dress
    waist: undefined,
    hip: undefined,
    rise: undefined,
    inseam: undefined,
    thighWidth: undefined,
    hemWidth: undefined,
    totalLength: undefined,
  };
}

export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();
  const { currentMember, user } = useAuth();

  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );

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
        const items = await fetchAllBrandsForCompany(effectiveCompanyId, true);
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

    return () => {
      cancelled = true;
    };
  }, [effectiveCompanyId]);

  const brandName = React.useMemo(() => {
    const found = brandOptions.find((b) => b.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  const [productName, setProductName] = React.useState("");

  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<ProductBlueprintCategorySnapshot | null>(null);

  const productBlueprintCategoryId = React.useMemo(
    () => productBlueprintCategory?.id ?? "",
    [productBlueprintCategory],
  );

  const productBlueprintCategoryLabel = React.useMemo(() => {
    if (!productBlueprintCategory) {
      return "";
    }

    return (
      productBlueprintCategory.nameJa ||
      productBlueprintCategory.nameEn ||
      productBlueprintCategory.code ||
      productBlueprintCategory.id
    );
  }, [productBlueprintCategory]);

  const isApparelCategory = React.useMemo(() => {
    const code = String(productBlueprintCategory?.code ?? "").trim();
    return isApparelCategoryCode(code);
  }, [productBlueprintCategory]);

  const [fit, setFit] = React.useState<Fit>("" as Fit);
  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    {},
  );

  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumber[]>([]);

  const measurementOptions: ApparelMeasurementOption[] = React.useMemo(() => {
    const code = String(productBlueprintCategory?.code ?? "").trim();

    if (!isApparelCategoryCode(code)) {
      return [];
    }

    return APPAREL_CATEGORY_MEASUREMENT_OPTIONS[code] ?? [];
  }, [productBlueprintCategory]);

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

  const validate = React.useCallback((): string[] => {
    const errors: string[] = [];

    if (!effectiveCompanyId) {
      errors.push("companyId が取得できません。ログインし直してください。");
    }

    if (!productName.trim()) {
      errors.push("商品名は必須です。");
    }

    if (!brandId) {
      errors.push("ブランドを選択してください。");
    }

    if (!productBlueprintCategoryId || !productBlueprintCategory) {
      errors.push("商品カテゴリを選択してください。");
    }

    if (weight < 0) {
      errors.push("重さは 0 以上の値を入力してください。");
    }

    if (isApparelCategory) {
      if (colors.length === 0) {
        errors.push("カラーバリエーションを1つ以上登録してください。");
      }

      if (sizes.length === 0) {
        errors.push("サイズバリエーションを1つ以上登録してください。");
      }

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

        if (hasEmpty) {
          errors.push("モデルナンバー欄に空欄があります。すべて入力してください。");
        }
      }

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

      if (sizes.length > 0) {
        const seenSizes = new Set<string>();
        const dupSizes = new Set<string>();

        sizes.forEach((s) => {
          const labelRaw = s.sizeLabel;
          const label =
            typeof labelRaw === "string"
              ? labelRaw.trim()
              : String(labelRaw ?? "").trim();

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
    }

    return errors;
  }, [
    effectiveCompanyId,
    productName,
    brandId,
    productBlueprintCategoryId,
    productBlueprintCategory,
    weight,
    isApparelCategory,
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

    if (!productBlueprintCategory) {
      alert("商品カテゴリを選択してください。");
      return;
    }

    const apiParams = {
      productName,
      brandId,
      productBlueprintCategoryId: productBlueprintCategory.id,
      productBlueprintCategory,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTag: { type: "qr" as const },
      companyId: effectiveCompanyId,
      colors: isApparelCategory ? colors : [],
      colorRgbMap: isApparelCategory ? colorRgbMap : {},
      sizes: isApparelCategory ? sizes : [],
      modelNumbers: isApparelCategory ? modelNumbers : [],
      assigneeId,
      createdBy: currentMember?.id ?? "",
      categoryFields: null,
    };

    try {
      const created = await createProductBlueprint(apiParams);
      const createdId = String((created as any)?.id ?? "");

      alert("商品設計の作成が完了しました。");

      if (createdId) {
        navigate(`/productBlueprint/detail/${createdId}`);
        return;
      }

      navigate("/productBlueprint");
    } catch (e: any) {
      alert(
        e instanceof Error
          ? e.message
          : "商品設計の作成に失敗しました。時間をおいて再度お試しください。",
      );
      throw e;
    }
  }, [
    validate,
    effectiveCompanyId,
    productName,
    brandId,
    productBlueprintCategory,
    fit,
    material,
    weight,
    qualityAssurance,
    isApparelCategory,
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,
    assigneeId,
    currentMember?.id,
    navigate,
  ]);

  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onChangeProductBlueprintCategory = React.useCallback(
    (category: ProductBlueprintCategorySnapshot | null) => {
      setProductBlueprintCategory(category);

      const code = String(category?.code ?? "").trim();
      const nextIsApparel = isApparelCategoryCode(code);

      if (!nextIsApparel) {
        setColors([]);
        setColorInput("");
        setColorRgbMap({});
        setSizes([]);
        setModelNumbers([]);
      }
    },
    [],
  );

  const onAddColor = React.useCallback(() => {
    if (!isApparelCategory) return;

    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;

    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [isApparelCategory, colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    setColors((prev) => prev.filter((c) => c !== name));

    setColorRgbMap((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });

    setModelNumbers((prev) => prev.filter((mn) => mn.color !== name));
  }, []);

  const onAddSize = React.useCallback(() => {
    if (!isApparelCategory) return;

    setSizes((prev) => [...prev, newSizeRow()]);
  }, [isApparelCategory]);

  const onRemoveSize = React.useCallback(
    (id: string) => {
      const target = sizes.find((s) => s.id === id);
      const labelRaw = target?.sizeLabel;
      const sizeLabel =
        typeof labelRaw === "string"
          ? labelRaw.trim()
          : String(labelRaw ?? "").trim();

      setSizes((prev) => prev.filter((s) => s.id !== id));

      if (sizeLabel) {
        setModelNumbers((prev) => prev.filter((mn) => mn.size !== sizeLabel));
      }
    },
    [sizes],
  );

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      const safePatch: Partial<Omit<SizeRow, "id">> = { ...patch };

      const clampField = (key: keyof Omit<SizeRow, "id">) => {
        const v = safePatch[key];

        if (typeof v === "number") {
          safePatch[key] = (v < 0 ? 0 : v) as never;
        }
      };

      clampField("shoulderWidth");
      clampField("bodyWidth");
      clampField("bodyLength");
      clampField("sleeveLength");
      clampField("neckWidth");

      clampField("waist");
      clampField("hip");
      clampField("rise");
      clampField("inseam");
      clampField("thighWidth");
      clampField("hemWidth");
      clampField("totalLength");

      const prevRow = sizes.find((s) => s.id === id);
      const prevLabelRaw = prevRow?.sizeLabel;
      const prevLabel =
        typeof prevLabelRaw === "string"
          ? prevLabelRaw.trim()
          : String(prevLabelRaw ?? "").trim();

      const nextLabelRaw = safePatch.sizeLabel;
      const nextLabel =
        typeof nextLabelRaw === "string"
          ? nextLabelRaw.trim()
          : nextLabelRaw == null
            ? null
            : String(nextLabelRaw).trim();

      if (nextLabel !== null && nextLabel !== prevLabel) {
        if (!nextLabel) {
          if (prevLabel) {
            setModelNumbers((prev) =>
              prev.filter((mn) => mn.size !== prevLabel),
            );
          }
        } else if (prevLabel) {
          setModelNumbers((prev) =>
            prev.map((mn) =>
              mn.size === prevLabel ? { ...mn, size: nextLabel } : mn,
            ),
          );
        }
      }

      setSizes((prev) =>
        prev.map((s) => (s.id === id ? { ...s, ...safePatch } : s)),
      );
    },
    [sizes],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const idx = prev.findIndex(
          (m) => m.size === sizeLabel && m.color === color,
        );
        const trimmed = nextCode.trim();

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

  const onChangeColorRgb = React.useCallback((name: string, rgbHex: string) => {
    const key = name.trim();
    if (!key) return;

    setColorRgbMap((prev) => ({
      ...prev,
      [key]: rgbHex,
    }));
  }, []);

  const handleChangeWeight = React.useCallback((v: number) => {
    if (Number.isNaN(v)) {
      setWeight(0);
      return;
    }

    setWeight(v < 0 ? 0 : v);
  }, []);

  const onSelectAssignee = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "").trim();
      if (!nextId) return;

      let nextName = "";

      if (currentMember?.id === nextId) {
        nextName =
          currentMember.fullName || currentMember.email || currentMember.id;
      } else {
        nextName = nextId;
      }

      setAssigneeId(nextId);
      setAssigneeName(nextName);
    },
    [currentMember],
  );

  const onEditAssignee = React.useCallback(() => {
    // 担当者選択UIの編集イベント用
  }, []);

  const onClickAssignee = React.useCallback(() => {
    // 担当者選択UIのクリックイベント用
  }, []);

  React.useEffect(() => {
    const validColors = new Set(colors.map((c) => c.trim()).filter(Boolean));
    const validSizes = new Set(
      sizes
        .map((s) => s.sizeLabel)
        .map((v) =>
          typeof v === "string" ? v.trim() : String(v ?? "").trim(),
        )
        .filter(Boolean),
    );

    setModelNumbers((prev) =>
      prev.filter((mn) => validColors.has(mn.color) && validSizes.has(mn.size)),
    );
  }, [colors, sizes]);

  return {
    title: "商品設計を作成",

    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId: (id: string) => setBrandId(id),

    productName,
    productBlueprintCategoryId,
    productBlueprintCategory,
    productBlueprintCategoryLabel,
    isApparelCategory,
    fit,
    material,
    weight,
    qualityAssurance,

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
    onChangeProductBlueprintCategory,
    onChangeFit: setFit,
    onChangeMaterial: setMaterial,
    onChangeWeight: handleChangeWeight,
    onChangeQualityAssurance: setQualityAssurance,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,

    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
  };
}