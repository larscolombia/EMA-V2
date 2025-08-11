// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/screens.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/content_header.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gpt_markdown/gpt_markdown.dart';


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

        if (quiz.animated) {
          return SingleChildScrollView(
            child: Column(
              children: [
                ContentHeader(subtitle: quiz.title, breadcrumb: quiz.shortTitle),
                GptMarkdown(quiz.feedback),
              ],
            )
          );
        } else {
          // controller.markAsAnimated(quiz.uid);
          final textAnimationProgress = ValueNotifier<int>(0);

          return SingleChildScrollView(
            reverse: true,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                ContentHeader(subtitle: quiz.title, breadcrumb: quiz.shortTitle),
                TweenAnimationBuilder<int>(
                  tween: IntTween(begin: 1, end: quiz.feedback.length),
                  // Ajusta la duración según la longitud del texto
                  duration: Duration(seconds: quiz.feedback.length ~/ 80),
                  builder: (context, value, child) {
                    // Actualiza el ValueNotifier para que AnimatedBuilder se reconstruya
                    return AnimatedBuilder(
                      animation: textAnimationProgress,
                      builder: (context, _) {
                        textAnimationProgress.value = value;
                        final displayedText = quiz.feedback.substring(0, textAnimationProgress.value);
                        return GptMarkdown(displayedText);
                      },
                    );
                  },
                ),
              ],
            ),
          );
        }
      },
    
      onLoading: Obx(() =>StateMessageWidget(
        message: controller.textLoading.value,
        type: StateMessageType.download,
        showLoading: true,
      )),
    
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
