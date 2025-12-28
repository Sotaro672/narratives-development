// frontend/sns/lib/features/auth/presentation/page/avatar_create.dart
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_avatar_create.dart';

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
  late final UseAvatarCreate _vm;

  @override
  void initState() {
    super.initState();
    _vm = UseAvatarCreate(from: widget.from);
    _vm.addListener(_onVmChanged);
  }

  void _onVmChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _vm.backTo();

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
                          bytes: _vm.iconBytes,
                          onPick: _vm.pickIconDummy,
                          onClear: _vm.clearIcon,
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'アバター名',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _vm.nameCtrl,
                          textInputAction: TextInputAction.next,
                          decoration: const InputDecoration(
                            labelText: 'アバター名',
                            border: OutlineInputBorder(),
                            hintText: '例: sotaro',
                          ),
                          onChanged: (_) => _vm.onNameChanged(),
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'プロフィール',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _vm.profileCtrl,
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
                          controller: _vm.linkCtrl,
                          keyboardType: TextInputType.url,
                          decoration: const InputDecoration(
                            labelText: '外部リンク（任意）',
                            border: OutlineInputBorder(),
                            hintText: '例: https://example.com',
                          ),
                        ),
                        const SizedBox(height: 20),

                        ElevatedButton(
                          onPressed: _vm.canSave
                              ? () => _vm.saveDummy(context)
                              : null,
                          child: _vm.saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('このアバターを保存する'),
                        ),

                        if (_vm.msg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind:
                                _vm.msg!.contains('作成しました') ||
                                    _vm.msg!.contains('選択しました')
                                ? _InfoKind.ok
                                : _InfoKind.error,
                            text: _vm.msg!,
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
  final Future<void> Function() onPick;
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
                      onPressed: () => onPick(),
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
