// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import {
  formatProductBlueprintDate,
  type SizeRow,
  type ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint, // ★ UPDATE 用
  softDeleteProductBlueprint, // ★ 論理削除用
} from "../../application/productBlueprintDetailService";

import type { Fit, ItemType, WashTagOption } from "../../domain/entity/catalog";

// ★ ModelNumber / SizeVariation の UI ロジックを model 側 hook に委譲
import { useModelCard } from "../../../../model/src/presentation/hook/useModelCard";

// ★ ブランド一覧取得
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

export {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
export type { Fit, WashTagOption } from "../../domain/entity/catalog";

type BrandOption = {
  id: string;
  name: string;
};

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;
  itemType: ItemType | "";
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: ProductIDTagType | "";

  /** ブランド編集用 */
  brandId: string;
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  /** color 名 → rgb hex (#rrggbb) */
  colorRgbMap: Record<string, string>;

  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;
  creator: string;
  createdAt: string;

  onBack: () => void;
  onSave: () => void;
  /** 論理削除（削除ボタン用） */
  onDelete: () => void;

  onChangeProductName: (v: string) => void;
  onChangeItemType: (v: ItemType) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterials: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeWashTags: (v: string[]) => void;
  onChangeProductIdTag: (v: string) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  /** カラーの RGB(hex) 更新用 */
  onChangeColorRgb: (name: string, hex: string) => void;

  /** サイズ削除 */
  onRemoveSize: (id: string) => void;
  /** サイズ追加（SizeVariationCard の「サイズを追加」ボタン用） */
  onAddSize: () => void;
  /** サイズ 1 行分の変更（SizeVariationCard の各セル編集用） */
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  /** モデルナンバー変更（ModelNumberCard のセル編集用） */
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onClickAssignee: () => void;
}

export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  const [pageTitle, setPageTitle] = React.useState<string>("");

  const [productName, setProductName] = React.useState<string>("");
  const [brand, setBrand] = React.useState<string>("");

  const [itemType, setItemType] = React.useState<ItemType | "">("");
  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [materials, setMaterials] = React.useState<string>("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);

  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType | "">("");

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumberRow[]>([]);
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    {},
  );

  const [assignee, setAssignee] = React.useState("担当者未設定");
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");

  // ★ サーバに渡すための ID を保持
  const [brandId, setBrandId] = React.useState<string>("");
  const [assigneeId, setAssigneeId] = React.useState<string>("");

  // ★ ブランド選択用 state
  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [brandLoading, setBrandLoading] = React.useState<boolean>(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  // ---------------------------------
  // service → 詳細データ + variations を反映
  // ---------------------------------
  React.useEffect(() => {
    if (!blueprintId) return;

    (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        const brandNameFromService = (detail as any).brandName as
          | string
          | undefined;
        const assigneeNameFromService = (detail as any).assigneeName as
          | string
          | undefined;
        const createdByNameFromService = (detail as any).createdByName as
          | string
          | undefined;

        const productBlueprintId = detail.id ?? blueprintId;
        const itemTypeFromDetail = detail.itemType as ItemType;

        setPageTitle(detail.productName ?? productBlueprintId);
        setProductName(detail.productName ?? "");

        // brandId / assigneeId を state に保持
        setBrand(brandNameFromService ?? "");
        setBrandId(detail.brandId ?? "");
        setAssigneeId(detail.assigneeId ?? "");

        setItemType(itemTypeFromDetail ?? "");
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const tagType =
          (detail.productIdTag?.type as ProductIDTagType | undefined) ?? "";
        setProductIdTagType(tagType);

        // --------------------------------------------------
        // ブランド一覧を取得（同一 companyId に紐づくもの）
        // --------------------------------------------------
        setBrandLoading(true);
        setBrandError(null);
        try {
          const companyId = detail.companyId ?? "";
          if (companyId) {
            const brands = await fetchAllBrandsForCompany(companyId, false);
            const options: BrandOption[] = brands.map((b: any) => ({
              id: b.id,
              name: b.name,
            }));
            setBrandOptions(options);

            // brandId がセットされている場合、brand ラベルが空ならここで補完
            if (!brandNameFromService && detail.brandId) {
              const found = options.find((o) => o.id === detail.brandId);
              if (found) {
                setBrand(found.name);
              }
            }
          } else {
            setBrandOptions([]);
          }
        } catch (e) {
          console.error(
            "[useProductBlueprintDetail] fetchAllBrandsForCompany failed:",
            e,
          );
          setBrandError(e as Error);
          setBrandOptions([]);
        } finally {
          setBrandLoading(false);
        }

        // --------------------------------------------------
        // ModelVariation 取得
        // --------------------------------------------------
        try {
          const variations =
            await listModelVariationsByProductBlueprintId(
              productBlueprintId,
            );

          const varsAny = variations as any[];

          // -------------------------------
          // colors（Color.Name / color.name）
          // -------------------------------
          const uniqueColors = Array.from(
            new Set(
              varsAny
                .map((v) => {
                  const nm =
                    typeof v.color?.name === "string"
                      ? v.color.name
                      : typeof v.Color?.Name === "string"
                        ? v.Color.Name
                        : "";
                  return nm.trim();
                })
                .filter((c: string) => !!c),
            ),
          );
          setColors(uniqueColors);

          // -------------------------------
          // sizes（Size / size）+ measurements を反映
          // -------------------------------
          const uniqueSizes = Array.from(
            new Set(
              varsAny
                .map((v) => {
                  const sz =
                    typeof v.size === "string"
                      ? v.size
                      : typeof v.Size === "string"
                        ? v.Size
                        : "";
                  return sz.trim();
                })
                .filter((s: string) => !!s),
            ),
          );

          const sizeRows: SizeRow[] = uniqueSizes.map((label, index) => {
            // any ベースで組み立ててから SizeRow にキャストする
            const base: any = {
              id: String(index + 1),
              sizeLabel: label,
            };

            // 該当サイズの最初の variation
            const found = varsAny.find((v) => {
              const sz =
                typeof v.size === "string"
                  ? v.size
                  : typeof v.Size === "string"
                    ? v.Size
                    : "";
              return sz.trim() === label;
            });

            const ms: Record<string, number | null> | undefined =
              found?.measurements ?? found?.Measurements;

            if (ms && typeof ms === "object") {
              if (itemTypeFromDetail === "ボトムス") {
                // ボトムス用: Firestore の日本語キー → base.* にマッピング
                base.waist = ms["ウエスト"] ?? undefined;
                base.hip = ms["ヒップ"] ?? undefined;
                base.rise = ms["股上"] ?? undefined;
                base.inseam = ms["股下"] ?? undefined;

                const thighVal = ms["わたり幅"] ?? undefined;
                if (thighVal != null) {
                  // 正規フィールド + alias の両方に入れておく
                  base.thigh = thighVal;
                }

                base.hemWidth = ms["裾幅"] ?? undefined;
              } else {
                // デフォルト（トップス）
                const lenVal = ms["着丈"] ?? undefined;
                if (lenVal != null) {
                  base.length = lenVal;
                }

                const chestVal = ms["胸囲"] ?? undefined;
                if (chestVal != null) {
                  base.chest = chestVal; // 正規フィールド
                }
                const widthVal = ms["身幅"] ?? undefined;
                if (widthVal != null) {
                  base.width = widthVal; // 正規フィールド
                }
                const shoulderVal = ms["肩幅"] ?? undefined;
                if (shoulderVal != null) {
                  base.shoulder = shoulderVal;
                }

                const sleeveVal = ms["袖丈"] ?? undefined;
                if (sleeveVal != null) {
                  base.sleeveLength = sleeveVal;
                }
              }
            }

            return base as SizeRow;
          });

          setSizes(sizeRows);

          // -------------------------------
          // modelNumbers（ModelNumber / modelNumber）
          // -------------------------------
          const modelNumberRows: ModelNumberRow[] = varsAny.map((v) => {
            const size =
              (typeof v.size === "string"
                ? v.size
                : (v.Size as string | undefined)) ?? "";

            const color =
              (typeof v.color?.name === "string"
                ? v.color.name
                : (v.Color?.Name as string | undefined)) ?? "";

            const code =
              (typeof v.modelNumber === "string"
                ? v.modelNumber
                : (v.ModelNumber as string | undefined)) ?? "";

            return { size, color, code } as ModelNumberRow;
          });
          setModelNumbers(modelNumberRows);

          // -------------------------------
          // colorRgbMap（rgb int → #rrggbb）
          // -------------------------------
          const rgbMap: Record<string, string> = {};
          varsAny.forEach((v) => {
            const name =
              (typeof v.color?.name === "string"
                ? v.color.name
                : (v.Color?.Name as string | undefined)) ?? "";

            const rgbVal =
              typeof v.color?.rgb === "number"
                ? v.color.rgb
                : typeof v.Color?.RGB === "number"
                  ? v.Color.RGB
                  : undefined;

            if (name && typeof rgbVal === "number") {
              const hex =
                "#" +
                (rgbVal >>> 0).toString(16).padStart(6, "0").toLowerCase();
              rgbMap[name] = hex;
            }
          });
          setColorRgbMap(rgbMap);
        } catch (e) {
          console.error(
            "[useProductBlueprintDetail] listModelVariationsByProductBlueprintId failed:",
            e,
          );
          setColors([]);
          setSizes([]);
          setModelNumbers([]);
          setColorRgbMap({});
        }

        // assignee
        setAssignee(
          assigneeNameFromService ?? detail.assigneeId ?? "担当者未設定",
        );

        // creator
        setCreator(
          createdByNameFromService ?? detail.createdBy ?? "作成者未設定",
        );

        setCreatedAt(formatProductBlueprintDate(detail.createdAt) || "");
      } catch (e) {
        console.error("[useProductBlueprintDetail] fetch failed:", e);
      }
    })();
  }, [blueprintId]);

  // ---------------------------------
  // モデルナンバー変更（アプリケーション状態の更新専用ロジック）
  // ---------------------------------
  const handleChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const idx = prev.findIndex(
          (m) => m.size === sizeLabel && m.color === color,
        );
        const trimmed = nextCode.trim();

        // 空文字の場合はエントリを削除（必要に応じてバリデーションで拾う）
        if (!trimmed) {
          if (idx === -1) return prev;
          const copy = [...prev];
          copy.splice(idx, 1);
          return copy;
        }

        const next: ModelNumberRow = {
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

  // ---------------------------------
  // ModelNumberCard 用ロジックは useModelCard に委譲
  // ---------------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes,
    colors,
    modelNumbers: modelNumbers as any, // 形は ModelNumberRow と互換
    colorRgbMap,
  });

  // ---------------------------------
  // Handlers: 保存
  // ---------------------------------
  const onSave = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計ID が不明です");
      return;
    }

    // itemType が未選択の場合は一旦ガード
    if (!itemType) {
      alert("アイテム種別を選択してください");
      return;
    }

    // ★ モデルナンバーのバリデーション
    // サイズラベル & カラーの組み合わせで、どこか 1 つでも modelNumber が空ならエラー
    const hasEmptyModelNumber = sizes.some((s) => {
      const sizeLabel = (s.sizeLabel ?? "").trim();
      if (!sizeLabel) return false; // サイズ未設定行は無視

      return colors.some((c) => {
        const color = (c ?? "").trim();
        if (!color) return false; // 空のカラー名は無視

        const code = getCode(sizeLabel, color);
        return !code || !code.trim();
      });
    });

    if (hasEmptyModelNumber) {
      alert("モデルナンバーが空欄です");
      return;
    }

    (async () => {
      try {
        // ★ 保存ボタン押下時のスナップショットを出力（編集後の状態）
        console.log("[useProductBlueprintDetail] onSave payload snapshot", {
          blueprintId,
          productName,
          itemType,
          fit,
          materials,
          weight,
          washTags,
          productIdTagType,
          sizes,
          modelNumbers,
          colorRgbMap,
          brandId,
          assigneeId,
        });

        await updateProductBlueprint({
          id: blueprintId,
          productName,
          itemType: itemType as ItemType,
          fit,
          material: materials,
          weight,
          qualityAssurance: washTags,
          productIdTag: productIdTagType
            ? { type: productIdTagType }
            : undefined,
          sizes,
          modelNumbers,
          colorRgbMap,
          // ★ サーバ側で空にされないよう brandId / assigneeId も送る
          brandId,
          assigneeId,
        } as any);

        alert("保存しました");
      } catch (e) {
        console.error("[useProductBlueprintDetail] update failed:", e);
        alert("保存に失敗しました");
      }
    })();
  }, [
    blueprintId,
    productName,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTagType,
    sizes,
    modelNumbers,
    colorRgbMap,
    brandId,
    assigneeId,
    colors,
    getCode,
  ]);

  // ---------------------------------
  // Handlers: 論理削除
  // ---------------------------------
  const onDelete = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計ID が不明です");
      return;
    }

    const ok = window.confirm(
      "この商品設計を削除しますか？\n関連するモデルバリエーションも含めて論理削除されます。",
    );
    if (!ok) return;

    (async () => {
      try {
        await softDeleteProductBlueprint(blueprintId);
        alert("削除しました");
        // 一覧へ戻る
        navigate("/productBlueprint");
      } catch (e) {
        console.error("[useProductBlueprintDetail] delete failed:", e);
        alert("削除に失敗しました");
      }
    })();
  }, [blueprintId, navigate]);

  // ★ 戻るボタン: 相対 -1 ではなく、商品設計一覧の絶対パスへ
  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onAddColor = React.useCallback(() => {
    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    const key = name.trim();
    if (!key) return;

    // colors から削除
    setColors((prev) => prev.filter((c) => c !== key));

    // 対応する RGB も削除しておく
    setColorRgbMap((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });

    // この color に紐づく modelNumber も削除
    setModelNumbers((prevMN) => prevMN.filter((m) => m.color !== key));
  }, []);

  // ★ カラーの RGB(hex) 更新
  const onChangeColorRgb = React.useCallback((name: string, hex: string) => {
    const colorName = name.trim();
    let value = hex.trim();
    if (!colorName || !value) return;

    // 「#」が無かったら付ける
    if (!value.startsWith("#")) {
      value = `#${value}`;
    }

    setColorRgbMap((prev) => ({
      ...prev,
      [colorName]: value,
    }));
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => {
      const target = prev.find((s) => s.id === id);
      const next = prev.filter((s) => s.id !== id);

      if (target) {
        const sizeLabel = (target.sizeLabel ?? "").trim();
        if (sizeLabel) {
          // この size に紐づく modelNumber も削除
          setModelNumbers((prevMN) =>
            prevMN.filter((m) => m.size !== sizeLabel),
          );
        }
      }

      return next;
    });
  }, []);

  // ★ サイズ追加: 新しい空行を 1 行追加
  const onAddSize = React.useCallback(() => {
    setSizes((prev) => {
      // 既存 id の最大値 + 1 を採番（数字でない id があっても安全側に倒す）
      const nextNum =
        prev.reduce((max, row) => {
          const n = Number(row.id);
          if (Number.isNaN(n)) return max;
          return n > max ? n : max;
        }, 0) + 1;

      const next: SizeRow = {
        id: String(nextNum),
        sizeLabel: "",
        // 他のフィールド(length, chest, ...) は undefined のままで OK
      } as SizeRow;

      return [...prev, next];
    });
  }, []);

  // ★ サイズ 1 行分の変更
  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      // 数値項目は 0 未満にならないようにクランプ（create と同じノリ）
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

  const onClickAssignee = React.useCallback(() => {
    console.log("assignee clicked:", assignee);
  }, [assignee]);

  // ★ ブランド変更ハンドラ（id と表示名の両方を更新）
  const onChangeBrandId = React.useCallback(
    (id: string) => {
      setBrandId(id);
      const found = brandOptions.find((b) => b.id === id);
      if (found) {
        setBrand(found.name);
      }
    },
    [brandOptions],
  );

  return {
    pageTitle,

    productName,
    brand,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTag: productIdTagType || "",

    // ブランド編集用
    brandId,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,

    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    getCode,

    assignee,
    creator,
    createdAt,

    onBack,
    onSave,
    onDelete,

    onChangeProductName: setProductName,
    onChangeItemType: (v: ItemType) => setItemType(v),
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,
    onChangeProductIdTag: (v: string) =>
      setProductIdTagType(v as ProductIDTagType),

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onRemoveSize,
    onAddSize,
    onChangeSize,

    // ★ UI からの onChange:
    //   - useModelCard 側の codeMap を更新
    //   - application state (modelNumbers) も更新
    onChangeModelNumber: (sizeLabel, color, nextCode) => {
      uiOnChangeModelNumber(sizeLabel, color, nextCode);
      handleChangeModelNumber(sizeLabel, color, nextCode);
    },

    onClickAssignee,
  };
}
