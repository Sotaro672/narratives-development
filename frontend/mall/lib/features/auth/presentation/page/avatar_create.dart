//frontend\mall\lib\features\auth\presentation\page\avatar_create.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart';
import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/header.dart';

import '../../../avatar/presentation/component/avatar_form.dart';
import '../../../avatar/presentation/vm/avatar_form_vm.dart';

class AvatarCreatePage extends StatefulWidget {
  const AvatarCreatePage({super.key});

  @override
  State<AvatarCreatePage> createState() => _AvatarCreatePageState();
}

class _AvatarCreatePageState extends State<AvatarCreatePage> {
  late final AvatarFormVm _vm;

  @override
  void initState() {
    super.initState();
    _vm = AvatarFormVm(mode: AvatarFormMode.create);
    _vm.addListener(_onVmChanged);
  }

  void _onVmChanged() {
    if (!mounted) return;
    setState(() {});
  }

  @override
  void dispose() {
    _vm.removeListener(_onVmChanged);
    _vm.dispose();
    super.dispose();
  }

  String _consumeBackTo() {
    try {
      // consumeReturnTo() が non-nullable の場合に合わせる
      final v = NavStore.I.consumeReturnTo();
      final s = v.toString().trim();
      if (s.isNotEmpty) return s;
    } catch (_) {}
    return AppRoutePath.avatar;
  }

  Future<void> _onSave() async {
    final ok = await _vm.save();
    if (!mounted) return;

    if (ok) {
      // Pattern B: 戻り先は NavStore。無ければ avatar へ。
      final backTo = _consumeBackTo();
      context.go(backTo);
    }
  }

  @override
  Widget build(BuildContext context) {
    final backTo = _consumeBackTo();

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: 'アバター作成',
              showBack: true,
              backTo: backTo,
              actions: const [],
              onTapTitle: () => context.go(AppRoutePath.home),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: AvatarForm(vm: _vm, onSave: _onSave),
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
