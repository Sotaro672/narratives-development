//frontend\mall\lib\features\auth\presentation\page\avatar_create.dart
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/shell/presentation/components/header.dart';
import '../hook/use_avatar_create.dart';

/// ✅ アバター作成
/// - アバターアイコン画像（Web対応：ファイル選択→bytes保持→プレビュー）
/// - アバター名
/// - プロフィール
/// - 外部リンク
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

  Future<void> _onSave() async {
    final ok = await _vm.save();
    if (!mounted) return;

    if (ok) {
      context.go('/');
    }
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
                              'アバター情報を登録してください。\n※ アイコン画像は「選択→プレビュー」まで実装済みです。アップロード連携は次のステップで行います。',
                        ),
                        const SizedBox(height: 16),

                        Text(
                          'アバターアイコン画像',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        _IconPickerCard(
                          bytes: _vm.iconBytes,
                          fileName: _vm.iconFileName,
                          onPick: _vm.pickIcon,
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
                          onPressed: _vm.canSave ? _onSave : null,
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
                            kind: _vm.isSuccessMessage
                                ? _InfoKind.ok
                                : _InfoKind.error,
                            text: _vm.msg!,
                          ),
                        ],

                        if (kDebugMode && _vm.iconBytes != null) ...[
                          const SizedBox(height: 12),
                          Text(
                            'debug: iconBytesLen=${_vm.iconBytes!.lengthInBytes}',
                            style: Theme.of(context).textTheme.bodySmall,
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
    required this.fileName,
    required this.onPick,
    required this.onClear,
  });

  final Uint8List? bytes;
  final String? fileName;
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
                  bytes == null ? '未選択' : '選択済み',
                  style: Theme.of(context).textTheme.bodyLarge,
                ),
                if (bytes != null && (fileName ?? '').trim().isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    fileName!,
                    style: Theme.of(context).textTheme.bodySmall,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
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

    return ClipOval(
      child: Image.memory(
        bytes!,
        width: 56,
        height: 56,
        fit: BoxFit.cover,
        errorBuilder: (_, __, ___) {
          return CircleAvatar(
            radius: 28,
            backgroundColor: scheme.primaryContainer,
            child: Icon(Icons.broken_image, color: scheme.onPrimaryContainer),
          );
        },
      ),
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
