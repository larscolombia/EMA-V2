import 'package:ema_educacion_medica_avanzada/app/quizzes/models/answer_model.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/models/quiz_status.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuestionInputs extends StatelessWidget {
  final controller = Get.find<QuizController>();

  QuestionInputs({
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    void onPressedUniqueAnswer (AnswerModel answer) {
      controller.saveAnswer(
        question: controller.currentQuestion.value,
        answer: answer,
      );
    }

    Widget buildQuestionInput(QuestionResponseModel question) {
      if (controller.state == null) { 
        return SizedBox();
      }

      if (controller.totalQuestions.value == 0) {
        return Padding(
          padding: const EdgeInsets.all(8),
          child: Text('Cargando...'),
        );
      }

      if (controller.state != null && controller.state!.status == QuizStatus.completed) {
        return Padding(
          padding: const EdgeInsets.only(bottom: 4),
          child: OutlineAiButton(
            text: 'Calificar Respuestas',
            onPressed: () {
              controller.evaluateCurrentQuiz();
            },
          ),
        );
      }

      if (controller.state != null && controller.state!.status == QuizStatus.scored) { 
        return Padding(
          padding: const EdgeInsets.only(bottom: 4),
          child: OutlineAiButton(
            text: 'Evaluar Desempeño',
            onPressed: () {
              controller.evaluateCurrentQuiz();
            },
          ),
        );
      }

      if (question.type == QuestionType.open) {
        return QuestionInputText(
          question: question,
          onAnswer: onPressedUniqueAnswer,
          isDisabled: controller.isTyping.value,
        );
      }

      if (question.type == QuestionType.singleChoice) {
        return QuestionInputSingleChoice(
          onAnswer: onPressedUniqueAnswer,
          question: question,
          isDisabled: controller.isTyping.value,
        );
      }

      if (question.type == QuestionType.trueFalse) {
          return QuestionInputTrueFalse(
          onAnswer: onPressedUniqueAnswer,
          question: question,
          isDisabled: controller.isTyping.value,
        );
      }

      return Text(question.question);
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 8),
      decoration: BoxDecoration(
        color: AppStyles.whiteColor,
      ),
      child: Obx(() {
        return Column(mainAxisSize: MainAxisSize.min, children: [
          LinearProgressIndicator(
            value: controller.progress.value,
            valueColor: AlwaysStoppedAnimation<Color>(AppStyles.tertiaryColor),
            backgroundColor: AppStyles.grey220,
            borderRadius: BorderRadius.circular(8),
            semanticsLabel: 'Progreso del cuestionario',
            semanticsValue: controller.progress.value.toString(),
            minHeight: 6,
          ),
      
          SizedBox(height: 8),
      
          buildQuestionInput(controller.currentQuestion.value),
        ]);
      }),
    );
  }
}
