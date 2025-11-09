// frontend/productBlueprint/src/presentation/pages/productBlueprintManagement.tsx

import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { PRODUCT_BLUEPRINTS } from "../../infrastructure/mockdata/mockdata";
import type {
  ProductBlueprint,
  ItemType,
} from "../../../../shell/src/shared/types/productBlueprint";

// "YYYY/MM/DD" → timestamp
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

// 一覧表示用のUI行モデル（ドメインとは分離）
type UiRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string; // YYYY/MM/DD
  lastModifiedAt: string; // YYYY/MM/DD
};

type SortKey = "createdAt" | "lastModifiedAt" | null;

// ISO8601 → "YYYY/MM/DD"（壊れてたらそのまま返す）
const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// BrandID → 表示名（モック用マッピング）
const brandLabelFromId = (brandId: string): string => {
  switch (brandId) {
    case "brand_lumina":
      return "LUMINA Fashion";
    case "brand_nexus":
      return "NEXUS Street";
    default:
      return brandId || "-";
  }
};

// AssigneeID → 表示名（モックなのでIDそのまま or 簡易変換）
const assigneeLabelFromId = (assigneeId: string): string =>
  assigneeId || "-";

export default function ProductBlueprintManagement() {
  const navigate = useNavigate();

  // フィルタ & ソート状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<SortKey>(null);
  const [sortedDir, setSortedDir] = useState<"asc" | "desc" | null>(null);

  // ProductBlueprint → UiRow へ変換＋フィルタ＋ソート
  const rows: UiRow[] = useMemo(() => {
    const all: UiRow[] = (PRODUCT_BLUEPRINTS as ProductBlueprint[]).map(
      (pb) => ({
        id: pb.id,
        productName: pb.productName,
        brandLabel: brandLabelFromId(pb.brandId),
        assigneeLabel: assigneeLabelFromId(pb.assigneeId),
        tagLabel: pb.productIdTag?.type
          ? pb.productIdTag.type.toUpperCase()
          : "-",
        createdAt: toDisplayDate(pb.createdAt),
        lastModifiedAt: toDisplayDate(pb.lastModifiedAt),
      })
    );

    let work = all;

    if (brandFilter.length > 0) {
      work = work.filter((r) => brandFilter.includes(r.brandLabel));
    }

    if (sortedKey && sortedDir) {
      work = [...work].sort((a, b) => {
        const av = toTs(a[sortedKey]);
        const bv = toTs(b[sortedKey]);
        return sortedDir === "asc" ? av - bv : bv - av;
      });
    }

    return work;
  }, [brandFilter, sortedKey, sortedDir]);

  // ヘッダー定義
  const headers: React.ReactNode[] = [
    "プロダクト",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={[
        { value: "LUMINA Fashion", label: "LUMINA Fashion" },
        { value: "NEXUS Street", label: "NEXUS Street" },
      ]}
      selected={brandFilter}
      onChange={(values: string[]) => setBrandFilter(values)}
    />,
    "担当者",
    "タグ種別",
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortedKey}
      direction={sortedDir}
      onChange={(key, dir) => {
        setSortedKey(key as SortKey);
        setSortedDir(dir);
      }}
    />,
    <SortableTableHeader
      key="lastModifiedAt"
      label="最終更新日"
      sortKey="lastModifiedAt"
      activeKey={sortedKey}
      direction={sortedDir}
      onChange={(key, dir) => {
        setSortedKey(key as SortKey);
        setSortedDir(dir);
      }}
    />,
  ];

  // 行クリックで詳細へ
  const handleRowClick = (r: UiRow) => {
    navigate(`/productBlueprint/detail/${encodeURIComponent(r.id)}`);
  };

  // 作成ボタン
  const handleCreate = () => {
    navigate("/productBlueprint/create");
  };

  return (
    <List
      title="商品設計"
      headerCells={headers}
      showCreateButton
      createLabel="商品設計を作成"
      onCreate={handleCreate}
      showResetButton
      onReset={() => {
        setBrandFilter([]);
        setSortedKey(null);
        setSortedDir(null);
      }}
    >
      {rows.map((r) => (
        <tr
          key={r.id}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          onClick={() => handleRowClick(r)}
        >
          <td>{r.productName}</td>
          <td>
            <span className="lp-brand-pill">{r.brandLabel}</span>
          </td>
          <td>{r.assigneeLabel}</td>
          <td>{r.tagLabel}</td>
          <td>{r.createdAt}</td>
          <td>{r.lastModifiedAt}</td>
        </tr>
      ))}
    </List>
  );
}
