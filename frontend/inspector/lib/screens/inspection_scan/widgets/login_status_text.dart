// lib/screens/inspection_scan/widgets/login_status_text.dart
import 'package:flutter/material.dart';

class LoginStatusText extends StatelessWidget {
  final String text;

  const LoginStatusText({super.key, required this.text});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Text(text, style: const TextStyle(fontSize: 12)),
    );
  }
}
