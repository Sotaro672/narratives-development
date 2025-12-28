// frontend/sns/lib/features/auth/presentation/page/avatar_create.dart
import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';

/// ✅ アバター作成（雛形）
/// - アバターアイコン画像（現状はダミー選択）
/// - アバター名
/// - プロフィール
/// - 外部リンク
///
/// ※ 実運用では image_picker 等で画像選択し、署名付きURL→GCSアップロード→URL保存 の流れにする想定。
class AvatarCreatePage extends StatefulWidget {
  const AvatarCreatePage({super.key, this.from});

  /// optional back route
  final String? from;

  @override
  State<AvatarCreatePage> createState() => _AvatarCreatePageState();
}

class _AvatarCreatePageState extends State<AvatarCreatePage> {
  final _nameCtrl = TextEditingController();
  final _profileCtrl = TextEditingController();
  final _linkCtrl = TextEditingController();

  Uint8List? _iconBytes; // ダミー
  bool _saving = false;
  String? _msg;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _profileCtrl.dispose();
    _linkCtrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  String _backTo() {
    final from = _s(widget.from);
    if (from.isNotEmpty) return from;
    return '/billing-address';
  }

  bool get _canSave {
    if (_saving) return false;
    if (_s(_nameCtrl.text).isEmpty) return false;
    // 画像を必須にする場合:
    // if (_iconBytes == null) return false;
    return true;
  }

  bool _isValidUrlOrEmpty(String s) {
    final v = _s(s);
    if (v.isEmpty) return true;
    final uri = Uri.tryParse(v);
    if (uri == null) return false;
    if (!uri.hasScheme) return false;
    if (uri.scheme != 'http' && uri.scheme != 'https') return false;
    return uri.host.isNotEmpty;
  }

  Future<void> _pickIconDummy() async {
    if (!mounted) return;
    setState(() {
      _iconBytes = Uint8List.fromList(List<int>.generate(64, (i) => i));
      _msg = 'アイコン画像を選択しました（ダミー）。';
    });
  }

  Future<void> _saveDummy() async {
    final link = _s(_linkCtrl.text);
    if (!_isValidUrlOrEmpty(link)) {
      if (!mounted) return;
      setState(() {
        _msg = '外部リンクは http(s) のURLを入力してください。';
      });
      return;
    }

    if (!mounted) return;
    setState(() {
      _saving = true;
      _msg = null;
    });

    Object? caught;
    try {
      await Future<void>.delayed(const Duration(milliseconds: 700));

      if (!mounted) return;

      setState(() {
        _msg = 'アバターを作成しました（ダミー）。';
      });

      // ✅ 保存後は Home に戻る
      context.go('/');
    } catch (e) {
      caught = e;
      if (mounted) {
        setState(() {
          _msg = e.toString();
        });
      }
    } finally {
      // ✅ finally 内で return しない（linter対策）
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }

    // caught は将来ログ用途などに使える（未使用でもOK）
    // ignore: unused_local_variable
    final _ = caught;
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _backTo();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'アバター作成',
              showBack: true,
              backTo: backTo,
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
                        const _InfoBox(
                          kind: _InfoKind.info,
                          text:
                              'アバター情報を登録してください。\n※ 実運用ではアイコン画像はアップロードしてURLを保存します。',
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'アバターアイコン画像',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        _IconPickerCard(
                          bytes: _iconBytes,
                          onPick: _pickIconDummy,
                          onClear: () => setState(() => _iconBytes = null),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'アバター名',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _nameCtrl,
                          textInputAction: TextInputAction.next,
                          decoration: const InputDecoration(
                            labelText: 'アバター名',
                            border: OutlineInputBorder(),
                            hintText: '例: sotaro',
                          ),
                          onChanged: (_) => setState(() {}),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'プロフィール',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _profileCtrl,
                          maxLines: 4,
                          decoration: const InputDecoration(
                            labelText: 'プロフィール',
                            border: OutlineInputBorder(),
                            hintText: '例: 私は○○のクリエイターです。',
                          ),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          '外部リンク',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _linkCtrl,
                          keyboardType: TextInputType.url,
                          decoration: const InputDecoration(
                            labelText: '外部リンク（任意）',
                            border: OutlineInputBorder(),
                            hintText: '例: https://example.com',
                          ),
                        ),
                        const SizedBox(height: 20),

                        ElevatedButton(
                          onPressed: _canSave ? _saveDummy : null,
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('このアバターを保存する'),
                        ),

                        if (_msg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind:
                                _msg!.contains('作成しました') ||
                                    _msg!.contains('選択しました')
                                ? _InfoKind.ok
                                : _InfoKind.error,
                            text: _msg!,
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

class _IconPickerCard extends StatelessWidget {
  const _IconPickerCard({
    required this.bytes,
    required this.onPick,
    required this.onClear,
  });

  final Uint8List? bytes;
  final VoidCallback onPick;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outlineVariant.withValues(alpha: 0.6)),
      ),
      child: Row(
        children: [
          _AvatarPreview(bytes: bytes),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  bytes == null ? '未選択' : '選択済み（ダミー）',
                  style: Theme.of(context).textTheme.bodyLarge,
                ),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    OutlinedButton(
                      onPressed: onPick,
                      child: const Text('画像を選択する'),
                    ),
                    if (bytes != null)
                      TextButton(onPressed: onClear, child: const Text('削除')),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _AvatarPreview extends StatelessWidget {
  const _AvatarPreview({required this.bytes});

  final Uint8List? bytes;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    if (bytes == null) {
      return CircleAvatar(
        radius: 28,
        backgroundColor: scheme.surfaceContainerHighest,
        child: Icon(Icons.person, color: scheme.onSurfaceVariant),
      );
    }

    // ダミー：実際の画像 bytes ではないのでプレースホルダ表示
    return CircleAvatar(
      radius: 28,
      backgroundColor: scheme.primaryContainer,
      child: Icon(Icons.image, color: scheme.onPrimaryContainer),
    );
  }
}

enum _InfoKind { info, ok, error }

class _InfoBox extends StatelessWidget {
  const _InfoBox({required this.kind, required this.text});

  final _InfoKind kind;
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    late final Color bg;
    switch (kind) {
      case _InfoKind.ok:
        bg = scheme.primaryContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.error:
        bg = scheme.errorContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.info:
        bg = scheme.surfaceContainerHighest.withValues(alpha: 0.55);
        break;
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text, style: Theme.of(context).textTheme.bodyMedium),
    );
  }
}
