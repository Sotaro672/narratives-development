// frontend/sns/lib/features/list/presentation/components/catalog_measurement.dart
import 'package:flutter/material.dart';

import '../hook/use_catalog_measurement.dart';

/// Measurement card (style only)
/// - models: ModelVariationDTO list など（metadata ラッパーも許容）
class CatalogMeasurementCard extends StatelessWidget {
  const CatalogMeasurementCard({
    super.key,
    required this.models,
    this.title = '採寸（サイズ別）',
  });

  final List<dynamic> models;
  final String title;

  @override
  Widget build(BuildContext context) {
    final vm = const UseCatalogMeasurement().build(
      models: models,
      title: title,
    );

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(vm.title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 10),
            if (!vm.hasRows)
              Text('採寸データがありません', style: Theme.of(context).textTheme.bodyMedium)
            else if (!vm.hasKeys)
              Text(
                'measurements が空です',
                style: Theme.of(context).textTheme.bodyMedium,
              )
            else
              _MeasurementTable(vm: vm),
          ],
        ),
      ),
    );
  }
}

class _MeasurementTable extends StatelessWidget {
  const _MeasurementTable({required this.vm});

  final CatalogMeasurementVM vm;

  TableRow _headerRow(BuildContext context) {
    final t = Theme.of(context).textTheme;

    Widget cell(
      String s, {
      bool bold = false,
      Alignment align = Alignment.centerLeft,
    }) {
      final style = bold ? t.labelLarge : t.labelMedium;
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        alignment: align,
        child: Text(
          s,
          style: style,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
      );
    }

    return TableRow(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
      ),
      children: [
        cell('サイズ', bold: true),
        ...vm.keys.map(
          (k) => cell(k, bold: true, align: Alignment.centerRight),
        ),
      ],
    );
  }

  TableRow _dataRow(BuildContext context, CatalogMeasurementRowVM r) {
    final t = Theme.of(context).textTheme;

    Widget cellText(String s, {Alignment align = Alignment.centerLeft}) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        alignment: align,
        child: Text(s, style: t.bodyMedium),
      );
    }

    String val(String key) {
      final v = r.measurements[key];
      if (v == null) return '-';
      return v.toString();
    }

    return TableRow(
      children: [
        cellText(r.size),
        ...vm.keys.map((k) => cellText(val(k), align: Alignment.centerRight)),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minWidth: 520),
        child: Table(
          defaultVerticalAlignment: TableCellVerticalAlignment.middle,
          border: TableBorder.all(
            color: Theme.of(context).dividerColor.withValues(alpha: 0.4),
          ),
          columnWidths: const <int, TableColumnWidth>{
            0: IntrinsicColumnWidth(),
          },
          children: [
            _headerRow(context),
            for (final r in vm.rows) _dataRow(context, r),
          ],
        ),
      ),
    );
  }
}
