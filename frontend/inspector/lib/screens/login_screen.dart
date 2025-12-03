// frontend/inspector/lib/screens/login_screen.dart
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

const backendBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

/// 検品結果をバックエンドに送信する関数
Future<void> patchInspection({
  required String productionId,
  required String productId,
  required String inspectionResult, // 'passed' / 'failed' / 'notYet' など
  required DateTime inspectedAt,
  String? status, // 'completed' など必要なら
}) async {
  final user = FirebaseAuth.instance.currentUser;
  if (user == null) {
    throw Exception('ログインしていません');
  }

  // 🔑 Firebase ID トークン取得
  final idToken = await user.getIdToken();

  final uri = Uri.parse('$backendBaseUrl/products/inspections');
  final resp = await http.patch(
    uri,
    headers: {
      'Authorization': 'Bearer $idToken', // ★ これが AuthMiddleware に渡る
      'Content-Type': 'application/json',
    },
    body: jsonEncode({
      'productionId': productionId,
      'productId': productId,
      'inspectionResult': inspectionResult,
      // inspectedBy はサーバ側で member.Service を使って決定する方針
      'inspectedAt': inspectedAt.toUtc().toIso8601String(),
      if (status != null) 'status': status,
    }),
  );

  if (resp.statusCode != 200) {
    throw Exception('検品更新に失敗しました: ${resp.statusCode} ${resp.body}');
  }
}

/// 検品アプリ用のログイン画面
class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      await FirebaseAuth.instance.signInWithEmailAndPassword(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      );
      // 成功すると authStateChanges 経由で Root が再ビルドされる
    } on FirebaseAuthException catch (e) {
      setState(() {
        _error = e.message ?? 'ログインに失敗しました';
      });
    } catch (_) {
      setState(() {
        _error = 'ログインに失敗しました';
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('検品ログイン')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            TextField(
              controller: _emailController,
              decoration: const InputDecoration(labelText: 'メールアドレス'),
              keyboardType: TextInputType.emailAddress,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _passwordController,
              decoration: const InputDecoration(labelText: 'パスワード'),
              obscureText: true,
            ),
            const SizedBox(height: 16),
            if (_error != null)
              Text(_error!, style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: _loading ? null : _login,
                child: _loading
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('ログイン'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
