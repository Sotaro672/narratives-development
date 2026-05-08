// frontend/inspector/lib/screens/login/widgets/login_form.dart
import 'package:flutter/material.dart';

class LoginForm extends StatelessWidget {
  final TextEditingController emailController;
  final TextEditingController passwordController;

  const LoginForm({
    super.key,
    required this.emailController,
    required this.passwordController,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(
          controller: emailController,
          decoration: const InputDecoration(labelText: 'メールアドレス'),
          keyboardType: TextInputType.emailAddress,
        ),
        const SizedBox(height: 12),
        TextField(
          controller: passwordController,
          decoration: const InputDecoration(labelText: 'パスワード'),
          obscureText: true,
        ),
      ],
    );
  }
}
