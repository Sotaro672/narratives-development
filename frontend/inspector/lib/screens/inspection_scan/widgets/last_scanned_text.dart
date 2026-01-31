// lib/screens/inspection_scan/widgets/last_scanned_text.dart
import 'package:flutter/material.dart';

class LastScannedText extends StatelessWidget {
  final String productId;

  const LastScannedText({super.key, required this.productId});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Text(
        '最後に読み取った productId: $productId',
        style: const TextStyle(fontSize: 12),
      ),
    );
  }
}
