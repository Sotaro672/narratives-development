//frontend\shell\src\layout\Main\Main.tsx
import { Routes, Route } from "react-router-dom";
import InquiryManagementPage from "../../../../inquiry/src/pages/InquiryManagementPage";
import ProductBlueprintManagementPage from "../../../../productBlueprint/src/pages/productBlueprintManagement";
import ProductionManagementPage from "../../../../production/src/pages/productionManagement";
import InventoryManagementPage from "../../../../inventory/src/pages/inventoryManagement";
import TokenBlueprintManagementPage from "../../../../tokenBlueprint/src/pages/tokenBlueprintManagement";
import MintRequestManagementPage from "../../../../mint/src/pages/mintRequestManagement";
import TokenOperationpage from "../../../../operation/src/pages/tokenOperation";
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
        {<Route path="/mint" element={<MintRequestManagementPage />} />}
        {<Route path="/operation" element={<TokenOperationpage />} />}
      </Routes>
    </div>
  );
}
