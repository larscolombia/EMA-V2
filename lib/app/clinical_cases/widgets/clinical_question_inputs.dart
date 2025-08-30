import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/question_input_text.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class ClinicalQuestionInputs extends StatelessWidget {
  final ClinicalCaseController controller;

  const ClinicalQuestionInputs({super.key, required this.controller});

  @override
  Widget build(BuildContext context) {
    void onPressedUniqueAnswer(AnswerModel answer) {
      // Get the current active question from currentQuestion in controller
      final activeQuestion = controller.currentQuestion.value;

      if (activeQuestion != null) {
        controller.sendAnswer(question: activeQuestion, userAnswer: answer);
      }
    }

    Widget buildQuestionInput(QuestionResponseModel? question) {
      if (controller.state == null || question == null) {
        return SizedBox();
      }

      if (controller.isComplete.value) {
        // Caso completo: si hay resumen oculto sin mostrar
        if (controller.hasHiddenInteractiveSummary && !controller.interactiveEvaluationGenerated.value) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
            child: Column(
              children: [
                OutlineAiButton(
                  text: 'Ver Evaluación Final',
                  onPressed: () async {
                    await controller.showInteractiveSummaryIfAvailable();
                    final caseModel = controller.currentCase.value;
                    if (caseModel != null) {
                      Get.toNamed(Routes.clinicalCaseEvaluation.path(caseModel.uid));
                    }
                  },
                ),
                const SizedBox(height: 8),
                OutlineAiButton(
                  text: 'Regresar al Inicio',
                  onPressed: () => Get.toNamed(Routes.home.name),
                ),
              ],
            ),
          );
        }
        // Si ya se mostró la evaluación
        if (controller.interactiveEvaluationGenerated.value) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
            child: OutlineAiButton(
              text: 'Regresar al Inicio',
              onPressed: () => Get.toNamed(Routes.home.name),
            ),
          );
        }
      }

      if (question.type == QuestionType.singleChoice) {
        return QuestionInputSingleChoice(
          onAnswer: onPressedUniqueAnswer,
          question: question,
          isDisabled:
              controller
                  .isTyping
                  .value, // We'll continue using isTyping to disable
        );
      }

      if (question.type == QuestionType.open) {
        return QuestionInputText(
          onAnswer: onPressedUniqueAnswer,
          question: question,
          title: 'Mi respuesta es...',
          isDisabled: controller.isTyping.value,
        );
      }

      return SizedBox();
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 8),
      decoration: BoxDecoration(
        color: AppStyles.whiteColor,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 4,
            offset: Offset(0, -2),
          ),
        ],
      ),
      child: Obx(() {
        // First check if the case is complete, if so show exit button
        if (controller.isComplete.value) {
          return buildQuestionInput(
            controller.questions.isEmpty
                ? QuestionResponseModel.empty()
                : controller.questions.last,
          );
        }

        // Get the current active question or the last question if there's no active one
        // This ensures the widget is still visible even when waiting for a new question
        final currentQuestion =
            controller.currentQuestion.value ??
            (controller.questions.isNotEmpty
                ? controller.questions.last
                : null);

        // Show input only if there's a question to show, otherwise empty space
        return currentQuestion != null
            ? buildQuestionInput(currentQuestion)
            : SizedBox();
      }),
    );
  }
}
