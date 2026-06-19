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

export type SalesOwnersCardMode = "view" | "edit";

type Props = {
  title?: string;
  mode?: SalesOwnersCardMode;
  owners?: SalesOwnerItem[];
  selectedAvatarIds?: string[];
  onSelectionChange?: (avatarIds: string[]) => void;
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

function getAvatarId(owner: SalesOwnerItem): string {
  return String(owner.avatarId ?? "").trim();
}

export default function SalesOwnersCard({
  title = "所有者一覧",
  mode = "edit",
  owners = [],
  selectedAvatarIds = [],
  onSelectionChange,
}: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("followerCount");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [selectedProductNames, setSelectedProductNames] = useState<string[]>(
    [],
  );
  const [currentPage, setCurrentPage] = useState(1);

  const isEditMode = mode === "edit";
  const isViewMode = mode === "view";

  const normalizedSelectedAvatarIds = useMemo(() => {
    return uniqueStrings(selectedAvatarIds);
  }, [selectedAvatarIds]);

  const selectedAvatarIdSet = useMemo(() => {
    return new Set(normalizedSelectedAvatarIds);
  }, [normalizedSelectedAvatarIds]);

  const visibleOwners = useMemo(() => {
    if (!isViewMode) {
      return owners;
    }

    if (normalizedSelectedAvatarIds.length === 0) {
      return owners;
    }

    return owners.filter((owner) => {
      const avatarId = getAvatarId(owner);
      return avatarId !== "" && selectedAvatarIdSet.has(avatarId);
    });
  }, [isViewMode, normalizedSelectedAvatarIds.length, owners, selectedAvatarIdSet]);

  const productFilterOptions = useMemo(() => {
    const names = visibleOwners.map((owner) =>
      String(owner.productName ?? "").trim(),
    );

    return uniqueStrings(names).map((name) => ({
      value: name,
      label: name,
    }));
  }, [visibleOwners]);

  const filteredOwners = useMemo(() => {
    if (selectedProductNames.length === 0) {
      return visibleOwners;
    }

    return visibleOwners.filter((owner) => {
      const productName = String(owner.productName ?? "").trim();
      return selectedProductNames.includes(productName);
    });
  }, [selectedProductNames, visibleOwners]);

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

  const pagedAvatarIds = useMemo(() => {
    return uniqueStrings(pagedOwners.map(getAvatarId));
  }, [pagedOwners]);

  const validAvatarIds = useMemo(() => {
    return uniqueStrings(owners.map(getAvatarId));
  }, [owners]);

  const selectedCount = useMemo(() => {
    if (isViewMode) {
      return visibleOwners.length;
    }

    return selectedAvatarIds.filter((avatarId) =>
      validAvatarIds.includes(String(avatarId ?? "").trim()),
    ).length;
  }, [isViewMode, selectedAvatarIds, validAvatarIds, visibleOwners.length]);

  const isAllSelected =
    isEditMode &&
    pagedAvatarIds.length > 0 &&
    pagedAvatarIds.every((avatarId) => selectedAvatarIdSet.has(avatarId));

  useEffect(() => {
    if (!isEditMode) {
      return;
    }

    const next = selectedAvatarIds.filter((avatarId) =>
      validAvatarIds.includes(String(avatarId ?? "").trim()),
    );

    if (next.length !== selectedAvatarIds.length) {
      onSelectionChange?.(uniqueStrings(next));
    }
  }, [isEditMode, onSelectionChange, selectedAvatarIds, validAvatarIds]);

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

  const handleToggleAll = (checked: boolean | "indeterminate") => {
    if (!isEditMode) return;
    if (pagedAvatarIds.length === 0) return;

    const nextChecked = checked === true;

    if (nextChecked) {
      onSelectionChange?.(
        uniqueStrings([...selectedAvatarIds, ...pagedAvatarIds]),
      );
      return;
    }

    onSelectionChange?.(
      selectedAvatarIds.filter(
        (avatarId) => !pagedAvatarIds.includes(String(avatarId ?? "").trim()),
      ),
    );
  };

  const handleToggleRow = (
    avatarId: string,
    checked: boolean | "indeterminate",
  ) => {
    if (!isEditMode) return;

    const normalizedAvatarId = String(avatarId ?? "").trim();
    if (!normalizedAvatarId) return;

    const nextChecked = checked === true;

    if (nextChecked) {
      onSelectionChange?.(
        uniqueStrings([...selectedAvatarIds, normalizedAvatarId]),
      );
      return;
    }

    onSelectionChange?.(
      selectedAvatarIds.filter(
        (item) => String(item ?? "").trim() !== normalizedAvatarId,
      ),
    );
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-3">
          <CardTitle>{title}</CardTitle>

          <div className="text-sm text-slate-500">
            {isEditMode ? `宛先選択 ${selectedCount} 件` : `宛先 ${selectedCount} 件`}
          </div>
        </div>
      </CardHeader>

      <CardContent>
        {visibleOwners.length === 0 ? (
          <p className="text-sm text-slate-500">
            表示可能なオーナーがありません。
          </p>
        ) : (
          <div className="space-y-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {isEditMode && (
                    <TableHead className="w-12">
                      <span
                        className={
                          pagedAvatarIds.length === 0
                            ? "opacity-40 pointer-events-none"
                            : undefined
                        }
                      >
                        <Checkbox
                          id="sales-owners-select-all"
                          checked={isAllSelected}
                          onCheckedChange={handleToggleAll}
                        />
                      </span>
                    </TableHead>
                  )}

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
                  const avatarId = getAvatarId(owner);
                  const avatarName = String(owner.avatarName ?? "").trim();
                  const avatarIconUrl = String(owner.avatarIconUrl ?? "").trim();
                  const productName = String(owner.productName ?? "").trim();
                  const followerCount = toSafeNumber(owner.followerCount);
                  const postCount = toSafeNumber(owner.postCount);
                  const checked =
                    avatarId !== "" && selectedAvatarIdSet.has(avatarId);

                  return (
                    <TableRow
                      key={key}
                      data-state={
                        isEditMode && checked ? "selected" : undefined
                      }
                    >
                      {isEditMode && (
                        <TableCell>
                          <span
                            className={
                              !avatarId
                                ? "opacity-40 pointer-events-none"
                                : undefined
                            }
                          >
                            <Checkbox
                              id={`sales-owner-${key}`}
                              checked={checked}
                              onCheckedChange={(nextChecked) =>
                                handleToggleRow(avatarId, nextChecked)
                              }
                            />
                          </span>
                        </TableCell>
                      )}

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