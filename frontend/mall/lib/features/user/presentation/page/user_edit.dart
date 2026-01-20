// frontend\mall\lib\features\user\presentation\page\user_edit.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../../../../app/routing/navigation.dart';

class UserEditPage extends StatefulWidget {
  const UserEditPage({super.key, this.tab});

  /// 初期表示タブ（router から渡す）
  /// 例: 'email', 'password'
  final String? tab;

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

  /// Pattern B: URL の from を使わず、NavStore の returnTo を優先して戻る
  /// - returnTo が無ければ /avatar にフォールバック
  String _backTo(BuildContext context) {
    final to = NavStore.I.consumeReturnTo().trim();
    if (to.isNotEmpty) return to;
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
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  int _initialTabIndex() {
    switch (_s(widget.tab)) {
      case 'password':
        return 1;
      case 'email':
      default:
        return 0;
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = FirebaseAuth.instance.currentUser;

    final photoUrl = _s(user?.photoURL);
    final email = _s(user?.email);
    final uid = _s(user?.uid);

    return DefaultTabController(
      length: 2,
      initialIndex: _initialTabIndex(),
      child: Scaffold(
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
              const Material(
                child: TabBar(
                  tabs: [
                    Tab(text: 'Email'),
                    Tab(text: 'Password'),
                  ],
                ),
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    _ProfileTab(
                      displayNameCtrl: _displayNameCtrl,
                      photoUrlCtrl: _photoUrlCtrl,
                      saving: _saving,
                      msg: _msg,
                      photoUrl: photoUrl,
                      email: email,
                      uid: uid,
                      onSave: _save,
                    ),
                    _PasswordTab(msg: _msg),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// ------------------------------------------------------------
/// 既存の「ユーザー情報編集」UI を Email タブ側（ここでは Profile として）に配置
class _ProfileTab extends StatelessWidget {
  const _ProfileTab({
    required this.displayNameCtrl,
    required this.photoUrlCtrl,
    required this.saving,
    required this.msg,
    required this.photoUrl,
    required this.email,
    required this.uid,
    required this.onSave,
  });

  final TextEditingController displayNameCtrl;
  final TextEditingController photoUrlCtrl;
  final bool saving;
  final String? msg;

  final String photoUrl;
  final String email;
  final String uid;

  final Future<void> Function() onSave;

  @override
  Widget build(BuildContext context) {
    return Center(
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
                    child: photoUrl.isEmpty ? const Icon(Icons.person) : null,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          email.isNotEmpty ? email : 'signed-in',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'uid: ${uid.isNotEmpty ? uid : '-'}',
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: displayNameCtrl,
                decoration: const InputDecoration(
                  labelText: 'displayName（任意）',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: photoUrlCtrl,
                decoration: const InputDecoration(
                  labelText: 'photoURL（任意）',
                  hintText: 'https://...',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: saving ? null : () => onSave(),
                child: saving
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('保存'),
              ),
              if ((msg ?? '').trim().isNotEmpty) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surfaceContainerHighest
                        .withValues(alpha: 0.55),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(msg!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// ------------------------------------------------------------
/// Password タブ（既存の PasswordUpdateBody などがあるなら差し替えてください）
class _PasswordTab extends StatelessWidget {
  const _PasswordTab({required this.msg});
  final String? msg;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text('Password', style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 8),
              const Text(
                'ここにパスワード変更UI（既存の password_update_body.dart 等）を配置してください。',
              ),
              if ((msg ?? '').trim().isNotEmpty) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surfaceContainerHighest
                        .withValues(alpha: 0.55),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(msg!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
