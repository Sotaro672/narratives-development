// frontend/console/shell/src/features/list/application/listManagementService.tsx 
 
import type { ReactNode } from "react"; 
 
import { 
  FilterableTableHeader, 
  SortableTableHeader, 
} from "../../../layout/List/List"; 
import type { ListStatus } from "../../../shared/types/list"; 
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa"; 
 
import type { ListManagementRowDTO } from "../infrastructure/repository"; 
import { fetchListsHTTP } from "../infrastructure/repository"; 
 
export type SortKey = "createdAt" | null; 
 
export type ListManagementRowVM = { 
  id: string; 
  readableId: string; 
  title: string; 
  productName: string; 
  tokenName: string; 
  totalSalesAmount: number; 
  totalOrderCount: number; 
  assigneeName: string; 
  status: ListStatus; 
  createdAt: string; 
  createdAtRaw: string; 
  statusBadgeText: string; 
  statusBadgeClass: string; 
}; 
 
type FilterOption = { 
  value: string; 
  label: string; 
}; 
 
export type FilterOptions = { 
  productOptions: FilterOption[]; 
  tokenOptions: FilterOption[]; 
  managerOptions: FilterOption[]; 
  statusOptions: Array<{ 
    value: ListStatus; 
    label: string; 
  }>; 
}; 
 
export type Filters = { 
  productFilter: string[]; 
  tokenFilter: string[]; 
  managerFilter: string[]; 
  statusFilter: string[]; 
}; 
 
const LIST_STATUS_PRESENTATION: Record< 
  ListStatus, 
  { 
    label: string; 
    badgeClassName: string; 
  } 
> = { 
  listing: { 
    label: "出品中", 
    badgeClassName: "list-status-badge is-active", 
  }, 
  suspended: { 
    label: "保留中", 
    badgeClassName: "list-status-badge is-paused", 
  }, 
}; 
 
function mapListDTOToVMRow( 
  dto: ListManagementRowDTO, 
): ListManagementRowVM { 
  const statusPresentation = LIST_STATUS_PRESENTATION[dto.status]; 
 
  return { 
    id: dto.id, 
    readableId: dto.readableId, 
    title: dto.title, 
    productName: dto.productName, 
    tokenName: dto.tokenName, 
    totalSalesAmount: dto.totalSalesAmount, 
    totalOrderCount: dto.totalOrderCount, 
    assigneeName: dto.assigneeName, 
    status: dto.status, 
    createdAt: safeDateTimeLabelJa(dto.createdAt, ""), 
    createdAtRaw: dto.createdAt, 
    statusBadgeText: statusPresentation.label, 
    statusBadgeClass: statusPresentation.badgeClassName, 
  }; 
} 
 
export async function loadListManagementRows(): Promise<{ 
  rows: ListManagementRowVM[]; 
  error: string | null; 
}> { 
  try { 
    const items = await fetchListsHTTP(); 
 
    return { 
      rows: items.map(mapListDTOToVMRow), 
      error: null, 
    }; 
  } catch (error: unknown) { 
    return { 
      rows: [], 
      error: 
        error instanceof Error 
          ? error.message 
          : String(error ?? "unknown_error"), 
    }; 
  } 
} 
 
function buildTextFilterOptions( 
  values: string[], 
): FilterOption[] { 
  return Array.from(new Set(values)) 
    .filter((value) => value !== "") 
    .map((value) => ({ 
      value, 
      label: value, 
    })); 
} 
 
export function buildFilterOptions( 
  rows: ListManagementRowVM[], 
): FilterOptions { 
  const statuses = Array.from( 
    new Set(rows.map((row) => row.status)), 
  ); 
 
  return { 
    productOptions: buildTextFilterOptions( 
      rows.map((row) => row.productName), 
    ), 
    tokenOptions: buildTextFilterOptions( 
      rows.map((row) => row.tokenName), 
    ), 
    managerOptions: buildTextFilterOptions( 
      rows.map((row) => row.assigneeName), 
    ), 
    statusOptions: statuses.map((status) => ({ 
      value: status, 
      label: LIST_STATUS_PRESENTATION[status].label, 
    })), 
  }; 
} 
 
export function applyFilters( 
  rows: ListManagementRowVM[], 
  filters: Filters, 
): ListManagementRowVM[] { 
  return rows.filter( 
    (row) => 
      (filters.productFilter.length === 0 || 
        filters.productFilter.includes(row.productName)) && 
      (filters.tokenFilter.length === 0 || 
        filters.tokenFilter.includes(row.tokenName)) && 
      (filters.managerFilter.length === 0 || 
        filters.managerFilter.includes(row.assigneeName)) && 
      (filters.statusFilter.length === 0 || 
        filters.statusFilter.includes(row.status)), 
  ); 
} 
 
function toTimeMs(value: string): number { 
  const time = new Date(value).getTime(); 
 
  return Number.isNaN(time) ? 0 : time; 
} 
 
export function applySort( 
  rows: ListManagementRowVM[], 
  activeKey: SortKey, 
  direction: "asc" | "desc" | null, 
): ListManagementRowVM[] { 
  if (!activeKey || !direction) { 
    return rows; 
  } 
 
  const sortedRows = [...rows]; 
 
  sortedRows.sort((a, b) => { 
    const comparison = 
      toTimeMs(a.createdAtRaw) - 
      toTimeMs(b.createdAtRaw); 
 
    return direction === "asc" 
      ? comparison 
      : -comparison; 
  }); 
 
  return sortedRows; 
} 
 
export function buildHeaders(args: { 
  options: FilterOptions; 
  selected: Filters; 
  onChange: { 
    setProductFilter: (value: string[]) => void; 
    setTokenFilter: (value: string[]) => void; 
    setManagerFilter: (value: string[]) => void; 
    setStatusFilter: (value: string[]) => void; 
  }; 
  sort: { 
    activeKey: SortKey; 
    direction: "asc" | "desc" | null; 
    onChange: ( 
      key: SortKey, 
      direction: "asc" | "desc" | null, 
    ) => void; 
  }; 
}): ReactNode[] { 
  const { options, selected, onChange, sort } = args; 
 
  const onChangeCreatedAt = ( 
    _key: string, 
    nextDirection: "asc" | "desc", 
  ) => { 
    sort.onChange("createdAt", nextDirection); 
  }; 
 
  return [ 
    <span key="readableId">出品ID</span>, 
    <FilterableTableHeader 
      key="product" 
      label="プロダクト名" 
      options={options.productOptions} 
      selected={selected.productFilter} 
      onChange={onChange.setProductFilter} 
    />, 
    <FilterableTableHeader 
      key="token" 
      label="トークン名" 
      options={options.tokenOptions} 
      selected={selected.tokenFilter} 
      onChange={onChange.setTokenFilter} 
    />, 
    <span key="sales">累計売上</span>, 
    <span key="orders">注文数</span>, 
    <FilterableTableHeader 
      key="manager" 
      label="担当者" 
      options={options.managerOptions} 
      selected={selected.managerFilter} 
      onChange={onChange.setManagerFilter} 
    />, 
    <FilterableTableHeader 
      key="status" 
      label="ステータス" 
      options={options.statusOptions} 
      selected={selected.statusFilter} 
      onChange={onChange.setStatusFilter} 
    />, 
    <SortableTableHeader 
      key="createdAt" 
      label="作成日" 
      sortKey="createdAt" 
      activeKey={sort.activeKey ?? undefined} 
      direction={sort.direction ?? undefined} 
      onChange={onChangeCreatedAt} 
    />, 
  ]; 
} 