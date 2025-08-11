import 'package:ema_educacion_medica_avanzada/app/quizzes/controllers/quiz_controller.dart';
import 'package:get/get.dart';


class QuizBinding extends Bindings {
  @override
  void dependencies() {
    Get.lazyPut<QuizController>(() => QuizController());
  }
}
