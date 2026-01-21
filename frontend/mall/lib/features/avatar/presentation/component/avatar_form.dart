//frontend\mall\lib\features\avatar\presentation\component\avatar_form.dart
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../vm/avatar_form_vm.dart';

class AvatarForm extends StatelessWidget {
  const AvatarForm({super.key, required this.vm, required this.onSave});

  final AvatarFormVm vm;
  final Future<void> Function() onSave;

  @override
  Widget build(BuildContext context) {
    final saving = vm.saving;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // ✅ 「アバター情報を登録してください」のカードは削除
        Text('アバターアイコン画像', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        _IconPickerCard(
          bytes: vm.iconBytes,
          fileName: vm.iconFileName,
          onPick: saving ? null : vm.pickIcon,
          onClear: saving ? null : vm.clearIcon,
        ),
        const SizedBox(height: 16),

        Text('アバター名', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        TextField(
          controller: vm.nameCtrl,
          textInputAction: TextInputAction.next,
          decoration: const InputDecoration(
            labelText: 'アバター名',
            border: OutlineInputBorder(),
            hintText: '例: sotaro',
          ),
          onChanged: (_) => vm.onNameChanged(),
        ),
        const SizedBox(height: 16),

        Text('プロフィール', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        TextField(
          controller: vm.profileCtrl,
          maxLines: 4,
          decoration: const InputDecoration(
            labelText: 'プロフィール',
            border: OutlineInputBorder(),
            hintText: '例: 私は○○のクリエイターです。',
          ),
        ),
        const SizedBox(height: 16),

        Text('外部リンク', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        TextField(
          controller: vm.linkCtrl,
          keyboardType: TextInputType.url,
          decoration: const InputDecoration(
            labelText: '外部リンク（任意）',
            border: OutlineInputBorder(),
            hintText: '例: https://example.com',
          ),
        ),
        const SizedBox(height: 20),

        ElevatedButton(
          onPressed: vm.canSave ? onSave : null,
          child: vm.saving
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(
                  vm.mode == AvatarFormMode.create
                      ? 'このアバターを保存する'
                      : 'この変更を保存する',
                ),
        ),

        if (vm.msg != null) ...[
          const SizedBox(height: 12),
          _InfoBox(
            kind: vm.isSuccessMessage ? _InfoKind.ok : _InfoKind.error,
            text: vm.msg!,
          ),
        ],

        if (kDebugMode && vm.iconBytes != null) ...[
          const SizedBox(height: 12),
          Text(
            'debug: iconBytesLen=${vm.iconBytes!.lengthInBytes}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
      ],
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
  final Future<void> Function()? onPick;
  final VoidCallback? onClear;

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
