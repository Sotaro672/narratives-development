// frontend/inspector/lib/screens/inspection_detail/widgets/action_section.dart
import 'package:flutter/material.dart';

import '../../../models/inspector_product_detail.dart';
import '../utils/inspection_formatters.dart';
import 'inspection_list_card.dart';

class ActionSection extends StatelessWidget {
  final InspectorProductDetail detail;
  final bool submitting;

  final VoidCallback onContinue;
  final Future<void> Function(InspectorProductDetail detail, String result)
  onSubmitResult;
  final Future<void> Function(String productionId) onComplete;

  const ActionSection({
    super.key,
    required this.detail,
    required this.submitting,
    required this.onContinue,
    required this.onSubmitResult,
    required this.onComplete,
  });

  @override
  Widget build(BuildContext context) {
    final nowStatus = detail.inspectionResult;
    final nowStatusLabel = nowStatus.isEmpty
        ? '未検査'
        : formatInspectionResultLabel(nowStatus);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (nowStatus.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                '現在の検品ステータス: $nowStatusLabel',
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),

          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: submitting
                      ? null
                      : () => onSubmitResult(detail, 'failed'),
                  child: const Text('不合格'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: submitting
                      ? null
                      : () => onSubmitResult(detail, 'passed'),
                  child: const Text('合格'),
                ),
              ),
            ],
          ),

          const SizedBox(height: 16),
          InspectionListCard(detail: detail),
          const SizedBox(height: 16),

          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: submitting ? null : onContinue,
                  child: const Text('検品を続ける'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton(
                  onPressed: submitting
                      ? null
                      : () => onComplete(detail.productionId),
                  child: const Text('検品を完了する'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
