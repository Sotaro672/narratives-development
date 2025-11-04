//frontend\shell\src\layout\Main\Main.tsx
import { Routes, Route } from "react-router-dom";
import InquiryManagementPage from "../../../../inquiry/src/pages/InquiryManagementPage";
import ProductBlueprintManagementPage from "../../../../productBlueprint/src/pages/productBlueprintManagement";
import ProductionManagementPage from "../../../../production/src/pages/productionManagement";
import InventoryManagementPage from "../../../../inventory/src/pages/inventoryManagement";
import TokenBlueprintManagementPage from "../../../../tokenBlueprint/src/pages/tokenBlueprintManagement";
import MintRequestManagementPage from "../../../../mintRequest/src/pages/mintRequestManagement";
import TokenOperationpage from "../../../../operation/src/pages/tokenOperation";
import ListManagementPage from "../../../../list/src/pages/listManagement";
import OrderManagementPage from "../../../../order/src/pages/orderManagement";
import MemberManagementPage from "../../../../member/src/pages/memberManagement";
import BrandManagementPage from "../../../../brand/src/pages/brandManagement";
import PermissionListPage from "../../../../permission/src/pages/permissionList";
import AdManagementPage from "../../../../ad/src/pages/adManagement";
import AccountManagementPage from "../../../../account/src/pages/accountManagement";
import TransactionListPage from "../../../../transaction/src/pages/transactionList";
import "./Main.css";

export default function Main() {
  return (
    <div className="main-content">
      <Routes>
        {<Route path="/inquiry" element={<InquiryManagementPage />} />}
        {<Route path="/productBlueprint" element={<ProductBlueprintManagementPage />} />}
        {<Route path="/production" element={<ProductionManagementPage />} />}
        {<Route path="/inventory" element={<InventoryManagementPage />} />}
        {<Route path="/tokenBlueprint" element={<TokenBlueprintManagementPage />} />}
        {<Route path="/mintRequest" element={<MintRequestManagementPage />} />}
        {<Route path="/operation" element={<TokenOperationpage />} />}
        {<Route path="/list" element={<ListManagementPage />} />}
        {<Route path="/order" element={<OrderManagementPage />} />}
        {<Route path="/member" element={<MemberManagementPage />} />}
        {<Route path="/brand" element={<BrandManagementPage />} />}
        {<Route path="/permission" element={<PermissionListPage />} />}
        {<Route path="/ad" element={<AdManagementPage />} />}
        {<Route path="/account" element={<AccountManagementPage />} />}
        {<Route path="/transaction" element={<TransactionListPage />} />}
      </Routes>
    </div>
  );
}
