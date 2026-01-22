// frontend\mall\lib\features\avatar\presentation\component\avatar_form.dart
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
    final loadingInitial = vm.loadingInitial;

    Future<void> onDeleteExistingIconObject() async {
      // 推奨B:
      // - DB の avatarIcon 文字列は更新しない（送らない）
      // - 既存画像の「削除」は GCS object の delete を叩く
      if (saving || loadingInitial) return;

      try {
        await vm.deleteExistingIconObject();
      } catch (e) {
        // UI 側では落とさない（vm.msg で出すのが本筋）
        if (kDebugMode) {
          // ignore: avoid_print
          print('deleteExistingIconObject failed: $e');
        }
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (loadingInitial) ...[
          const Center(child: CircularProgressIndicator()),
          const SizedBox(height: 16),
        ],

        Text('アバターアイコン画像', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        _IconPickerCard(
          bytes: vm.iconBytes,
          existingUrl: vm.existingAvatarIconUrl, // 既存URL表示用（固定URL）
          fileName: vm.iconFileName,
          onPick: (saving || loadingInitial) ? null : vm.pickIcon,
          onClearPicked: (saving || loadingInitial) ? null : vm.clearIcon,

          // ✅ 既存画像の削除 = GCS object delete（DB の avatarIcon は変更しない）
          onDeleteExistingObject: (saving || loadingInitial)
              ? null
              : onDeleteExistingIconObject,
        ),
        const SizedBox(height: 16),

        Text('アバター名', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        TextField(
          controller: vm.nameCtrl,
          textInputAction: TextInputAction.next,
          enabled: !(saving || loadingInitial),
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
          enabled: !(saving || loadingInitial),
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
          enabled: !(saving || loadingInitial),
          decoration: const InputDecoration(
            labelText: '外部リンク（任意）',
            border: OutlineInputBorder(),
            hintText: '例: https://example.com',
          ),
        ),
        const SizedBox(height: 20),

        ElevatedButton(
          onPressed: (saving || loadingInitial)
              ? null
              : (vm.canSave ? onSave : null),
          child: saving
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

        if (kDebugMode) ...[
          const SizedBox(height: 12),
          Text(
            'debug: iconBytesLen=${vm.iconBytes?.lengthInBytes ?? 0} '
            'existingUrl=${(vm.existingAvatarIconUrl ?? '').trim()}',
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
    required this.existingUrl,
    required this.fileName,
    required this.onPick,
    required this.onClearPicked,
    required this.onDeleteExistingObject,
  });

  final Uint8List? bytes;

  /// Backend の正規キー: avatarIcon（固定URL。文字列は原則変えない）
  final String? existingUrl;

  final String? fileName;

  /// 新規画像を選ぶ
  final Future<void> Function()? onPick;

  /// 新規選択（bytes）を取り消す（=元に戻る）
  final VoidCallback? onClearPicked;

  /// ✅ 既存画像の削除 = GCS object delete（DB の avatarIcon 文字列は変更しない）
  final Future<void> Function()? onDeleteExistingObject;

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    final hasPicked = bytes != null && bytes!.isNotEmpty;
    final hasExisting = _isHttpUrl(existingUrl);

    final statusText = hasPicked ? '選択済み' : (hasExisting ? '既存画像' : '未選択');

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outlineVariant.withValues(alpha: 0.6)),
      ),
      child: Row(
        children: [
          _AvatarPreview(bytes: bytes, existingUrl: existingUrl),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(statusText, style: Theme.of(context).textTheme.bodyLarge),

                if (hasPicked && (fileName ?? '').trim().isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    fileName!.trim(),
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

                    // 「削除」= 新規選択の取り消し（bytes を消す）
                    if (hasPicked)
                      TextButton(
                        onPressed: onClearPicked,
                        child: const Text('削除'),
                      ),

                    // 「既存画像を削除」= GCS object delete
                    // 新規選択中はまず取り消してから既存削除を促すため、hasPicked のときは出さない
                    if (!hasPicked && hasExisting)
                      TextButton(
                        onPressed: onDeleteExistingObject == null
                            ? null
                            : () async {
                                await onDeleteExistingObject!.call();
                              },
                        child: const Text('既存画像を削除'),
                      ),
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
  const _AvatarPreview({required this.bytes, required this.existingUrl});

  final Uint8List? bytes;

  /// Backend の正規キー: avatarIcon（http(s) URL のみ表示）
  final String? existingUrl;

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    final b = bytes;
    final hasPicked = b != null && b.isNotEmpty;

    if (hasPicked) {
      return ClipOval(
        child: Image.memory(
          b,
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

    final url = (existingUrl ?? '').trim();
    if (_isHttpUrl(url)) {
      return ClipOval(
        child: SizedBox(
          width: 56,
          height: 56,
          child: Image.network(
            url,
            fit: BoxFit.cover,
            errorBuilder: (_, __, ___) {
              return CircleAvatar(
                radius: 28,
                backgroundColor: scheme.surfaceContainerHighest,
                child: Icon(Icons.broken_image, color: scheme.onSurfaceVariant),
              );
            },
          ),
        ),
      );
    }

    return CircleAvatar(
      radius: 28,
      backgroundColor: scheme.surfaceContainerHighest,
      child: Icon(Icons.person, color: scheme.onSurfaceVariant),
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
