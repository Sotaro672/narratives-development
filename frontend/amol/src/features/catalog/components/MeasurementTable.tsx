//frontend\amol\src\features\catalog\components\MeasurementTable.tsx
import type { MeasurementTableRow } from "../types";

type MeasurementTableProps = {
  measurementRows: MeasurementTableRow[];
  measurementKeys: string[];
};

export default function MeasurementTable({
  measurementRows,
  measurementKeys,
}: MeasurementTableProps) {
  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">採寸表</h2>

      <div className="catalog-page-measurement-table-wrap">
        <table className="catalog-page-measurement-table">
          <thead>
            <tr>
              <th scope="col">サイズ</th>
              {measurementKeys.map((key) => (
                <th key={key} scope="col">
                  {key}
                </th>
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