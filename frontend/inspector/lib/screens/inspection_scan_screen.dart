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
  /// - URL (http/https) の場合は最後の path segment を ID とみなす
  ///   例: https://narratives.jp/oQtZOWW2OKFKvHIo0YsQ → oQtZOWW2OKFKvHIo0YsQ
  String? _extractProductId(String raw) {
    final trimmed = raw.trim();
    if (trimmed.isEmpty) return null;

    // まずは Uri.parse で正規の URL として扱えるか確認
    try {
      final uri = Uri.parse(trimmed);
      if (uri.scheme == 'http' || uri.scheme == 'https') {
        final segments = uri.pathSegments.where((s) => s.isNotEmpty).toList();
        if (segments.isNotEmpty) {
          return segments.last;
        }
      }
    } catch (_) {
      // 無視してフォールバックへ
    }

    // まだ URL っぽい文字列が残っている場合のフォールバック
    if (trimmed.contains('https://') || trimmed.contains('http://')) {
      final lastSlash = trimmed.lastIndexOf('/');
      if (lastSlash != -1 && lastSlash + 1 < trimmed.length) {
        return trimmed.substring(lastSlash + 1);
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
