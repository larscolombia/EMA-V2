// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/screens.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/content_header.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/full_feedback_animated.dart';

class QuizFeedBackView extends GetView<QuizController> {
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();

  QuizFeedBackView({super.key});

  @override
  Widget build(BuildContext context) {
    final body = controller.obx(
      (quiz) {
        if (quiz == null || quiz.feedback.isEmpty) {
          return const StateMessageWidget(
            message: 'No se encontró la retroalimentación.',
            type: StateMessageType.noSearchResults,
          );
        }

        // Mostrar todo el feedback (puntaje, retro, preguntas y referencias) con animación de escritura.
        return SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ContentHeader(subtitle: quiz.title, breadcrumb: quiz.shortTitle),
              Padding(
                padding: const EdgeInsets.only(top: 12.0),
                child: FullFeedbackAnimated(
                  fitGlobal: quiz.feedback,
                  questions: quiz.questions,
                  animate: true,
                ),
              ),
            ],
          ),
        );
      },

      onLoading: Obx(
        () => StateMessageWidget(
          message: controller.textLoading.value,
          type: StateMessageType.download,
          showLoading: true,
        ),
      ),

      onEmpty: const StateMessageWidget(
        message: 'No se encontró el feedback.',
        type: StateMessageType.noSearchResults,
      ),

      onError: (error) {
        return StateMessageWidget(
          message: 'Error al cargar el feedback',
          showHomeButton: true,
          type: StateMessageType.error,
        );
      },
    );

    return AppLayout(
      key: _scaffoldKey,
      backRoute: Routes.home.name,
      body: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
        child: body,
      ),
    );
  }
}
