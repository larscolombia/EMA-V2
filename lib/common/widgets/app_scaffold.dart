import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:url_launcher/url_launcher.dart';

class AppScaffold {

  static AppBar appBar({String backRoute = '', bool forceBackRoute = false}) {
    final chatController = Get.find<ChatController>();
    final userService = Get.find<UserService>();

    bool navigationLocked() {
      bool locked = chatController.isSending.value || chatController.isTyping.value;
      if (Get.isRegistered<ClinicalCaseController>()) {
        locked = locked || Get.find<ClinicalCaseController>().isTyping.value;
      }
      if (Get.isRegistered<QuizController>()) {
        locked = locked || Get.find<QuizController>().isTyping.value;
      }
      return locked;
    }

    return AppBar(
      centerTitle: true,
      
      leading: Obx(() {
        final disabled = navigationLocked();
        return IconButton(
          color: Colors.black,
          icon: backRoute.isNotEmpty || forceBackRoute
              ? AppIcons.arrowLeftSquare(
                  height: 34,
                  width: 34,
                )
              : AppIcons.menuSquare(
                  height: 34,
                  width: 34,
                ),
          onPressed: disabled
              ? null
              : () {
                  if (forceBackRoute) {
                    Get.back();
                  } else if (backRoute.isNotEmpty) {
                    Get.toNamed(backRoute);
                  } else {
                    // Scaffold.of(context).openDrawer();
                    userService.showProfileView();
                  }
                },
        );
      }),
      
      title: TextButton(
        onPressed: () {
          Get.toNamed(Routes.home.name, preventDuplicates: true);
        },
        child: const Image(
          image: AssetImage('assets/images/logotype_color.png'),
          height: 42,
          width: 120,
        ),
      ),
      
      actions: [
        Obx(() {
          final disabled = navigationLocked();
          return IconButton(
            icon: AppIcons.plume(
              color: AppStyles.tertiaryColor,
              height: 34,
              width: 34,
            ),
            onPressed: disabled
                ? null
                : () {
                    chatController.cleanChat();
                    Get.toNamed(Routes.home.name, preventDuplicates: true);
                  },
          );
        }),
      ],
    );
  }

  static Widget bottomSheet() {
    return Container(
      decoration: BoxDecoration(
        color: AppStyles.whiteColor,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AppScaffold.footerCredits(),
        ],
      ),
    );
  }

  static Widget bottomFieldBox(ChatController chatController, NavigationService navigationService) {
    final uiObserverService = Get.find<UiObserverService>();

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Padding(
          padding: EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          child: MessageFieldBox(chatController: chatController, navigatioService: navigationService),
        ),
        Obx(() {
          final showCredits = !uiObserverService.isKeyboardVisible.value;
          return showCredits ? AppScaffold.footerCredits() : SizedBox.shrink();
        }),
      ],
    );
  }

  static Widget footerCredits() {
    return TextButton(
      onPressed: () {
        launchUrl(Uri.parse('https://chat.lubot.lars.net.co'));
      },
      style: ButtonStyle(
        padding: WidgetStateProperty.all(EdgeInsets.zero),
        shape: WidgetStateProperty.all(RoundedRectangleBorder(
          borderRadius: BorderRadius.zero,
        )),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('Developed by: '),
          Text('LARS', style: TextStyle(color: AppStyles.tertiaryColor)),
        ],
      ),
    );
  }
}
