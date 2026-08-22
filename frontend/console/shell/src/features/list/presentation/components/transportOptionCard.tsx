// frontend/console/shell/src/features/list/presentation/components/transportOptionCard.tsx  
  
import type {  
  TransportationOption,  
} from "../../../../shared/types/inventory";  
  
import {  
  Card,  
  CardContent,  
} from "../../../../shared/ui/card";  
  
export type TransportOptionCardOption = {  
  transportationOption: TransportationOption;  
  transportationId?: string;  
  name: string;  
};  
  
type TransportOptionCardProps = {  
  options: TransportOptionCardOption[];  
  
  transportationOption: TransportationOption | "";  
  transportationId: string;  
  
  onSelectTransportationOption: (  
    value: string,  
  ) => void;  
  
  setTransportationId: (  
    value: string,  
  ) => void;  
  
  loading?: boolean;  
  disabled?: boolean;  
};  
  
function buildOptionValue(  
  option: TransportOptionCardOption,  
): string {  
  if (  
    option.transportationOption === "custom"  
  ) {  
    if (!option.transportationId) {  
      return "";  
    }  
  
    return `custom:${option.transportationId}`;  
  }  
  
  return option.transportationOption;  
}  
  
function buildSelectedValue(  
  transportationOption: TransportationOption | "",  
  transportationId: string,  
): string {  
  if (!transportationOption) {  
    return "";  
  }  
  
  if (transportationOption === "custom") {  
    if (!transportationId) {  
      return "";  
    }  
  
    return `custom:${transportationId}`;  
  }  
  
  return transportationOption;  
}  
  
export default function TransportOptionCard({  
  options,  
  transportationOption,  
  transportationId,  
  onSelectTransportationOption,  
  setTransportationId,  
  loading = false,  
  disabled = false,  
}: TransportOptionCardProps) {  
  const selectedValue =  
    buildSelectedValue(  
      transportationOption,  
      transportationId,  
    );  
  
  const selectableOptions =  
    options.filter((option) => {  
      if (  
        option.transportationOption !==  
        "custom"  
      ) {  
        return true;  
      }  
  
      return Boolean(  
        option.transportationId,  
      );  
    });  
  
  const handleChange = (  
    value: string,  
  ) => {  
    if (disabled) {  
      return;  
    }  
  
    if (!value) {  
      onSelectTransportationOption("");  
      setTransportationId("");  
      return;  
    }  
  
    const selectedOption =  
      selectableOptions.find(  
        (option) =>  
          buildOptionValue(option) ===  
          value,  
      );  
  
    if (!selectedOption) {  
      onSelectTransportationOption("");  
      setTransportationId("");  
      return;  
    }  
  
    onSelectTransportationOption(  
      selectedOption.transportationOption,  
    );  
  
    if (  
      selectedOption.transportationOption ===  
      "custom"  
    ) {  
      setTransportationId(  
        selectedOption.transportationId ?? "",  
      );  
      return;  
    }  
  
    setTransportationId("");  
  };  
  
  return (  
    <Card>  
      <CardContent className="p-4 space-y-2">  
        <div className="text-sm font-medium">  
          配送方法  
        </div>  
  
        {loading ? (  
          <div className="text-xs text-slate-400">  
            配送方法を読み込み中です…  
          </div>  
        ) : selectableOptions.length > 0 ? (  
          <select  
            value={selectedValue}  
            disabled={disabled}  
            onChange={(event) =>  
              handleChange(  
                event.target.value,  
              )  
            }  
            className="w-full h-10 px-3 rounded-md border border-slate-200 bg-white text-sm text-slate-800 outline-none focus:ring-2 focus:ring-slate-200 disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500"  
          >  
            <option value="">  
              配送方法を選択してください  
            </option>  
  
            {selectableOptions.map(  
              (option) => {  
                const value =  
                  buildOptionValue(option);  
  
                return (  
                  <option  
                    key={value}  
                    value={value}  
                  >  
                    {option.name}  
                  </option>  
                );  
              },  
            )}  
          </select>  
        ) : (  
          <div className="text-xs text-slate-400">  
            選択可能な配送方法がありません。  
          </div>  
        )}  
  
        {transportationOption ===  
          "custom" &&  
          transportationId && (  
            <div className="text-xs text-slate-500">  
              自社配送料金設定を使用します。  
            </div>  
          )}  
      </CardContent>  
    </Card>  
  );  
}