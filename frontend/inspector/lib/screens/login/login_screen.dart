// frontend/inspector/lib/screens/login/login_screen.dart
import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';

import 'widgets/login_error_text.dart';
import 'widgets/login_form.dart';
import 'widgets/login_submit_button.dart';

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
            LoginForm(
              emailController: _emailController,
              passwordController: _passwordController,
            ),
            const SizedBox(height: 16),
            LoginErrorText(error: _error),
            const SizedBox(height: 16),
            LoginSubmitButton(loading: _loading, onPressed: _login),
          ],
        ),
      ),
    );
  }
}
