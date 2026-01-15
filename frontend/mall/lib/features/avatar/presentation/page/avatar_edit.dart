//frontend\mall\lib\features\avatar\presentation\page\avatar_edit.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../../app/shell/presentation/components/header.dart';

/// Avatar edit page (placeholder for now).
/// - Accepts `from` so router can pass it safely.
/// - Later you can replace the body with real edit UI (icon/name/profile/link).
class AvatarEditPage extends StatelessWidget {
  const AvatarEditPage({super.key, this.from});

  /// optional back route (router から渡す用)
  final String? from;

  String _s(String? v) => (v ?? '').trim();

  String _backTo(BuildContext context) {
    // URL query (?from=...) でも拾える
    final qpFrom = GoRouterState.of(context).uri.queryParameters['from'];
    final f = _s(from ?? qpFrom);
    if (f.isNotEmpty) return f;
    return '/avatar';
  }

  @override
  Widget build(BuildContext context) {
    final user = FirebaseAuth.instance.currentUser;
    final email = _s(user?.email);
    final uid = _s(user?.uid);

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'Edit Avatar',
              showBack: true,
              backTo: _backTo(context),
              actions: const [],
              onTapTitle: () => context.go('/'),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          '（準備中）ここにアバター編集UIを実装します。',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 12),
                        Container(
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: Theme.of(context)
                                .colorScheme
                                .surfaceContainerHighest
                                .withValues(alpha: 0.45),
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: Text(
                            'signed-in: ${email.isNotEmpty ? email : '-'}\nuid: ${uid.isNotEmpty ? uid : '-'}',
                          ),
                        ),
                        const SizedBox(height: 16),

                        // いまある既存ページへ誘導（本実装までの暫定）
                        OutlinedButton(
                          onPressed: () {
                            final uri = Uri(
                              path: '/avatar',
                              queryParameters: {'from': _backTo(context)},
                            );
                            context.go(uri.toString());
                          },
                          child: const Text('プロフィールへ戻る'),
                        ),
                        const SizedBox(height: 8),
                        ElevatedButton(
                          onPressed: () {
                            // 既存の AvatarCreatePage を編集導線として使う（暫定）
                            final uri = Uri(
                              path: '/avatar-create',
                              queryParameters: {'from': _backTo(context)},
                            );
                            context.go(uri.toString());
                          },
                          child: const Text('アバター作成/更新へ（/avatar-create）'),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
