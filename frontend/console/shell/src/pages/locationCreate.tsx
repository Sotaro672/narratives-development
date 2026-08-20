// frontend/console/shell/src/pages/locationCreate.tsx 
 
import { 
  Card, 
  CardContent, 
} from "../shared/ui/card"; 
import LocationFormFields from "../features/company/presentation/components/LocationFormFields"; 
import { useLocationCreate } from "../features/company/presentation/hook/useLocationCreate"; 
import PageStyle from "../layout/PageStyle/PageStyle"; 
 
export default function LocationCreate() { 
  const { 
    vm, 
    handlers, 
  } = useLocationCreate(); 
 
  const disabled = vm.saving; 
 
  return ( 
    <PageStyle 
      layout="single" 
      title="在庫保管場所登録" 
      onBack={handlers.onBack} 
      onSave={handlers.onSave} 
      isSaving={vm.saving} 
    > 
      <div className="mx-auto w-full max-w-3xl"> 
        <Card> 
          <CardContent> 
            <LocationFormFields 
              value={{ 
                name: vm.name, 
                zipCode: vm.zipCode, 
                state: vm.state, 
                city: vm.city, 
                street: vm.street, 
                street2: vm.street2, 
              }} 
              errors={{ 
                name: vm.nameError, 
                zipCode: vm.zipCodeError, 
                state: vm.stateError, 
                city: vm.cityError, 
                street: vm.streetError, 
              }} 
              disabled={disabled} 
              onChangeName={handlers.onChangeName} 
              onChangeZipCode={handlers.onChangeZipCode} 
              onChangeState={handlers.onChangeState} 
              onChangeCity={handlers.onChangeCity} 
              onChangeStreet={handlers.onChangeStreet} 
              onChangeStreet2={handlers.onChangeStreet2} 
            /> 
 
            {vm.error && ( 
              <div 
                role="alert" 
                className="mt-5 rounded-lg border border-red-200 bg-red-50 px-4 py-3" 
              > 
                <p className="text-sm text-red-600"> 
                  {vm.error} 
                </p> 
              </div> 
            )} 
 
            {vm.saving && ( 
              <p className="mt-5 text-sm text-slate-500"> 
                在庫保管場所を登録しています... 
              </p> 
            )} 
          </CardContent> 
        </Card> 
      </div> 
    </PageStyle> 
  ); 
}