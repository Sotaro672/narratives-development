// frontend/console/shell/src/pages/locationManagement.tsx  
  
import List from "../layout/List/List";  
import { useLocationManagement } from "../features/company/presentation/hook/useLocationManagement";  
  
export default function LocationManagement() {  
  const {  
    rows,  
    handlers: {  
      handleCreate,  
      handleRowClick,  
      handleReset,  
    },  
    isResetting,  
  } = useLocationManagement();  
  
  const headers = [  
    "保管場所名",  
    "住所",  
    "作成者",  
    "登録日",  
    "更新者",  
    "最終更新日",  
  ];  
  
  return (  
    <List  
      title="在庫保管場所"  
      headerCells={headers}  
      showCreateButton  
      createLabel="保管場所を追加"  
      onCreate={handleCreate}  
      showResetButton  
      isResetting={isResetting}  
      onReset={handleReset}  
    >  
      {rows.map((row) => (  
        <tr  
          key={row.id}  
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"  
          role="button"  
          tabIndex={0}  
          onClick={() => handleRowClick(row)}  
          onKeyDown={(event) => {  
            if (event.key === "Enter" || event.key === " ") {  
              event.preventDefault();  
              handleRowClick(row);  
            }  
          }}  
        >  
          <td>{row.name || "-"}</td>  
          <td>{row.address || "-"}</td>  
          <td>{row.createdByName || "-"}</td>  
          <td>{row.createdAt || "-"}</td>  
          <td>{row.updatedByName || "-"}</td>  
          <td>{row.updatedAt || "-"}</td>  
        </tr>  
      ))}  
    </List>  
  );  
}