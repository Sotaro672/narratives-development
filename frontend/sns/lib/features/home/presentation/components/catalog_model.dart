// frontend\sns\lib\features\home\presentation\components\catalog_model.dart
import 'package:flutter/material.dart';

import '../../../model/infrastructure/model_repository_http.dart';

class CatalogModelCard extends StatelessWidget {
  const CatalogModelCard({
    super.key,
    required this.productBlueprintId,
    required this.models,
    required this.modelError,
  });

  final String productBlueprintId;
  final List<ModelVariationDTO>? models;
  final String? modelError;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Model', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),

            _KeyValueRow(
              label: 'productBlueprintId',
              value: productBlueprintId.isNotEmpty
                  ? productBlueprintId
                  : '(unknown)',
            ),

            const SizedBox(height: 10),

            if (models != null) ...[
              if (models!.isEmpty)
                Text('(empty)', style: Theme.of(context).textTheme.bodyMedium)
              else
                ...models!.map((v) {
                  final mId = v.id.trim();

                  final titleParts = <String>[
                    v.modelNumber.trim(),
                    v.size.trim(),
                    v.color.name.trim(),
                  ].where((s) => s.isNotEmpty).toList();

                  final title = titleParts.isNotEmpty
                      ? titleParts.join(' / ')
                      : '(empty)';

                  return Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: Theme.of(context).textTheme.bodyLarge,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'modelId: ${mId.isNotEmpty ? mId : '(empty)'}',
                          style: Theme.of(context).textTheme.labelSmall,
                        ),
                        if (v.measurements.isNotEmpty) ...[
                          const SizedBox(height: 6),
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: v.measurements.entries.map((e) {
                              return Chip(
                                label: Text('${e.key}: ${e.value}'),
                                visualDensity: VisualDensity.compact,
                              );
                            }).toList(),
                          ),
                        ],
                      ],
                    ),
                  );
                }),
            ] else ...[
              if ((modelError ?? '').trim().isNotEmpty)
                Text(
                  'model error: ${modelError!.trim()}',
                  style: Theme.of(context).textTheme.labelSmall,
                )
              else
                Text(
                  'model is not loaded',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
            ],
          ],
        ),
      ),
    );
  }
}

/// ✅ 依存を増やしたくないため、このファイル内に閉じる
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
