// frontend/console/shell/src/features/inventory/presentation/hook/useInventoryDetail.tsx 
 
import * as React from "react"; 
 
import type { 
  InventoryDetailRowDTO, 
  InventoryDetailViewModel, 
  InventoryShippingAddressDTO, 
  InventoryTransportationOptionDTO, 
  TransportationOption, 
} from "../../../../shared/types/inventory"; 
 
import { 
  loadInventoryDetailViewModel, 
  saveInventoryShippingAddress, 
  saveInventoryTransportation, 
} from "../../application/inventoryDetailService"; 
 
import { fetchListsByInventoryIdHTTP } from "../../../list/infrastructure/repository"; 
 
export type InventoryListItem = { 
  id: string; 
  readableId: string; 
}; 
 
export type UseInventoryDetailResult = { 
  vm: InventoryDetailViewModel | null; 
  rows: InventoryDetailRowDTO[]; 
  loading: boolean; 
  error: string | null; 
 
  selectedShippingAddressId: string; 
  shippingAddressOptions: InventoryShippingAddressDTO[]; 
  shippingAddressSaving: boolean; 
  shippingAddressError: string | null; 
 
  transportationOption: TransportationOption | ""; 
  transportationId: string; 
  transportationOptions: InventoryTransportationOptionDTO[]; 
  transportationSaving: boolean; 
  transportationError: string | null; 
 
  listItems: InventoryListItem[]; 
  listLoading: boolean; 
  listError: string | null; 
 
  handleSelectShippingAddress: (shippingAddressId: string) => void; 
  handleSaveShippingAddress: () => Promise<void>; 
 
  handleSelectTransportationOption: (value: string) => void; 
  setTransportationId: (transportationId: string) => void; 
  handleSaveTransportation: () => Promise<void>; 
}; 
 
export function useInventoryDetail( 
  inventoryId: string | undefined, 
): UseInventoryDetailResult { 
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null); 
  const [loading, setLoading] = React.useState(false); 
  const [error, setError] = React.useState<string | null>(null); 
 
  const [selectedShippingAddressId, setSelectedShippingAddressId] = React.useState(""); 
  const [shippingAddressSaving, setShippingAddressSaving] = React.useState(false); 
  const [shippingAddressError, setShippingAddressError] = React.useState<string | null>(null); 
 
  const [transportationOption, setTransportationOption] = React.useState<TransportationOption | "">(""); 
  const [transportationId, setTransportationIdState] = React.useState(""); 
  const [transportationSaving, setTransportationSaving] = React.useState(false); 
  const [transportationError, setTransportationError] = React.useState<string | null>(null); 
 
  const [listItems, setListItems] = React.useState<InventoryListItem[]>([]); 
  const [listLoading, setListLoading] = React.useState(false); 
  const [listError, setListError] = React.useState<string | null>(null); 
 
  const invId = React.useMemo(() => inventoryId ?? "", [inventoryId]); 
 
  React.useEffect(() => { 
    if (!invId) { 
      setVm(null); 
      setError(null); 
      setLoading(false); 
      setSelectedShippingAddressId(""); 
      setShippingAddressSaving(false); 
      setShippingAddressError(null); 
      setTransportationOption(""); 
      setTransportationIdState(""); 
      setTransportationSaving(false); 
      setTransportationError(null); 
      return; 
    } 
 
    let cancelled = false; 
 
    async function load() { 
      try { 
        setLoading(true); 
        setError(null); 
        setShippingAddressError(null); 
        setTransportationError(null); 
 
        const nextVm = await loadInventoryDetailViewModel(invId); 
        if (cancelled) { 
          return; 
        } 
 
        setVm(nextVm); 
        setSelectedShippingAddressId(nextVm.shippingAddressId); 
        setTransportationOption(nextVm.transportationOption); 
        setTransportationIdState(nextVm.transportationId); 
      } catch (error) { 
        if (cancelled) { 
          return; 
        } 
 
        setError( 
          error instanceof Error 
            ? error.message 
            : String(error), 
        ); 
        setVm(null); 
        setSelectedShippingAddressId(""); 
        setTransportationOption(""); 
        setTransportationIdState(""); 
      } finally { 
        if (!cancelled) { 
          setLoading(false); 
        } 
      } 
    } 
 
    void load(); 
 
    return () => { 
      cancelled = true; 
    }; 
  }, [invId]); 
 
  React.useEffect(() => { 
    if (!invId) { 
      setListItems([]); 
      setListLoading(false); 
      setListError(null); 
      return; 
    } 
 
    let cancelled = false; 
 
    async function loadLists() { 
      try { 
        setListLoading(true); 
        setListError(null); 
 
        const items = await fetchListsByInventoryIdHTTP(invId); 
        if (cancelled) { 
          return; 
        } 
 
        setListItems( 
          items 
            .filter((item) => item.inventoryId === invId) 
            .map((item) => ({ 
              id: item.id, 
              readableId: item.readableId, 
            })), 
        ); 
      } catch (error) { 
        if (cancelled) { 
          return; 
        } 
 
        setListItems([]); 
        setListError( 
          error instanceof Error 
            ? error.message 
            : String(error), 
        ); 
      } finally { 
        if (!cancelled) { 
          setListLoading(false); 
        } 
      } 
    } 
 
    void loadLists(); 
 
    return () => { 
      cancelled = true; 
    }; 
  }, [invId]); 
 
  const rows = React.useMemo<InventoryDetailRowDTO[]>( 
    () => vm?.rows ?? [], 
    [vm], 
  ); 
 
  const shippingAddressOptions = React.useMemo<InventoryShippingAddressDTO[]>( 
    () => vm?.shippingAddressOptions ?? [], 
    [vm], 
  ); 
 
  const transportationOptions = React.useMemo<InventoryTransportationOptionDTO[]>( 
    () => vm?.transportationOptions ?? [], 
    [vm], 
  ); 
 
  const handleSelectShippingAddress = React.useCallback( 
    (shippingAddressId: string) => { 
      if (!shippingAddressId) { 
        return; 
      } 
 
      setSelectedShippingAddressId(shippingAddressId); 
      setShippingAddressError(null); 
    }, 
    [], 
  ); 
 
  const handleSaveShippingAddress = React.useCallback( 
    async () => { 
      if (!invId) { 
        setShippingAddressError("inventoryId is empty"); 
        return; 
      } 
 
      if (!selectedShippingAddressId) { 
        setShippingAddressError("在庫保管場所を選択してください。"); 
        return; 
      } 
 
      if (vm?.shippingAddressId === selectedShippingAddressId) { 
        setShippingAddressError(null); 
        return; 
      } 
 
      try { 
        setShippingAddressSaving(true); 
        setShippingAddressError(null); 
 
        const nextVm = await saveInventoryShippingAddress( 
          invId, 
          selectedShippingAddressId, 
        ); 
 
        setVm(nextVm); 
        setSelectedShippingAddressId(nextVm.shippingAddressId); 
        setTransportationOption(nextVm.transportationOption); 
        setTransportationIdState(nextVm.transportationId); 
      } catch (error) { 
        setShippingAddressError( 
          error instanceof Error 
            ? error.message 
            : String(error), 
        ); 
      } finally { 
        setShippingAddressSaving(false); 
      } 
    }, 
    [ 
      invId, 
      selectedShippingAddressId, 
      vm?.shippingAddressId, 
    ], 
  ); 
 
  const handleSelectTransportationOption = React.useCallback( 
    (value: string) => { 
      switch (value) { 
        case "yamato": 
        case "sagawa": 
        case "post": 
          setTransportationOption(value); 
          setTransportationIdState(""); 
          setTransportationError(null); 
          return; 
 
        case "custom": 
          setTransportationOption(value); 
          setTransportationError(null); 
          return; 
 
        default: 
          setTransportationOption(""); 
          setTransportationIdState(""); 
          setTransportationError(null); 
      } 
    }, 
    [], 
  ); 
 
  const setTransportationId = React.useCallback( 
    (nextTransportationId: string) => { 
      setTransportationIdState(nextTransportationId); 
      setTransportationError(null); 
    }, 
    [], 
  ); 
 
  const handleSaveTransportation = React.useCallback( 
    async () => { 
      if (!invId) { 
        setTransportationError("inventoryId is empty"); 
        return; 
      } 
 
      if (!transportationOption) { 
        setTransportationError("配送方法を選択してください。"); 
        return; 
      } 
 
      if ( 
        transportationOption === "custom" && 
        !transportationId 
      ) { 
        setTransportationError("自社配送料金設定を選択してください。"); 
        return; 
      } 
 
      if ( 
        transportationOption !== "custom" && 
        transportationId 
      ) { 
        setTransportationError("固定配送方法ではtransportationIdを指定できません。"); 
        return; 
      } 
 
      const savedTransportationOption = 
        vm?.transportationOption ?? ""; 
 
      const savedTransportationId = 
        vm?.transportationId ?? ""; 
 
      if ( 
        savedTransportationOption === transportationOption && 
        savedTransportationId === transportationId 
      ) { 
        setTransportationError(null); 
        return; 
      } 
 
      try { 
        setTransportationSaving(true); 
        setTransportationError(null); 
 
        const nextVm = await saveInventoryTransportation( 
          invId, 
          transportationOption, 
          transportationId, 
        ); 
 
        setVm(nextVm); 
        setSelectedShippingAddressId(nextVm.shippingAddressId); 
        setTransportationOption(nextVm.transportationOption); 
        setTransportationIdState(nextVm.transportationId); 
      } catch (error) { 
        setTransportationError( 
          error instanceof Error 
            ? error.message 
            : String(error), 
        ); 
      } finally { 
        setTransportationSaving(false); 
      } 
    }, 
    [ 
      invId, 
      transportationOption, 
      transportationId, 
      vm?.transportationOption, 
      vm?.transportationId, 
    ], 
  ); 
 
  return { 
    vm, 
    rows, 
    loading, 
    error, 
 
    selectedShippingAddressId, 
    shippingAddressOptions, 
    shippingAddressSaving, 
    shippingAddressError, 
 
    transportationOption, 
    transportationId, 
    transportationOptions, 
    transportationSaving, 
    transportationError, 
 
    listItems, 
    listLoading, 
    listError, 
 
    handleSelectShippingAddress, 
    handleSaveShippingAddress, 
 
    handleSelectTransportationOption, 
    setTransportationId, 
    handleSaveTransportation, 
  }; 
}