// import 'package:ema_educacion_medica_avanzada/common/bindings.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/screens/actions_list_screen.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/views/clinical_case_interactive_view.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/views/clinical_case_evaluation_view.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/views/quiz_feed_back_view.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/subscriptions.dart';
import 'package:ema_educacion_medica_avanzada/common/screens.dart';
import 'package:get/get.dart';

part 'routes.dart';


class AppPages {
  AppPages._();

  static final routes = [
    GetPage(
      name: Routes.actionsList.name,
      page: () => ActionsListScreen(),
    ),
    GetPage(
      name: Routes.start.name,
      page: () => StartScreen(),
    ),
    GetPage(
      name: Routes.home.name,
      page: () => ChatHomeView(),
    ),
    GetPage(
      name: Routes.profile.name,
      page: () => ProfileView(),
    ),
    GetPage(
      name: Routes.subscriptions.name,
      page: () => SubscriptionsView(),
    ),
    GetPage(
      name: Routes.login.name,
      page: () => LoginScreen(),
    ),
    GetPage(
      name: Routes.register.name,
      page: () => RegisterFormView(),
    ),
    GetPage(
      name: Routes.forgotPassword.name,
      page: () => ForgotPasswordScreen(),
    ),
    GetPage(
      name: Routes.clinicalCaseAnalytical.name,
      page: () => ClinicalCaseAnalyticalView(),
    ),
    GetPage(
      name: Routes.clinicalCaseInteractive.name,
      page: () => ClinicalCaseInteractiveView(),
    ),
    GetPage(
      name: Routes.clinicalCaseEvaluation.name,
      page: () => ClinicalCaseEvaluationView(),
    ),
    GetPage(
      name: Routes.quizDetail.name,
      page: () => QuizDetailView(),
    ),
    GetPage(
      name: Routes.quizFeedBack.name,
      page: () => QuizFeedBackView(),
    ),
  ];
}
