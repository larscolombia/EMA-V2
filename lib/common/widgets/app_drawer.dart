import 'package:ema_educacion_medica_avanzada/app/actions/widgets/actions_drawer_input.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/widgets/actions_drawer_widget.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../app/profiles/profiles.dart';

class AppDrawer extends StatelessWidget {
  final UserModel userProfile;
  final ChatController chatController;

  const AppDrawer({super.key, required this.userProfile, required this.chatController});

  bool _navigationLocked() {
    bool locked = chatController.isSending.value || chatController.isTyping.value;
    if (Get.isRegistered<ClinicalCaseController>()) {
      locked = locked || Get.find<ClinicalCaseController>().isTyping.value;
    }
    if (Get.isRegistered<QuizController>()) {
      locked = locked || Get.find<QuizController>().isTyping.value;
    }
    return locked;
  }

  @override
  Widget build(BuildContext context) {
    final appBar = AppBar(
      automaticallyImplyLeading: false,
      actions: [
        Obx(() {
          final disabled = _navigationLocked();
          return IconButton(
            icon: AppIcons.plume(
              height: 34,
              width: 34,
              color: AppStyles.tertiaryColor,
            ),
            onPressed: disabled
                ? null
                : () {
                    chatController.cleanChat();
                    Get.back(closeOverlays: true);
                    WidgetsBinding.instance.addPostFrameCallback((_) {
                      chatController.focusOnChatInputText();
                    });
                  },
          );
        }),
      ],
      title: ActionsDrawerSearchInput(),
    );

    return Scaffold(
      backgroundColor: AppStyles.whiteColor,
      resizeToAvoidBottomInset: false,
      appBar: appBar,
      body: SafeArea(child:
        ActionsDrawerListWidget(),
      ),
      bottomNavigationBar: GestureDetector(
        onTap: () {
          Get.toNamed(Routes.profile.name);
        },
        child: ProfileHeader(profile: userProfile, inDrawer: true),
      ),
    );
  }
}
