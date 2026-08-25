// frontend/console/shell/src/pages/inquiryManagement.tsx 
 
import { useMemo } from "react"; 
 
import List, { 
  FilterableTableHeader, 
  SortableTableHeader, 
} from "../../../shell/src/layout/List/List"; 
 
import { useInquiryManagementPage } from "../features/inquiry/presentation/hooks/useInquiryManagementPage"; 
 
export default function InquiryManagementPage() { 
  const { 
    loading, 
    isResetting, 
    errorMessage, 
 
    rows, 
 
    statusFilter, 
    productNameFilter, 
    brandNameFilter, 
 
    statusOptions, 
    productNameOptions, 
    brandNameOptions, 
 
    sortKey, 
    sortDirection, 
 
    setStatusFilter, 
    setProductNameFilter, 
    setBrandNameFilter, 
 
    handleSortChange, 
    handleRefresh, 
    handleClickRow, 
  } = useInquiryManagementPage(); 
 
  const rowElements = useMemo(() => { 
    return rows.map((row) => { 
      return ( 
        <tr 
          key={row.inquiryId} 
          role="button" 
          tabIndex={0} 
          style={{ 
            cursor: "pointer", 
          }} 
          onClick={() => { 
            handleClickRow(row.inquiryId); 
          }} 
          onKeyDown={(event) => { 
            if ( 
              event.key === "Enter" || 
              event.key === " " 
            ) { 
              event.preventDefault(); 
              handleClickRow(row.inquiryId); 
            } 
          }} 
        > 
          <td>{row.subject}</td> 
          <td>{row.inquiryType}</td> 
          <td>{row.customerName}</td> 
          <td>{row.status}</td> 
          <td>{row.productName}</td> 
          <td>{row.brandName}</td> 
          <td>{row.createdAt}</td> 
          <td>{row.updatedAt}</td> 
        </tr> 
      ); 
    }); 
  }, [handleClickRow, rows]); 
 
  const headers = useMemo(() => { 
    return [ 
      "件名", 
      "問い合わせ種別", 
      "お客様名", 
 
      <FilterableTableHeader 
        key="status" 
        label="ステータス" 
        options={statusOptions} 
        selected={statusFilter} 
        onChange={setStatusFilter} 
      />, 
 
      <FilterableTableHeader 
        key="productName" 
        label="商品名" 
        options={productNameOptions} 
        selected={productNameFilter} 
        onChange={setProductNameFilter} 
      />, 
 
      <FilterableTableHeader 
        key="brandName" 
        label="ブランド" 
        options={brandNameOptions} 
        selected={brandNameFilter} 
        onChange={setBrandNameFilter} 
      />, 
 
      <SortableTableHeader 
        key="createdAt" 
        label="問い合わせ日" 
        sortKey="createdAt" 
        activeKey={sortKey} 
        direction={sortDirection} 
        onChange={handleSortChange} 
      />, 
 
      <SortableTableHeader 
        key="updatedAt" 
        label="最終更新日" 
        sortKey="updatedAt" 
        activeKey={sortKey} 
        direction={sortDirection} 
        onChange={handleSortChange} 
      />, 
    ]; 
  }, [ 
    statusOptions, 
    statusFilter, 
    setStatusFilter, 
 
    productNameOptions, 
    productNameFilter, 
    setProductNameFilter, 
 
    brandNameOptions, 
    brandNameFilter, 
    setBrandNameFilter, 
 
    sortKey, 
    sortDirection, 
    handleSortChange, 
  ]); 
 
  return ( 
    <div className="p-0"> 
      <List 
        title="問い合わせ管理" 
        headerCells={headers} 
        showCreateButton={false} 
        showResetButton 
        onReset={handleRefresh} 
        isResetting={isResetting} 
      > 
        {loading ? ( 
          <tr> 
            <td colSpan={8}> 
              <div className="inq__empty"> 
                問い合わせ一覧を読み込み中です。 
              </div> 
            </td> 
          </tr> 
        ) : errorMessage ? ( 
          <tr> 
            <td colSpan={8}> 
              <div className="inq__empty"> 
                {errorMessage} 
              </div> 
            </td> 
          </tr> 
        ) : rowElements.length > 0 ? ( 
          rowElements 
        ) : ( 
          <tr> 
            <td colSpan={8}> 
              <div className="inq__empty"> 
                問い合わせはありません。 
              </div> 
            </td> 
          </tr> 
        )} 
      </List> 
    </div> 
  ); 
}