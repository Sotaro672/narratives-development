// lib/screens/inspection_scan/widgets/scanner_view.dart
import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

class ScannerView extends StatelessWidget {
  final MobileScannerController controller;
  final void Function(BarcodeCapture capture) onDetect;
  final bool processing;

  const ScannerView({
    super.key,
    required this.controller,
    required this.onDetect,
    required this.processing,
  });

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        MobileScanner(controller: controller, onDetect: onDetect),
        if (processing)
          Container(
            color: Colors.black.withValues(alpha: 0.4),
            child: const Center(child: CircularProgressIndicator()),
          ),
      ],
    );
  }
}
