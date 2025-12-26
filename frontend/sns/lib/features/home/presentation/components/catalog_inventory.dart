// frontend\sns\lib\features\home\presentation\components\catalog_inventory.dart
import 'package:flutter/material.dart';

class CatalogInventoryCard extends StatelessWidget {
  const CatalogInventoryCard({
    super.key,
    required this.productBlueprintId,
    required this.tokenBlueprintId,
    required this.totalStock,
    required this.inventory,
    required this.inventoryError,
    required this.modelStockRows,
  });

  final String productBlueprintId;
  final String tokenBlueprintId;

  final int? totalStock;

  /// vm.inventory（型に依存しないため Object?）
  final Object? inventory;

  final String? inventoryError;

  /// vm.modelStockRows（elements must have: label, modelId, stockCount）
  final List<dynamic>? modelStockRows;

  @override
  Widget build(BuildContext context) {
    final pbId = productBlueprintId.trim();
    final tbId = tokenBlueprintId.trim();

    final inv = inventory;
    final invErr = inventoryError;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Inventory', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            _KeyValueRow(
              label: 'productBlueprintId',
              value: pbId.isNotEmpty ? pbId : '(unknown)',
            ),
            const SizedBox(height: 6),
            _KeyValueRow(
              label: 'tokenBlueprintId',
              value: tbId.isNotEmpty ? tbId : '(unknown)',
            ),
            const SizedBox(height: 6),
            _KeyValueRow(
              label: 'total stock',
              value: totalStock != null
                  ? totalStock.toString()
                  : '(not loaded)',
            ),

            if (inv != null) ...[
              const SizedBox(height: 12),
              Text('By model', style: Theme.of(context).textTheme.titleSmall),
              const SizedBox(height: 6),
              if (modelStockRows == null || modelStockRows!.isEmpty)
                Text('(empty)', style: Theme.of(context).textTheme.bodyMedium)
              else
                ...modelStockRows!.map((r) {
                  final modelId = (r.modelId ?? '').toString();
                  final count = (r.stockCount ?? '').toString();
                  final label = (r.label ?? '').toString();

                  return Padding(
                    padding: const EdgeInsets.symmetric(vertical: 6),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          label.isNotEmpty ? label : '(no label)',
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'modelId: ${modelId.isNotEmpty ? modelId : '(empty)'}   stock: $count',
                          style: Theme.of(context).textTheme.labelSmall,
                        ),
                      ],
                    ),
                  );
                }),
            ] else ...[
              if (invErr != null && invErr.trim().isNotEmpty) ...[
                const SizedBox(height: 10),
                Text(
                  'inventory error: $invErr',
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
