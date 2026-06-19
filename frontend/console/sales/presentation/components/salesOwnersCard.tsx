// frontend/console/sales/src/presentation/components/salesOwnersCard.tsx
import { useEffect, useMemo, useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";
import { Checkbox } from "../../../shell/src/shared/ui/checkbox";
import AvatarIcon from "../../../shell/src/shared/ui/icon";
import Pagination from "../../../shell/src/shared/ui/pagination";
import FilterableTableHeader from "../../../shell/src/shared/ui/filterable-table-header";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../shell/src/shared/ui/table";
import { SortableTableHeader } from "../../../shell/src/layout/List/List";

export type SalesOwnerItem = {
  avatarId?: string;
  avatarName?: string;
  avatarIconUrl?: string;
  mintAddress?: string;
  productName?: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
};

type Props = {
  owners?: SalesOwnerItem[];
};

type SortKey = "followerCount" | "postCount";
type SortDir = "asc" | "desc";

const PAGE_SIZE = 10;

function compareNumbers(a: number, b: number): number {
  return a - b;
}

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function ownerKey(owner: SalesOwnerItem, index: number): string {
  const avatarId = String(owner.avatarId ?? "").trim();
  const mintAddress = String(owner.mintAddress ?? "").trim();
  const productName = String(owner.productName ?? "").trim();

  return avatarId || mintAddress || productName || `empty-${index}`;
}

function sortOwners(
  owners: SalesOwnerItem[],
  sortKey: SortKey,
  sortDir: SortDir,
): SalesOwnerItem[] {
  const next = [...owners];

  next.sort((a, b) => {
    let result = 0;

    switch (sortKey) {
      case "followerCount":
        result = compareNumbers(
          toSafeNumber(a.followerCount),
          toSafeNumber(b.followerCount),
        );
        break;
      case "postCount":
        result = compareNumbers(
          toSafeNumber(a.postCount),
          toSafeNumber(b.postCount),
        );
        break;
      default:
        result = 0;
        break;
    }

    return sortDir === "asc" ? result : -result;
  });

  return next;
}

function uniqueStrings(values: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = String(value ?? "").trim();
    if (!normalized) continue;
    if (seen.has(normalized)) continue;
    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

export default function SalesOwnersCard({ owners = [] }: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("followerCount");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [selectedOwnerKeys, setSelectedOwnerKeys] = useState<string[]>([]);
  const [selectedProductNames, setSelectedProductNames] = useState<string[]>(
    [],
  );
  const [currentPage, setCurrentPage] = useState(1);

  const productFilterOptions = useMemo(() => {
    const names = owners.map((owner) => String(owner.productName ?? "").trim());
    return uniqueStrings(names).map((name) => ({
      value: name,
      label: name,
    }));
  }, [owners]);

  const filteredOwners = useMemo(() => {
    if (selectedProductNames.length === 0) {
      return owners;
    }

    return owners.filter((owner) => {
      const productName = String(owner.productName ?? "").trim();
      return selectedProductNames.includes(productName);
    });
  }, [owners, selectedProductNames]);

  const sortedOwners = useMemo(() => {
    return sortOwners(filteredOwners, sortKey, sortDir);
  }, [filteredOwners, sortDir, sortKey]);

  const totalPages = useMemo(() => {
    if (sortedOwners.length === 0) return 1;
    return Math.ceil(sortedOwners.length / PAGE_SIZE);
  }, [sortedOwners]);

  const pagedOwners = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    const end = start + PAGE_SIZE;
    return sortedOwners.slice(start, end);
  }, [currentPage, sortedOwners]);

  const allOwnerKeys = useMemo(() => {
    return pagedOwners.map((owner, index) =>
      ownerKey(owner, (currentPage - 1) * PAGE_SIZE + index),
    );
  }, [currentPage, pagedOwners]);

  const isAllSelected =
    allOwnerKeys.length > 0 &&
    allOwnerKeys.every((key) => selectedOwnerKeys.includes(key));

  useEffect(() => {
    setSelectedOwnerKeys((prev) => {
      const validKeys = sortedOwners.map((owner, index) =>
        ownerKey(owner, index),
      );
      return prev.filter((key) => validKeys.includes(key));
    });
  }, [sortedOwners]);

  useEffect(() => {
    setCurrentPage(1);
  }, [selectedProductNames, sortKey, sortDir]);

  useEffect(() => {
    if (currentPage > totalPages) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  const handleChangeSort = (nextKey: string) => {
    const normalizedKey: SortKey =
      nextKey === "postCount" ? "postCount" : "followerCount";

    setSortKey((prevKey) => {
      if (prevKey === normalizedKey) {
        setSortDir((prevDir) => (prevDir === "asc" ? "desc" : "asc"));
        return prevKey;
      }

      setSortDir("desc");
      return normalizedKey;
    });
  };

  const handleToggleAll = (checked: boolean) => {
    setSelectedOwnerKeys((prev) => {
      if (checked) {
        return uniqueStrings([...prev, ...allOwnerKeys]);
      }
      return prev.filter((key) => !allOwnerKeys.includes(key));
    });
  };

  const handleToggleRow = (key: string, checked: boolean) => {
    setSelectedOwnerKeys((prev) => {
      if (checked) {
        if (prev.includes(key)) return prev;
        return [...prev, key];
      }
      return prev.filter((item) => item !== key);
    });
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-3">
          <CardTitle>所有者一覧</CardTitle>
        </div>
      </CardHeader>

      <CardContent>
        {owners.length === 0 ? (
          <p className="text-sm text-slate-500">
            表示可能なオーナーがありません。
          </p>
        ) : (
          <div className="space-y-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">
                    <Checkbox
                      id="sales-owners-select-all"
                      checked={isAllSelected}
                      onCheckedChange={handleToggleAll}
                    />
                  </TableHead>
                  <TableHead>アバター</TableHead>
                  <TableHead>
                    <FilterableTableHeader
                      label="商品名"
                      options={productFilterOptions}
                      selected={selectedProductNames}
                      onChange={setSelectedProductNames}
                      dialogTitle="商品名で絞り込み"
                    />
                  </TableHead>
                  <TableHead>
                    <SortableTableHeader
                      label="フォロワー数"
                      sortKey="followerCount"
                      activeKey={sortKey}
                      direction={sortDir}
                      onChange={handleChangeSort}
                    />
                  </TableHead>
                  <TableHead>
                    <SortableTableHeader
                      label="投稿数"
                      sortKey="postCount"
                      activeKey={sortKey}
                      direction={sortDir}
                      onChange={handleChangeSort}
                    />
                  </TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {pagedOwners.map((owner, index) => {
                  const absoluteIndex = (currentPage - 1) * PAGE_SIZE + index;
                  const key = ownerKey(owner, absoluteIndex);
                  const avatarName = String(owner.avatarName ?? "").trim();
                  const avatarIconUrl = String(owner.avatarIconUrl ?? "").trim();
                  const productName = String(owner.productName ?? "").trim();
                  const followerCount = toSafeNumber(owner.followerCount);
                  const postCount = toSafeNumber(owner.postCount);
                  const checked = selectedOwnerKeys.includes(key);

                  return (
                    <TableRow
                      key={key}
                      data-state={checked ? "selected" : undefined}
                    >
                      <TableCell>
                        <Checkbox
                          id={`sales-owner-${key}`}
                          checked={checked}
                          onCheckedChange={(nextChecked) =>
                            handleToggleRow(key, nextChecked)
                          }
                        />
                      </TableCell>

                      <TableCell>
                        <div className="flex items-center gap-3">
                          <AvatarIcon
                            src={avatarIconUrl}
                            name={avatarName}
                            size="md"
                          />

                          <div className="h-10 flex items-center text-sm font-medium text-slate-900">
                            {avatarName || "-"}
                          </div>
                        </div>
                      </TableCell>

                      <TableCell>{productName || "-"}</TableCell>
                      <TableCell>{followerCount}</TableCell>
                      <TableCell>{postCount}</TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>

            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={setCurrentPage}
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}