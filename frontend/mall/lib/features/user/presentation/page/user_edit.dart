//frontend\sns\lib\features\user\presentation\page\user_edit.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../../app/shell/presentation/components/header.dart';

class UserEditPage extends StatefulWidget {
  const UserEditPage({super.key, this.from});

  /// optional back route (もし router から渡したい場合用)
  final String? from;

  @override
  State<UserEditPage> createState() => _UserEditPageState();
}

class _UserEditPageState extends State<UserEditPage> {
  final _displayNameCtrl = TextEditingController();
  final _photoUrlCtrl = TextEditingController();

  bool _saving = false;
  String? _msg;

  @override
  void initState() {
    super.initState();
    final user = FirebaseAuth.instance.currentUser;
    _displayNameCtrl.text = (user?.displayName ?? '').trim();
    _photoUrlCtrl.text = (user?.photoURL ?? '').trim();
  }

  @override
  void dispose() {
    _displayNameCtrl.dispose();
    _photoUrlCtrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  String _backTo(BuildContext context) {
    // ✅ router から渡されなくても、URL query (?from=...) から拾える
    final qpFrom = GoRouterState.of(context).uri.queryParameters['from'];
    final f = _s(widget.from ?? qpFrom);
    if (f.isNotEmpty) return f;
    return '/avatar';
  }

  bool _isValidHttpUrlOrEmpty(String v0) {
    final v = _s(v0);
    if (v.isEmpty) return true;
    final uri = Uri.tryParse(v);
    if (uri == null) return false;
    if (!uri.hasScheme) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;
    return uri.host.isNotEmpty;
  }

  Future<void> _save() async {
    setState(() {
      _saving = true;
      _msg = null;
    });

    try {
      final user = FirebaseAuth.instance.currentUser;
      if (user == null) {
        if (mounted) {
          setState(() => _msg = 'サインインが必要です。');
        }
        return;
      }

      final displayName = _s(_displayNameCtrl.text);
      final photoUrl = _s(_photoUrlCtrl.text);

      if (!_isValidHttpUrlOrEmpty(photoUrl)) {
        if (mounted) {
          setState(() => _msg = 'photoURL は http(s) のURLを入力してください。');
        }
        return;
      }

      // null を渡すと削除できる
      await user.updateDisplayName(displayName.isEmpty ? null : displayName);
      await user.updatePhotoURL(photoUrl.isEmpty ? null : photoUrl);

      await user.reload();

      if (!mounted) return;
      setState(() => _msg = 'ユーザー情報を更新しました。');
    } catch (e) {
      if (!mounted) return;
      setState(() => _msg = e.toString());
    } finally {
      // ✅ finally で return しない（lint回避）
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = FirebaseAuth.instance.currentUser;

    final photoUrl = _s(user?.photoURL);
    final email = _s(user?.email);
    final uid = _s(user?.uid);

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'Account',
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
                        Row(
                          children: [
                            CircleAvatar(
                              radius: 28,
                              backgroundImage: photoUrl.isNotEmpty
                                  ? NetworkImage(photoUrl)
                                  : null,
                              child: photoUrl.isEmpty
                                  ? const Icon(Icons.person)
                                  : null,
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    email.isNotEmpty ? email : 'signed-in',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.titleMedium,
                                  ),
                                  const SizedBox(height: 2),
                                  Text(
                                    'uid: ${uid.isNotEmpty ? uid : '-'}',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.bodySmall,
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 16),

                        TextField(
                          controller: _displayNameCtrl,
                          decoration: const InputDecoration(
                            labelText: 'displayName（任意）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _photoUrlCtrl,
                          decoration: const InputDecoration(
                            labelText: 'photoURL（任意）',
                            hintText: 'https://...',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),

                        ElevatedButton(
                          onPressed: _saving ? null : _save,
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('保存'),
                        ),

                        if ((_msg ?? '').trim().isNotEmpty) ...[
                          const SizedBox(height: 12),
                          Container(
                            padding: const EdgeInsets.all(12),
                            decoration: BoxDecoration(
                              color: Theme.of(context)
                                  .colorScheme
                                  .surfaceContainerHighest
                                  .withValues(alpha: 0.55),
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(_msg!),
                          ),
                        ],
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
