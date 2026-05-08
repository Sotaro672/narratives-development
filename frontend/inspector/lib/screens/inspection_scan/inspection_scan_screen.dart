// lib/screens/inspection_scan/inspection_scan_screen.dart
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import 'actions/inspection_scan_actions.dart';
import 'widgets/scanner_view.dart';
import 'widgets/login_status_text.dart';
import 'widgets/last_scanned_text.dart';

class InspectionScanScreen extends StatefulWidget {
  const InspectionScanScreen({super.key});

  @override
  State<InspectionScanScreen> createState() => _InspectionScanScreenState();
}

class _InspectionScanScreenState extends State<InspectionScanScreen> {
  final _actions = InspectionScanActions();
  final MobileScannerController _scannerController = MobileScannerController();

  bool _processing = false;
  String? _lastProductId;

  @override
  void dispose() {
    _scannerController.dispose();
    super.dispose();
  }

  Future<void> _logout() async {
    await FirebaseAuth.instance.signOut();
  }

  void _startProcessing(String productId) {
    setState(() {
      _processing = true;
      _lastProductId = productId;
    });
  }

  void _finishProcessing() {
    setState(() {
      _processing = false;
    });
  }

  Future<void> _onDetect(BarcodeCapture capture) {
    return _actions.handleDetect(
      context: context,
      processing: _processing,
      onStartProcessing: _startProcessing,
      onFinishProcessing: _finishProcessing,
      capture: capture,
    );
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
            LoginStatusText(text: 'ログイン中: ${user.email ?? user.uid}'),
          Expanded(
            child: ScannerView(
              controller: _scannerController,
              onDetect: _onDetect,
              processing: _processing,
            ),
          ),
          if (_lastProductId != null)
            LastScannedText(productId: _lastProductId!),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
