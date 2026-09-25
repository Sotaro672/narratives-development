// frontend/mall/src/features/catalog/presentation/components/MeasurementTable.tsx

import SectionHeader from "../../../../components/ui/SectionHeader";
import type { MeasurementTableRow } from "../../../shared/types/catalog";

type MeasurementTableProps = {
  measurementRows: MeasurementTableRow[];
  measurementKeys: string[];
};

export default function MeasurementTable({
  measurementRows,
  measurementKeys,
}: MeasurementTableProps) {
  return (
    <section className="catalog-page-measurement">
      <SectionHeader
        title="採寸表"
        titleAs="h2"
        className="catalog-page-card-header"
      />

      <div className="catalog-page-measurement-table-wrap">
        <table className="catalog-page-measurement-table">
          <thead>
            <tr>
              <th scope="col">サイズ</th>
              {measurementKeys.map((key) => (
                <th key={key} scope="col">{key}</th>
              ))}
            </tr>
          </thead>

          <tbody>
            {measurementRows.map((row) => (
              <tr key={row.id}>
                <th scope="row">{row.size}</th>
                {measurementKeys.map((key) => (
                  <td key={`${row.id}-${key}`}>
                    {typeof row.measurements?.[key] === "number"
                      ? row.measurements[key]
                      : "-"}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}