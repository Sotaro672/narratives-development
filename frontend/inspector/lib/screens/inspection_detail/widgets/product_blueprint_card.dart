// frontend/inspector/lib/screens/inspection_detail/widgets/product_blueprint_card.dart
import 'package:flutter/material.dart';

import '../../../models/inspector_product_detail.dart';

class ProductBlueprintCard extends StatelessWidget {
  final InspectorProductDetail detail;

  const ProductBlueprintCard({super.key, required this.detail});

  @override
  Widget build(BuildContext context) {
    final bp = detail.productBlueprint;

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '商品設計情報',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text('商品名: ${bp.productName}'),
            Text('ブランド名: ${bp.brandName}'),
            Text('会社名: ${bp.companyName}'),
            Text('アイテム種別: ${bp.itemType}'),
            if (bp.fit.isNotEmpty) Text('フィット: ${bp.fit}'),
            if (bp.material.isNotEmpty) Text('素材: ${bp.material}'),
            Text('重さ: ${bp.weight}'),
            const SizedBox(height: 8),
            if (bp.qualityAssurance.isNotEmpty) ...[
              const Text(
                '品質表示・注意事項',
                style: TextStyle(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 4),
              Wrap(
                spacing: 8,
                runSpacing: 4,
                children: bp.qualityAssurance
                    .map((q) => Chip(label: Text(q)))
                    .toList(),
              ),
            ],
            const SizedBox(height: 8),
            Text('タグ種別: ${bp.productIdTagType}'),
            if (bp.assigneeId.isNotEmpty) Text('担当者ID: ${bp.assigneeId}'),
          ],
        ),
      ),
    );
  }
}
