// frontend/console/production/src/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ currentMember.fullName 取得など（将来の表示用）
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ Production / ProductBlueprint 詳細取得サービス
import {
  loadProductionDetail,
  loadProductBlueprintDetail,
  type ProductionDetail,
  type ProductBlueprintDetail,
} from "../../application/productionDetailService";

// ★ ProductionQuantityCard 用の行型（Create 画面と共通）
import type { ProductionQuantityRow } from "../../application/productionCreateService";

type Mode = "view" | "edit";

/**
 * Production 詳細画面用 Hook
 *
 * - URL パラメータの productionId を保持
 * - edit / view モードの出し分け機能を提供（デフォルトは view）
 * - 戻るボタン用の onBack を提供
 * - loadProductionDetail を使って Production 詳細を取得
 * - production.productBlueprintId を使って ProductBlueprint 詳細も取得
 * - production.models から ProductionQuantityRow[] を生成して画面へ渡す
 */
export function useProductionDetail() {
  const navigate = useNavigate();
  const { productionId } = useParams<{ productionId: string }>();

  // ==========================
  // currentMember 情報（表示などに使える）
  // ==========================
  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";

  // ==========================
  // モード管理（view / edit）
  // ==========================
  const [mode, setMode] = React.useState<Mode>("view"); // ★ デフォルト view

  const isViewMode = mode === "view";
  const isEditMode = mode === "edit";

  const switchToView = React.useCallback(() => setMode("view"), []);
  const switchToEdit = React.useCallback(() => setMode("edit"), []);
  const toggleMode = React.useCallback(
    () => setMode((prev) => (prev === "view" ? "edit" : "view")),
    [],
  );

  // ==========================
  // Production 詳細データ
  // ==========================
  const [production, setProduction] = React.useState<ProductionDetail | null>(
    null,
  );
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // ==========================
  // ProductBlueprint 詳細データ
  // ==========================
  const [productBlueprint, setProductBlueprint] =
    React.useState<ProductBlueprintDetail | null>(null);
  const [pbLoading, setPbLoading] = React.useState(false);
  const [pbError, setPbError] = React.useState<string | null>(null);

  // ==========================
  // ProductionQuantityCard 用 rows
  // ==========================
  const [quantityRows, setQuantityRows] = React.useState<
    ProductionQuantityRow[]
  >([]);

  // --------------------------
  // Production 詳細取得
  // --------------------------
  React.useEffect(() => {
    if (!productionId) return;

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);
        setProductBlueprint(null); // Production 変更時は一旦リセット
        setPbError(null);

        const data = await loadProductionDetail(productionId);
        if (cancelled) return;

        console.log("[useProductionDetail] loaded production detail:", data);
        setProduction(data);
      } catch (e) {
        console.error("[useProductionDetail] failed to load:", e);
        if (!cancelled) {
          setError("生産情報の取得に失敗しました");
          setProduction(null);
          setQuantityRows([]);
          setProductBlueprint(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [productionId]);

  // --------------------------
  // production.models → quantityRows へマッピング
  //   ※ ProductionQuantityRow は Create 画面と共通の型:
  //      - modelVariationId
  //      - modelCode
  //      - size
  //      - colorName
  //      - colorCode
  //      - stock
  //
  //   ここでは、それらに加えて
  //      - modelNumber
  //      - color
  //      - rgb
  //      - quantity
  //   などの追加プロパティも持たせておく（型的には許容される）。
  // --------------------------
  React.useEffect(() => {
    if (!production || !Array.isArray((production as any).models)) {
      setQuantityRows([]);
      return;
    }

    const rawModels = (production as any).models as any[];

    const rows: ProductionQuantityRow[] = rawModels.map((m: any, index) => {
      // モデルID（modelVariationId 相当）
      const modelVariationId =
        m.modelVariationId ??
        m.ModelVariationID ??
        m.modelId ??
        m.ModelID ??
        m.id ??
        m.ID ??
        `${index}`;

      // 型番（modelNumber → modelCode に流し込む）
      const modelNumber =
        m.modelNumber ??
        m.ModelNumber ??
        m.modelCode ??
        m.ModelCode ??
        "";

      // サイズ
      const size = m.size ?? m.Size ?? "";

      // カラー名 / コード
      const colorName = m.colorName ?? m.ColorName ?? "";
      const colorCode = m.colorCode ?? m.ColorCode ?? "";
      const color = colorName || colorCode;

      // RGB（あれば拾う）
      const rgb =
        m.rgb ??
        m.Rgb ??
        m.RGB ??
        m.colorRgb ??
        m.ColorRgb ??
        m.ColorRGB ??
        null;

      // 数量
      const quantityRaw =
        m.quantity ?? m.Quantity ?? m.stock ?? m.Stock ?? 0;
      const quantity = Number.isFinite(Number(quantityRaw))
        ? Math.max(0, Math.floor(Number(quantityRaw)))
        : 0;

      const row: ProductionQuantityRow & {
        // 追加情報（ProductionQuantityCard 側で使いたくなった時用）
        modelNumber?: string;
        color?: string;
        rgb?: number | string | null;
        quantity?: number;
        id?: string;
      } = {
        // ★ ProductionQuantityRow で必須なフィールド
        modelVariationId,
        modelCode: modelNumber,
        size,
        colorName,
        colorCode,
        stock: quantity,

        // ★ 追加情報（型には無いが、オブジェクトとしては保持）
        id: modelVariationId,
        modelNumber,
        color,
        rgb,
        quantity,
      };

      return row;
    });

    console.log("[useProductionDetail] mapped quantityRows:", rows);
    setQuantityRows(rows);
  }, [production]);

  // --------------------------
  // productBlueprintId → ProductBlueprint 詳細取得
  // --------------------------
  React.useEffect(() => {
    const blueprintId = production?.productBlueprintId;
    if (!blueprintId) {
      setProductBlueprint(null);
      setPbError(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setPbLoading(true);
        setPbError(null);

        const pb = await loadProductBlueprintDetail(blueprintId);
        if (cancelled) return;

        console.log(
          "[useProductionDetail] loaded productBlueprint detail:",
          pb,
        );
        setProductBlueprint(pb);
      } catch (e) {
        console.error(
          "[useProductionDetail] failed to load productBlueprint:",
          e,
        );
        if (!cancelled) {
          setPbError("商品設計情報の取得に失敗しました");
          setProductBlueprint(null);
        }
      } finally {
        if (!cancelled) {
          setPbLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // ==========================
  // 戻る
  // ==========================
  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  return {
    // モード関連
    mode,
    isViewMode,
    isEditMode,
    switchToView,
    switchToEdit,
    toggleMode,

    // 画面制御
    onBack: handleBack,

    // Production 詳細データ関連
    productionId: productionId ?? null,
    production,
    setProduction,
    loading,
    error,

    // ProductBlueprint 詳細データ関連
    productBlueprint,
    pbLoading,
    pbError,

    // ProductionQuantityCard 用
    quantityRows,
    setQuantityRows,

    // 参考情報（ヘッダなどで使用可）
    creator,
  };
}
