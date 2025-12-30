// frontend/sns/lib/features/avatar/presentation/page/avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class AvatarPage extends StatelessWidget {
  const AvatarPage({super.key, this.from});

  /// ✅ router.dart から渡される「遷移元」
  final String? from;

  String _s(String? v) => (v ?? '').trim();

  String _displayNameFor(User u) {
    final dn = _s(u.displayName);
    if (dn.isNotEmpty) return dn;

    final email = _s(u.email);
    if (email.isNotEmpty) return email.split('@').first;

    return 'My Profile';
  }

  String _currentUri(BuildContext context) {
    return GoRouterState.of(context).uri.toString();
  }

  String _effectiveFrom(BuildContext context) {
    final v = _s(from);
    if (v.isNotEmpty) return v;
    return _currentUri(context);
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      // ✅ photoURL / displayName 更新を拾う
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;

        // 念のため：未ログインならログインへ誘導
        if (user == null) {
          final backTo = _effectiveFrom(context);
          final loginUri = Uri(
            path: '/login',
            queryParameters: {'from': backTo},
          );
          return Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.lock_outline, size: 40),
                    const SizedBox(height: 12),
                    Text(
                      'Sign in required',
                      style: Theme.of(context).textTheme.titleLarge,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 8),
                    const Text(
                      'プロフィールを表示するにはサインインが必要です。',
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: () => context.go(loginUri.toString()),
                      child: const Text('Sign in'),
                    ),
                  ],
                ),
              ),
            ),
          );
        }

        final photoUrl = _s(user.photoURL);
        final name = _displayNameFor(user);

        // Instagram 風：上部に丸アイコン + 名前、中央寄せのカードっぽい余白
        return Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560),
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      CircleAvatar(
                        radius: 44,
                        backgroundColor: Theme.of(
                          context,
                        ).colorScheme.surfaceContainerHighest,
                        backgroundImage: photoUrl.isNotEmpty
                            ? NetworkImage(photoUrl)
                            : null,
                        child: photoUrl.isEmpty
                            ? Icon(
                                Icons.person,
                                size: 44,
                                color: Theme.of(
                                  context,
                                ).colorScheme.onSurfaceVariant,
                              )
                            : null,
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              name,
                              style: Theme.of(context).textTheme.titleLarge,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                            const SizedBox(height: 6),
                            Text(
                              'Profile',
                              style: Theme.of(context).textTheme.bodySmall
                                  ?.copyWith(
                                    color: Theme.of(
                                      context,
                                    ).colorScheme.onSurfaceVariant,
                                  ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  const Divider(height: 1),
                  const SizedBox(height: 12),

                  // ✅ ここから先は次のステップで拡張（bio/link/grid等）
                  Text(
                    '（次）ここにプロフィール文・リンク・投稿グリッドを追加します。',
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),

                  const SizedBox(height: 24),
                  OutlinedButton(
                    onPressed: () {
                      // いまは既存の作成/編集画面へ
                      final here = _currentUri(context);
                      final uri = Uri(
                        path: '/avatar-create',
                        queryParameters: {'from': here},
                      );
                      context.go(uri.toString());
                    },
                    child: const Text('Edit profile'),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}
