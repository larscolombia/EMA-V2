// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';

import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuestionsList extends StatefulWidget {
  final List<Widget> header;

  const QuestionsList({
    super.key,
    required this.header,
  });

  @override
  State<QuestionsList> createState() => _QuestionsListState();
}

class _QuestionsListState extends State<QuestionsList> {
  final controller = Get.find<QuizController>();
  final scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    controller.setScrollController(scrollController);
  }

  @override
  void dispose() {
    controller.setScrollController(null);
    super.dispose();
  }

  Widget _buildQuestion(QuestionResponseModel question) {
    if (question.type == QuestionType.open) {
      return QuestionMessageOpen(question: question);
    }

    if (question.type == QuestionType.singleChoice) {
      return QuestionMessageSingle(question: question);
    }

    if (question.type == QuestionType.trueFalse) {
      return QuestionMessageTrueFalse(question: question);
    }

    return SizedBox();
  }

  @override
  Widget build(BuildContext context) {
    controller.setScrollController(scrollController);

    return SingleChildScrollView(
      controller: scrollController,
      padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
      child: Obx(() {

        final questions = controller.inChatList.map((q) {
          return _buildQuestion(q);
        }).toList();

        final current = !controller.isComplete.value
          ? [_buildQuestion(controller.currentQuestion.value)]
          : [];

        return Column(
          mainAxisSize: MainAxisSize.max,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ...widget.header,
            ...questions,
            ...current,
          ],
        );
      }),
    );
  }
}
