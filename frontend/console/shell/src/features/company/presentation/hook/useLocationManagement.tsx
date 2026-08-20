// frontend/console/shell/src/features/company/presentation/hook/useLocationManagement.tsx  
  
import {  
  useCallback,  
  useEffect,  
  useMemo,  
  useState,  
} from "react";  
import { useNavigate } from "react-router-dom";  
  
import type { CompanyShippingAddressReadModel } from "../../../../shared/types/shippingAddress";  
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";  
  
import {  
  listCompanyShippingAddresses,  
} from "../../application/locationManagementService";  
  
export type LocationManagementRow = {  
  id: string;  
  name: string;  
  zipCode: string;  
  address: string;  
  createdByName: string;  
  createdAt: string;  
  updatedByName: string;  
  updatedAt: string;  
};  
  
export type UseLocationManagementResult = {  
  rows: LocationManagementRow[];  
  
  handlers: {  
    handleCreate: () => void;  
    handleRowClick: (  
      row: LocationManagementRow,  
    ) => void;  
    handleReset: () => void;  
  };  
  
  isResetting: boolean;  
};  
  
function buildAddress(  
  address: CompanyShippingAddressReadModel,  
): string {  
  return [  
    address.state,  
    address.city,  
    address.street,  
    address.street2,  
  ]  
    .filter((value) => Boolean(value))  
    .join("");  
}  
  
function toManagementRow(  
  address: CompanyShippingAddressReadModel,  
): LocationManagementRow {  
  return {  
    id: address.id,  
    name: address.name,  
    zipCode: address.zipCode,  
    address: buildAddress(address),  
    createdByName: address.createdByName || "",  
    createdAt: safeDateTimeLabelJa(  
      address.createdAt,  
      "",  
    ),  
    updatedByName: address.updatedByName || "",  
    updatedAt: safeDateTimeLabelJa(  
      address.updatedAt,  
      "",  
    ),  
  };  
}  
  
export function useLocationManagement(): UseLocationManagementResult {  
  const navigate = useNavigate();  
  
  const [  
    locations,  
    setLocations,  
  ] = useState<CompanyShippingAddressReadModel[]>(  
    [],  
  );  
  
  const [  
    isResetting,  
    setIsResetting,  
  ] = useState(false);  
  
  const load =  
    useCallback(  
      async (): Promise<void> => {  
        setIsResetting(true);  
  
        try {  
          const result =  
            await listCompanyShippingAddresses();  
  
          setLocations(  
            result,  
          );  
        } catch {  
          setLocations(  
            [],  
          );  
        } finally {  
          setIsResetting(false);  
        }  
      },  
      [],  
    );  
  
  useEffect(() => {  
    void load();  
  }, [  
    load,  
  ]);  
  
  const rows =  
    useMemo<LocationManagementRow[]>(  
      () =>  
        locations.map(  
          toManagementRow,  
        ),  
      [  
        locations,  
      ],  
    );  
  
  const handleCreate =  
    useCallback(() => {  
      navigate(  
        "/stockLocation/create",  
      );  
    }, [  
      navigate,  
    ]);  
  
  const handleRowClick =  
    useCallback(  
      (  
        row: LocationManagementRow,  
      ) => {  
        if (!row.id) {  
          return;  
        }  
  
        navigate(  
          `/stockLocation/${encodeURIComponent(  
            row.id,  
          )}`,  
        );  
      },  
      [  
        navigate,  
      ],  
    );  
  
  const handleReset =  
    useCallback(() => {  
      void load();  
    }, [  
      load,  
    ]);  
  
  return {  
    rows,  
  
    handlers: {  
      handleCreate,  
      handleRowClick,  
      handleReset,  
    },  
  
    isResetting,  
  };  
}  
  
export default useLocationManagement;