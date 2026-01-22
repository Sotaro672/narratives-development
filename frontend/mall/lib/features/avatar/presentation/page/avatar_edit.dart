// frontend\mall\lib\features\avatar\presentation\page\avatar_edit.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart';
import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/header.dart';

import '../../infrastructure/api.dart';
import '../../infrastructure/avatar_api_client.dart';
import '../component/avatar_form.dart';
import '../vm/avatar_form_vm.dart';

class AvatarEditPage extends StatefulWidget {
  const AvatarEditPage({super.key});

  @override
  State<AvatarEditPage> createState() => _AvatarEditPageState();
}

class _AvatarEditPageState extends State<AvatarEditPage> {
  late final AvatarApiClient _apiClient;
  late final MallAuthedApi _meApi;
  late final AvatarFormVm _vm;

  bool _loaded = false;

  /// NavStore.consumeReturnTo() は副作用があるため、ページ内で 1 回だけ消費して保持する
  late final String _backTo;

  @override
  void initState() {
    super.initState();

    _backTo = _consumeBackToOnce();

    _apiClient = AvatarApiClient();
    _meApi = MallAuthedApi();

    // ✅ submitPatch を DI して「更新が実行される」ようにする
    // ✅ avatarIcon は me PATCH では更新しない運用（推奨B）なので送らない
    _vm = AvatarFormVm(
      mode: AvatarFormMode.edit,
      apiClient: _apiClient,
      submitPatch: _submitPatch,
    );

    _vm.addListener(_onVmChanged);
    _load();
  }

  String _consumeBackToOnce() {
    try {
      final v = NavStore.I.consumeReturnTo();
      final s = v.toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}
    return AppRoutePath.avatar;
  }

  Future<void> _load() async {
    await _vm.loadInitialIfNeeded();
    if (!mounted) return;
    setState(() => _loaded = true);
  }

  void _onVmChanged() {
    if (!mounted) return;
    setState(() {});
  }

  /// ✅ AvatarFormVm から渡された patch を、実際の HTTP PATCH に変換して実行する
  ///
  /// 方針（me handler）:
  /// - avatarId は解決不要（サーバ側で uid -> avatarId を解決）
  /// - PATCH /mall/me/avatar を呼ぶ
  /// - avatarIcon は送らない（推奨B：残しても良いがフロントが送らない運用で担保）
  Future<void> _submitPatch(AvatarPatchRequest patch) async {
    // avatarIcon は “固定URL運用” のため、me PATCH には載せない
    final json = <String, dynamic>{};
    if (patch.avatarName != null) json['avatarName'] = patch.avatarName;
    if (patch.profile != null) json['profile'] = patch.profile;
    if (patch.externalLink != null) json['externalLink'] = patch.externalLink;

    // avatarIcon は明示的に無視（送らない）
    // if (patch.avatarIcon != null) { ... } しない

    // no-op patch は送らない（UI操作上は通常ここに来ない想定だが安全側）
    if (json.isEmpty) return;

    final uri = _meApi.uri('/mall/me/avatar');
    await _meApi.sendAuthed('PATCH', uri, jsonBody: json);
  }

  Future<void> _onSave() async {
    // ✅ submitPatch は VM に DI 済みなので、ここではそのまま save() でOK
    final ok = await _vm.save();
    if (!mounted) return;

    if (ok) {
      context.go(_backTo);
    }
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    _meApi.dispose();
    _apiClient.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'Edit Avatar',
              showBack: true,
              backTo: _backTo,
              actions: const [],
              onTapTitle: () => context.go(AppRoutePath.home),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: !_loaded
                        ? const Center(child: CircularProgressIndicator())
                        : AvatarForm(vm: _vm, onSave: _onSave),
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
