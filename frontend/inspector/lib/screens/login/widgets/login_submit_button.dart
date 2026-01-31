// frontend/inspector/lib/screens/login/widgets/login_submit_button.dart
import 'package:flutter/material.dart';

class LoginSubmitButton extends StatelessWidget {
  final bool loading;
  final VoidCallback? onPressed;

  const LoginSubmitButton({
    super.key,
    required this.loading,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton(
        onPressed: loading ? null : onPressed,
        child: loading
            ? const SizedBox(
                height: 18,
                width: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Text('ログイン'),
      ),
    );
  }
}
