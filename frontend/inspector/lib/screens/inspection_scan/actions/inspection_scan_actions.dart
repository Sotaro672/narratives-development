// lib/screens/inspection_scan/actions/inspection_scan_actions.dart
import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../inspection_detail/inspection_detail_screen.dart';
import '../utils/product_id_extractor.dart';

class InspectionScanActions {
  Future<void> handleDetect({
    required BuildContext context,
    required bool processing,
    required void Function(String productId) onStartProcessing,
    required VoidCallback onFinishProcessing,
    required BarcodeCapture capture,
  }) async {
    if (processing) return;

    final raw = capture.barcodes.first.rawValue;
    if (raw == null || raw.isEmpty) return;

    final productId = extractProductIdFromQrRaw(raw);
    if (productId == null || productId.isEmpty) return;

    onStartProcessing(productId);

    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => InspectionDetailScreen(productId: productId),
      ),
    );

    if (context.mounted) {
      onFinishProcessing();
    }
  }
}
