// lib/screens/inspection_scan_screen.dart
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import 'inspection_detail_screen.dart'; // ★ 追加：詳細画面をインポート

class InspectionScanScreen extends StatefulWidget {
  const InspectionScanScreen({super.key});

  @override
  State<InspectionScanScreen> createState() => _InspectionScanScreenState();
}

class _InspectionScanScreenState extends State<InspectionScanScreen> {
  final MobileScannerController _scannerController = MobileScannerController();
  bool _processing = false;
  String? _lastProductId;

  @override
  void dispose() {
    _scannerController.dispose();
    super.dispose();
  }

  /// QR 内容から productId を抽出
  /// - 「ただのID」の場合はそのまま
  /// - URL の場合は /products/{id} をパース
  String? _extractProductId(String raw) {
    final trimmed = raw.trim();
    if (trimmed.isEmpty) return null;

    // URL の場合
    Uri? uri;
    try {
      uri = Uri.parse(trimmed);
    } catch (_) {
      uri = null;
    }

    if (uri != null && uri.hasScheme) {
      final segments = uri.pathSegments;
      final idx = segments.indexOf('products');
      if (idx != -1 && idx + 1 < segments.length) {
        return segments[idx + 1];
      }
    }

    // URL でなければ「そのままID」とみなす
    return trimmed;
  }

  Future<void> _onDetect(BarcodeCapture capture) async {
    if (_processing) return;

    final raw = capture.barcodes.first.rawValue;
    if (raw == null || raw.isEmpty) return;

    final maybeProductId = _extractProductId(raw);
    if (maybeProductId == null || maybeProductId.isEmpty) return;
    final String productId = maybeProductId;

    setState(() {
      _processing = true;
      _lastProductId = productId;
    });

    // ★ ダイアログは廃止し、詳細画面へ遷移
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => InspectionDetailScreen(productId: productId),
      ),
    );

    // 戻ってきたら再スキャン可能にする
    if (mounted) {
      setState(() {
        _processing = false;
      });
    }
  }

  Future<void> _logout() async {
    await FirebaseAuth.instance.signOut();
  }

  @override
  Widget build(BuildContext context) {
    final user = FirebaseAuth.instance.currentUser;

    return Scaffold(
      appBar: AppBar(
        title: const Text('検品スキャナー'),
        actions: [
          if (user != null)
            IconButton(
              onPressed: _logout,
              icon: const Icon(Icons.logout),
              tooltip: 'ログアウト',
            ),
        ],
      ),
      body: Column(
        children: [
          if (user != null)
            Padding(
              padding: const EdgeInsets.all(8),
              child: Text(
                'ログイン中: ${user.email ?? user.uid}',
                style: const TextStyle(fontSize: 12),
              ),
            ),
          Expanded(
            child: Stack(
              children: [
                MobileScanner(
                  controller: _scannerController,
                  onDetect: _onDetect,
                ),
                if (_processing)
                  Container(
                    color: Colors.black.withValues(alpha: 0.4),
                    child: const Center(child: CircularProgressIndicator()),
                  ),
              ],
            ),
          ),
          if (_lastProductId != null)
            Padding(
              padding: const EdgeInsets.all(8),
              child: Text(
                '最後に読み取った productId: $_lastProductId',
                style: const TextStyle(fontSize: 12),
              ),
            ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
