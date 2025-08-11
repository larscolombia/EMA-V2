import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


class QuizzDetailCard extends GetView<QuizController> {
  final _categoryController = Get.find<CategoriesController>();

  QuizzDetailCard({super.key});

  @override
  Widget build(BuildContext context) {
    final category = _categoryController.currentCategory.value;

    return Scaffold(
      appBar: AppScaffold.appBar(
          backRoute: Routes.quizzesCategory.path(category.id.toString())),
      extendBody: true,

      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
        child: controller.obx(
          (quiz) {
            if (quiz == null) {
              return const Center(child: Text('No hay quizzes disponibles'));
            }

            return Column(
              mainAxisSize: MainAxisSize.max,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  '${category.name} > Cuestionario',
                  style: AppStyles.breadCrumb,
                ),

                SizedBox(height: 12),
                Text(
                  quiz.title,
                  style: AppStyles.subtitle,
                ),

                SizedBox(height: 20),
                Container(
                  padding: const EdgeInsets.only(top: 12, bottom: 14),
                  decoration: BoxDecoration(
                    border: Border.symmetric(
                      horizontal: BorderSide(
                        color: AppStyles.tertiaryColor,
                        width: 0.5,
                      ),
                    ),
                    borderRadius: BorderRadius.circular(0),
                  ),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.start,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text.rich(
                        TextSpan(
                          text: 'Preguntas: ',
                          style: AppStyles.info1,
                          children: [
                            TextSpan(
                              text: quiz.numQuestions.toString(),
                              style: AppStyles.info2,
                            ),
                          ],
                        ),
                      ),
                      Text.rich(
                        TextSpan(
                          text: 'Tiempo estimado: ',
                          style: AppStyles.info1,
                          children: [
                            TextSpan(
                              text: '${quiz.numQuestions} min',
                              style: AppStyles.info2,
                            ),
                          ],
                        ),
                      ),
                      Text.rich(
                        TextSpan(
                          text: 'Nivel de dificultad: ',
                          style: AppStyles.info1,
                          children: [
                            TextSpan(
                              text: quiz.level.name,
                              style: AppStyles.info2,
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),

                SizedBox(height: 20),
                Align(
                  alignment: Alignment.center,
                  child: FilledButton(
                    onPressed: () {},
                    style: ButtonStyle(
                      backgroundColor: WidgetStateProperty.all(AppStyles.tertiaryColor),
                      padding: WidgetStateProperty.all(const EdgeInsets.symmetric(horizontal: 32, vertical: 12)),
                      shape: WidgetStateProperty.all(
                        RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(14),
                        ),
                      ),
                    ),
                    child: Text(
                      'Continuar cuestionario',
                      style: AppStyles.buttonText,
                    ),
                  ),
                ),

                SizedBox(height: 64),
              ],
            );
          },
          onLoading: const Center(child: CircularProgressIndicator()),
          onError: (error) => Center(child: Text(error ?? 'Error al cargar quizzes')),
          onEmpty: const Center(child: Text('No hay quizzes disponibles')),
        ),
      ),

      bottomNavigationBar: Container(
        decoration: BoxDecoration(
          color: AppStyles.whiteColor,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              'Progreso del cuestionario',
              style: AppStyles.info1,
            ),
            Padding(
              padding: const EdgeInsets.only(left: 28, right: 28, top: 8, bottom: 0),
              child: LinearProgressIndicator(
                value: 0.25,
                valueColor: AlwaysStoppedAnimation<Color>(AppStyles.tertiaryColor),
                backgroundColor: AppStyles.grey220,
                borderRadius: BorderRadius.circular(12),
                semanticsLabel: 'Progreso del cuestionario',
                semanticsValue: '10',
                minHeight: 21,
              ),
            ),
            AppScaffold.footerCredits(),
          ],
        ),
      ),
    );
  }
}
