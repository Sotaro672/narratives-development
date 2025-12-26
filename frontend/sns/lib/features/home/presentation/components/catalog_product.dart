//frontend\sns\lib\features\home\presentation\components\catalog_product.dart
import 'package:flutter/material.dart';

/// ProductBlueprint の型定義がこのファイルから直接参照できないため、
/// 受け取りは dynamic にしています（CatalogPage 側から vm.productBlueprint をそのまま渡す想定）。
class CatalogProductCard extends StatelessWidget {
  const CatalogProductCard({
    super.key,
    required this.productBlueprintId,
    required this.productBlueprint,
    required this.error,
  });

  final String productBlueprintId;
  final dynamic
  productBlueprint; // ProductBlueprint DTO/entity (from use_catalog)
  final String? error;

  String _s(String? v, {String fallback = '(empty)'}) {
    final t = (v ?? '').trim();
    return t.isNotEmpty ? t : fallback;
  }

  @override
  Widget build(BuildContext context) {
    final pbId = productBlueprintId.trim();
    final pb = productBlueprint;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Product', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            if (pb != null) ...[
              _KeyValueRow(label: 'productName', value: _s(pb.productName)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'brandId', value: _s(pb.brandId)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'companyId', value: _s(pb.companyId)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'itemType', value: _s(pb.itemType)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'fit', value: _s(pb.fit)),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'material', value: _s(pb.material)),
              const SizedBox(height: 6),
              _KeyValueRow(
                label: 'weight',
                value: pb.weight != null ? '${pb.weight}' : '(empty)',
              ),
              const SizedBox(height: 6),
              _KeyValueRow(
                label: 'printed',
                value: pb.printed == true ? 'true' : 'false',
              ),
              const SizedBox(height: 12),
              Text(
                'Quality assurance',
                style: Theme.of(context).textTheme.titleSmall,
              ),
              const SizedBox(height: 6),
              if ((pb.qualityAssurance ?? const <dynamic>[]).isEmpty)
                Text('(empty)', style: Theme.of(context).textTheme.bodyMedium)
              else
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: (pb.qualityAssurance as List)
                      .map(
                        (s) => Chip(
                          // ✅ String(s) は不可 → toString() / '$s'
                          label: Text(s.toString()),
                          visualDensity: VisualDensity.compact,
                        ),
                      )
                      .toList(),
                ),
              const SizedBox(height: 12),
              Text(
                'ProductId tag',
                style: Theme.of(context).textTheme.titleSmall,
              ),
              const SizedBox(height: 6),
              _KeyValueRow(label: 'type', value: _s(pb.productIdTagType)),
            ] else ...[
              _KeyValueRow(
                label: 'productBlueprintId',
                value: pbId.isNotEmpty ? pbId : '(unknown)',
              ),
              if (error != null && error!.trim().isNotEmpty) ...[
                const SizedBox(height: 10),
                Text(
                  'product error: ${error!.trim()}',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ] else ...[
                const SizedBox(height: 10),
                Text(
                  'product is not loaded',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }
}

class _KeyValueRow extends StatelessWidget {
  const _KeyValueRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 160,
          child: Text(label, style: Theme.of(context).textTheme.labelMedium),
        ),
        Expanded(child: Text(value)),
      ],
    );
  }
}
