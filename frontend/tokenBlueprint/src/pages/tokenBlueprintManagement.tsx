// frontend/tokenBlueprint/src/pages/tokenBlueprintManagement.tsx
import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type TokenBlueprint = {
  name: string;
  symbol: string;
  brand: string;
  manager: string;
  createdAt: string; // YYYY/M/D
};

const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  {
    name: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    createdAt: "2024/1/20",
  },
  {
    name: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    createdAt: "2024/1/18",
  },
  {
    name: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    manager: "山田 太郎",
    createdAt: "2024/1/15",
  },
  {
    name: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    manager: "佐藤 美咲",
    createdAt: "2024/1/12",
  },
  {
    name: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    manager: "高橋 健太",
    createdAt: "2024/1/10",
  },
];

const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function TokenBlueprintManagementPage() {
  const navigate = useNavigate();

  // フィルタ状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] = useState<"createdAt" | null>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // オプション
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const managerOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.manager))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // フィルタ + ソート
  const rows = useMemo(() => {
    let data = TOKEN_BLUEPRINTS.filter(
      (r) =>
        (brandFilter.length === 0 || brandFilter.includes(r.brand)) &&
        (managerFilter.length === 0 || managerFilter.includes(r.manager))
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const av = toTs(a.createdAt);
        const bv = toTs(b.createdAt);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [brandFilter, managerFilter, sortKey, sortDir]);

  // 行クリックで詳細へ遷移（symbol を ID として使用）
  const goDetail = (symbol: string) => {
    navigate(`/tokenBlueprint/${encodeURIComponent(symbol)}`);
  };

  const headers: React.ReactNode[] = [
    "トークン名",
    "シンボル",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={(vals: string[]) => setBrandFilter(vals)}
    />,
    <FilterableTableHeader
      key="manager"
      label="担当者"
      options={managerOptions}
      selected={managerFilter}
      onChange={(vals: string[]) => setManagerFilter(vals)}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "createdAt");
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計"
        headerCells={headers}
        showCreateButton
        createLabel="トークン設計を作成"
        showResetButton
        // ▼ 作成ボタン押下で作成ページへ遷移
        onCreate={() => navigate("/tokenBlueprint/create")}
        onReset={() => {
          setBrandFilter([]);
          setManagerFilter([]);
          setSortKey(null);
          setSortDir(null);
          console.log("リスト更新");
        }}
      >
        {rows.map((t) => (
          <tr
            key={`${t.symbol}-${t.createdAt}`}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(t.symbol)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(t.symbol);
              }
            }}
          >
            <td>{t.name}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brand}</span>
            </td>
            <td>{t.manager}</td>
            <td>{t.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
