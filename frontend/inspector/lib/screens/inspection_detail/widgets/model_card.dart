// frontend/inspector/lib/screens/inspection_detail/widgets/model_card.dart
import 'package:flutter/material.dart';

import '../../../models/inspector_product_detail.dart';

class ModelCard extends StatelessWidget {
  final InspectorProductDetail detail;

  const ModelCard({super.key, required this.detail});

  @override
  Widget build(BuildContext context) {
    final entries = detail.measurements.entries.toList()
      ..sort((a, b) => a.key.compareTo(b.key));

    final colorInt = (() {
      final v = detail.color.rgb;
      if ((v & 0xFF000000) == 0) {
        return 0xFF000000 | v;
      }
      return v;
    })();

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'モデル情報',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text('modelNumber: ${detail.modelNumber}'),
            if (detail.size.isNotEmpty) Text('サイズ: ${detail.size}'),
            const SizedBox(height: 8),
            Row(
              children: [
                const Text('カラー:'),
                const SizedBox(width: 8),
                Container(
                  width: 18,
                  height: 18,
                  decoration: BoxDecoration(
                    color: Color(colorInt),
                    borderRadius: BorderRadius.circular(4),
                    border: Border.all(color: Colors.grey.shade400),
                  ),
                ),
                const SizedBox(width: 8),
                Text(detail.color.name ?? ''),
              ],
            ),
            if (entries.isNotEmpty) ...[
              const SizedBox(height: 8),
              const Text('採寸値', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: entries
                    .map((e) => Chip(label: Text('${e.key}: ${e.value}')))
                    .toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
