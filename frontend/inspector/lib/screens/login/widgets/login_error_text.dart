// frontend/inspector/lib/screens/login/widgets/login_error_text.dart
import 'package:flutter/material.dart';

class LoginErrorText extends StatelessWidget {
  final String? error;

  const LoginErrorText({super.key, required this.error});

  @override
  Widget build(BuildContext context) {
    if (error == null) return const SizedBox.shrink();
    return Text(error!, style: const TextStyle(color: Colors.red));
  }
}
