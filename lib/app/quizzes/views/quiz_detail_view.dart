import 'package:ema_educacion_medica_avanzada/app/chat/widgets/animations/chat_typing_indicator.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/screens.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/content_header.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuizDetailView extends StatelessWidget {
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();
  final controller = Get.find<QuizController>();

  QuizDetailView({super.key});

  @override
  Widget build(BuildContext context) {
    return AppLayout(
      key: _scaffoldKey,
      backRoute: Routes.home.name,
      body: controller.obx(
        (quiz) {
          return Column(
            children: [
              Expanded(
                child: QuestionsList(
                  header: ContentHeader.headerItems(
                    subtitle: quiz!.title,
                    breadcrumb: quiz.shortTitle,
                  ),
                ),
              ),
              Obx(() {
                // Mostrar la animación si está tipeando o si no se han recibido preguntas
                if (controller.isTyping.value ||
                    controller.rxQuestions.isEmpty) {
                  return const ChatTypingIndicator();
                }
                return Container();
              }),
              QuestionInputs(),
            ],
          );
        },
        onLoading: Obx(() {
          return StateMessageWidget(
            message: controller.textLoading.value,
            type: StateMessageType.download,
            showLoading: true,
          );
        }),
        onEmpty: const StateMessageWidget(
          message: 'No se encontró un cuestionario disponible',
          type: StateMessageType.noSearchResults,
        ),
        onError: (error) {
          return StateMessageWidget(
            message: 'Error al cargar el cuestionario',
            showHomeButton: true,
            type: StateMessageType.error,
          );
        },
      ),
    );
  }
}
