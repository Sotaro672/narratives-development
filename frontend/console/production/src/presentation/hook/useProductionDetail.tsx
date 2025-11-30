// frontend/console/production/src/presentation/hook/useProductionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ currentMember.fullName 取得など（将来の表示用）
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ Production / ProductBlueprint / ModelVariation 詳細取得サービス
import {
  loadProductionDetail,
  loadProductBlueprintDetail,
  loadModelVariationIndexByProductBlueprintId,
  buildQuantityRowsFromModels,
  type ProductionDetail,
  type ProductBlueprintDetail,
  type ModelVariationSummary,
  type ProductionQuantityRow as DetailQuantityRow,
} from "../../application/productionDetailService";

// ★ ProductionQuantityCard 用の行型（Create 画面と共通）
import type { ProductionQuantityRow as CreateQuantityRow } from "../../application/productionCreateService";

type Mode = "view" | "edit";

/**
 * Production 詳細画面用 Hook
 *
 * - URL パラメータの productionId を保持
 * - edit / view モードの出し分け機能を提供（デフォルトは view）
 * - 戻るボタン用の onBack を提供
 * - loadProductionDetail を使って Production 詳細を取得
 * - production.productBlueprintId を使って ProductBlueprint 詳細も取得
 * - productBlueprintId から ModelVariation 一覧を取得して index 化
 * - production.models と modelIndex を突き合わせて ProductionQuantityRow[] を生成
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
  // ModelVariation index
  //   - key: variationId
  //   - value: ModelVariationSummary（modelNumber / size / color / rgb）
  // ==========================
  const [modelIndex, setModelIndex] = React.useState<
    Record<string, ModelVariationSummary>
  >({});

  // ==========================
  // ProductionQuantityCard 用 rows
  //   - 型は Create 画面と共通（ProductionQuantityCard の props に合わせる）
  // ==========================
  const [quantityRows, setQuantityRows] = React.useState<CreateQuantityRow[]>(
    [],
  );

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
        setModelIndex({});
        setQuantityRows([]);

        const data = await loadProductionDetail(productionId);
        if (cancelled) return;

        setProduction(data);
      } catch (e) {
        if (!cancelled) {
          setError("生産情報の取得に失敗しました");
          setProduction(null);
          setQuantityRows([]);
          setProductBlueprint(null);
          setModelIndex({});
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

        setProductBlueprint(pb);
      } catch (e) {
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

  // --------------------------
  // productBlueprintId → ModelVariation 一覧取得
  //   - /models/by-blueprint/{productBlueprintId}/variations を叩いて index 化
  // --------------------------
  React.useEffect(() => {
    const blueprintId = production?.productBlueprintId;
    if (!blueprintId) {
      setModelIndex({});
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const index = await loadModelVariationIndexByProductBlueprintId(
          blueprintId,
        );
        if (cancelled) return;

        setModelIndex(index);
      } catch (e) {
        if (!cancelled) {
          setModelIndex({});
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [production?.productBlueprintId]);

  // --------------------------
  // production.models × modelIndex → quantityRows へマッピング
  //
  // 1) buildQuantityRowsFromModels で
  //    - id
  //    - modelNumber
  //    - size
  //    - color
  //    - rgb
  //    - quantity
  //    の入った DetailQuantityRow[] を作成
  //
  // 2) ProductionQuantityCard が期待する CreateQuantityRow 形式:
  //    - modelVariationId
  //    - modelCode
  //    - size
  //    - colorName
  //    - colorCode
  //    - quantity
  //    に変換しつつ、追加情報（rgb / quantity / modelNumber など）も
  //    オブジェクトには保持しておく。
  // --------------------------
  React.useEffect(() => {
    if (!production || !Array.isArray((production as any).models)) {
      setQuantityRows([]);
      return;
    }

    const rawModels = (production as any).models as any[];

    // ★ modelIndex を使って id → modelNumber / size / color / rgb を解決
    const detailRows: DetailQuantityRow[] = buildQuantityRowsFromModels(
      rawModels,
      modelIndex,
    );

    const mapped: CreateQuantityRow[] = detailRows.map((row) => {
      const quantity = row.quantity ?? 0;

      const createRow: CreateQuantityRow & {
        // 追加情報（今後カード側で使いたくなるかもしれないので保持）
        id?: string;
        modelNumber?: string;
        color?: string;
        rgb?: number | string | null;
        quantity?: number;
      } = {
        // ProductionQuantityCard が必須としているフィールド
        modelVariationId: row.id,
        modelCode: row.modelNumber,
        size: row.size,
        colorName: row.color,
        colorCode: "", // 現状は色情報は name のみなので空
        quantity: quantity,

        // 追加情報
        id: row.id,
        modelNumber: row.modelNumber,
        color: row.color,
        rgb: row.rgb ?? null,
      };

      return createRow;
    });

    setQuantityRows(mapped);
  }, [production, modelIndex]);

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
