// frontend/console/list/src/presentation/hook/useListDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import type { InventoryRow } from "../../../../inventory/src/presentation/components/inventoryCard";

export type InventoryItem = {
  id: string;
  sku: string;
  productName: string;
  color: string;
  size: string;
  stock: number;
};

export type UseListDetailVM = {
  listId: string | null;
  pageTitle: string;

  items: InventoryItem[];
  selected: Record<string, boolean>;
  quantities: Record<string, number>;

  totalSelected: number;
  inventoryRows: InventoryRow[];

  handleToggle: (id: string) => void;
  handleQuantityChange: (id: string, value: string) => void;

  onConfirmSelected: () => void;
  onBack: () => void;

  admin: {
    assigneeName: string;
    createdByName: string;
    createdAt: string;
    onEditAssignee: () => void;
    onClickAssignee: () => void;
  };
};

export function useListDetail(): UseListDetailVM {
  const navigate = useNavigate();
  const { listId } = useParams<{ listId: string }>();

  // モック在庫データ（後でAPI連携に差し替え）
  const [items] = React.useState<InventoryItem[]>([
    {
      id: "inv-001",
      sku: "LUM-SS25-001-BLK-S",
      productName: "シルクブラウス プレミアムライン",
      color: "ブラック",
      size: "S",
      stock: 30,
    },
    {
      id: "inv-002",
      sku: "LUM-SS25-001-BLK-M",
      productName: "シルクブラウス プレミアムライン",
      color: "ブラック",
      size: "M",
      stock: 42,
    },
    {
      id: "inv-003",
      sku: "LUM-SS25-002-IVR-F",
      productName: "リネンシャツ リラックスフィット",
      color: "アイボリー",
      size: "F",
      stock: 18,
    },
  ]);

  // 選択状態（在庫選択カード用）
  const [selected, setSelected] = React.useState<Record<string, boolean>>({});
  const [quantities, setQuantities] = React.useState<Record<string, number>>({});

  const handleToggle = React.useCallback((id: string) => {
    setSelected((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));

    // 初回選択時は数量のデフォルトを 1 にする（既に値があるなら維持）
    setQuantities((prev) =>
      prev[id] != null
        ? prev
        : {
            ...prev,
            [id]: 1,
          },
    );
  }, []);

  const handleQuantityChange = React.useCallback((id: string, value: string) => {
    const num = Number(value);
    if (Number.isNaN(num) || num < 0) return;

    setQuantities((prev) => ({
      ...prev,
      [id]: num,
    }));
  }, []);

  const totalSelected = React.useMemo(() => {
    return items.reduce((sum, item) => {
      if (!selected[item.id]) return sum;
      const q = quantities[item.id] ?? 0;
      return sum + q;
    }, 0);
  }, [items, selected, quantities]);

  // InventoryCard 用 rows: InventoryRow 定義に準拠（閲覧専用表示）
  // - InventoryRow は modelNumber / color が必須
  const inventoryRows: InventoryRow[] = React.useMemo(
    () =>
      items.map((item) => ({
        modelNumber: item.sku, // SKUを型番として扱う
        size: item.size,
        color: item.color,
        stock: item.stock,
      })),
    [items],
  );

  const onConfirmSelected = React.useCallback(() => {
    const picked = items
      .filter((it) => !!selected[it.id])
      .map((it) => ({
        inventoryId: it.id,
        sku: it.sku,
        quantity: quantities[it.id] ?? 0,
        stock: it.stock,
      }));

    // TODO: API連携（確定処理）
    // eslint-disable-next-line no-console
    console.log("選択在庫:", {
      listId,
      picked,
      totalSelected,
      selected,
      quantities,
    });
  }, [items, listId, quantities, selected, totalSelected]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  const pageTitle = React.useMemo(() => {
    return `リスト詳細：${listId ?? "不明ID"}`;
  }, [listId]);

  const admin = React.useMemo(
    () => ({
      assigneeName: "佐藤 美咲",
      createdByName: "山田 太郎",
      createdAt: "2025/10/25 14:30",
      onEditAssignee: () => {
        // eslint-disable-next-line no-console
        console.log("edit assignee");
      },
      onClickAssignee: () => {
        // eslint-disable-next-line no-console
        console.log("assignee clicked");
      },
    }),
    [],
  );

  return {
    listId: listId ?? null,
    pageTitle,

    items,
    selected,
    quantities,

    totalSelected,
    inventoryRows,

    handleToggle,
    handleQuantityChange,

    onConfirmSelected,
    onBack,

    admin,
  };
}
