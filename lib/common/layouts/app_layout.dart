import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_drawer_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_keyboard_visibility/flutter_keyboard_visibility.dart';
import 'package:get/get.dart';

class AppLayout extends StatelessWidget {
  final _actionsDrawerController = Get.find<ActionsDrawerListController>();
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();

  final chatController = Get.find<ChatController>();
  final navigationService = Get.find<NavigationService>();
  final userService = Get.find<UserService>();

  final Widget body;
  final Widget? bottomNavigationBar;
  final String backRoute;

  AppLayout({
    super.key,
    required this.body,
    this.bottomNavigationBar,
    this.backRoute = '',
  });

  bool _navigationLocked() {
    bool locked =
        chatController.isSending.value || chatController.isTyping.value;
    if (Get.isRegistered<ClinicalCaseController>()) {
      locked = locked || Get.find<ClinicalCaseController>().isTyping.value;
    }
    if (Get.isRegistered<QuizController>()) {
      locked = locked || Get.find<QuizController>().isTyping.value;
    }
    return locked;
  }

  AppBar appBar() {
    return AppBar(
      centerTitle: true,
      leading: Obx(() {
        final disabled = _navigationLocked();
        return IconButton(
          color: Colors.black,
          icon:
              backRoute.isEmpty
                  ? AppIcons.menuSquare(height: 34, width: 34)
                  : AppIcons.arrowLeftSquare(height: 34, width: 34),
          onPressed:
              disabled
                  ? null
                  : () {
                    if (backRoute.isNotEmpty) {
                      Get.toNamed(backRoute);
                    } else {
                      _actionsDrawerController.loadActions(0);
                      _scaffoldKey.currentState!.openDrawer();
                    }
                  },
        );
      }),
      title: TextButton(
        onPressed: () {
          Get.offAllNamed(Routes.home.name);
        },
        child: const Image(
          image: AssetImage('assets/images/logotype_color.png'),
          height: 42,
          width: 120,
        ),
      ),
      actions: [
        Obx(() {
          final disabled = _navigationLocked();
          return IconButton(
            icon: AppIcons.plume(
              color: AppStyles.tertiaryColor,
              height: 34,
              width: 34,
            ),
            onPressed:
                disabled
                    ? null
                    : () {
                      chatController.cleanChat();

                      Get.toNamed(Routes.home.name, preventDuplicates: true);

                      WidgetsBinding.instance.addPostFrameCallback((_) {
                        chatController.focusOnChatInputText();
                      });
                    },
          );
        }),
        SizedBox(width: 4),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return KeyboardDismissOnTap(
      dismissOnCapturedTaps: false,
      child: WillPopScope(
        onWillPop: () async {
          if (_navigationLocked()) {
            Notify.snackbar(
              'En progreso',
              'Espera a que finalice la respuesta',
              NotifyType.warning,
            );
            return false;
          }
          return true;
        },
        child: Scaffold(
          key: _scaffoldKey,
          appBar: appBar(),
          body: SafeArea(child: body),
          drawer: AppDrawer(
            userProfile: userService.currentUser.value,
            chatController: chatController,
          ),
          drawerScrimColor: AppStyles.primary900,
          bottomNavigationBar: bottomNavigationBar,
        ),
      ),
    );
  }
}
