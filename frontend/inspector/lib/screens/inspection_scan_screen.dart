import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../services/product_api.dart';

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

    // String? を一旦受けて null を弾いたあと、non-null な String に確定させる
    final maybeProductId = _extractProductId(raw);
    if (maybeProductId == null || maybeProductId.isEmpty) return;
    final String productId = maybeProductId;

    setState(() {
      _processing = true;
      _lastProductId = productId;
    });

    // 合否ダイアログを表示
    final result = await showDialog<String>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('検品結果を送信'),
          content: Text('productId: $productId\n\n検品結果を選択してください。'),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop('cancel'),
              child: const Text('キャンセル'),
            ),
            TextButton(
              onPressed: () => Navigator.of(context).pop('fail'),
              child: const Text('不合格'),
            ),
            ElevatedButton(
              onPressed: () => Navigator.of(context).pop('pass'),
              child: const Text('合格'),
            ),
          ],
        );
      },
    );

    if (!mounted) return;

    if (result == 'pass' || result == 'fail') {
      // ここで non-null に確定させる
      final String decidedResult = result!;

      try {
        await ProductApi.submitInspection(
          productId: productId, // String
          result: decidedResult, // String に確定
        );
        if (!mounted) return;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('検品結果を送信しました（$decidedResult）')));
      } catch (e) {
        if (!mounted) return;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('送信に失敗しました: $e')));
      }
    }

    // 少し待ってから再スキャン可能にする
    await Future.delayed(const Duration(seconds: 1));
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
                    // withOpacity は非推奨なので withValues を使用
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
