// frontend/inspector/lib/screens/inspection_detail/widgets/inspection_list_card.dart
import 'package:flutter/material.dart';

import '../../../models/inspector_product_detail.dart';
import '../utils/inspection_formatters.dart';

class InspectionListCard extends StatelessWidget {
  final InspectorProductDetail detail;

  const InspectionListCard({super.key, required this.detail});

  @override
  Widget build(BuildContext context) {
    final inspections = detail.inspections;
    if (inspections.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Text('検品履歴はまだありません。'),
      );
    }

    final int quantity = inspections.length;
    final int totalPassed = inspections
        .where((r) => r.inspectionResult == 'passed')
        .length;

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '検品履歴',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('modelNumber: ${detail.modelNumber}'),
                  Text('生産量: $quantity'),
                  Text('合格数: $totalPassed'),
                ],
              ),
            ),
            const Divider(height: 1),
            const SizedBox(height: 8),
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: inspections.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final item = inspections[index];
                final resultLabel = formatInspectionResultLabel(
                  item.inspectionResult,
                );
                final modelNumber = item.modelNumber ?? '';

                return ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  title: Text(
                    modelNumber.isNotEmpty
                        ? 'productId: ${item.productId} / modelNumber: $modelNumber'
                        : 'productId: ${item.productId}',
                    style: const TextStyle(fontSize: 13),
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('検査結果: $resultLabel'),
                      if (item.inspectedBy != null &&
                          item.inspectedBy!.isNotEmpty)
                        Text('検査者: ${item.inspectedBy}'),
                      if (item.inspectedAt != null)
                        Text('検査日時: ${item.inspectedAt}'),
                    ],
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
