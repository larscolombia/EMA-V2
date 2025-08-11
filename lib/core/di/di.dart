// Dependency Injection
import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_drawer_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/controllers/actions_list_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/data/api_category_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/api_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/data/local_clinical_case_data.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/interfaces/clinical_case_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/services/clinical_cases_services.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';

import '../../app/subscriptions/subscriptions.dart';
import '../auth/session_service.dart';

Future<void> registerDependencies() async {
  // Core Services
  Get.put<UiObserverService>(UiObserverService());
  Get.put<NavigationService>(NavigationService());
  Get.put<DatabaseService>(DatabaseService());
  Get.put<LaravelAuthService>(LaravelAuthService());
  Get.put<UserLocalDataService>(UserLocalDataService());
  Get.put(UserService(), permanent: true);
  Get.put<ApiService>(ApiService());
  Get.put<AttachmentService>(AttachmentService());
  Get.put<SessionService>(SessionService());

  // App Services
  Get.lazyPut<ActionsService>(() => ActionsService(), fenix: true);
  Get.lazyPut<ApiCategoryData>(() => ApiCategoryData(), fenix: true);
  Get.lazyPut<CategoriesController>(() => CategoriesController(), fenix: true);
  Get.lazyPut<ChatsService>(() => ChatsService(), fenix: true);
  Get.lazyPut<ClinicalCaseController>(() => ClinicalCaseController(),
      fenix: true);
  Get.lazyPut<ClinicalCasesServices>(() => ClinicalCasesServices(),
      fenix: true);
  Get.lazyPut<IApiChatData>(() => ApiChatData(), fenix: true);
  Get.lazyPut<QuizzesService>(() => QuizzesService(), fenix: true);
  Get.lazyPut<ProfileService>(() => ApiProfileService(), fenix: true);
  Get.lazyPut<CountryService>(() => CountryService(), fenix: true);
  Get.lazyPut<SubscriptionService>(() => ApiSubscriptionService(), fenix: true);
  Get.lazyPut<UserTestProgressService>(() => ApiUserTestProgressService(),
      fenix: true);
  Get.lazyPut<SubscriptionService>(() => ApiSubscriptionService(), fenix: true);

  // App Controllers
  Get.lazyPut<ActionsDrawerListController>(() => ActionsDrawerListController(),
      fenix: true);
  Get.lazyPut<ActionsListController>(() => ActionsListController(),
      fenix: true);
  Get.lazyPut<ChatController>(() => ChatController(), fenix: true);
  Get.lazyPut<QuizController>(() => QuizController(), fenix: true);
  // Get.lazyPut<QuizzesController>(() => QuizzesController(), fenix: true);
  Get.lazyPut<ProfileController>(() => ProfileController(), fenix: true);
  Get.lazyPut<SubscriptionController>(() => SubscriptionController(),
      fenix: true);
  Get.lazyPut<UserTestProgressController>(() => UserTestProgressController(),
      fenix: true);

  // Repositories
  // Get.lazyPut<FakeRemoteCategoryData>(() => FakeRemoteCategoryData(),
  //     fenix: true);
  Get.lazyPut<IQuizRemoteData>(() => ApiQuizzData(), fenix: true);
  Get.lazyPut<LocalActionsData>(() => LocalActionsData(), fenix: true);
  Get.lazyPut<LocalChatData>(() => LocalChatData(), fenix: true);
  Get.lazyPut<LocalChatMessageData>(() => LocalChatMessageData(), fenix: true);
  // Get.lazyPut<IClinicalCaseMessageLocalData>(() => LocalClinicalCaseMessageData(), fenix: true);
  Get.lazyPut<LocalQuestionsData>(() => LocalQuestionsData(), fenix: true);
  Get.lazyPut<LocalQuizzData>(() => LocalQuizzData(), fenix: true);
  Get.lazyPut<ApiClinicalCaseData>(() => ApiClinicalCaseData(), fenix: true);
  // Get.lazyPut<IQuizRemoteData>(() => FakerQuizzData(), fenix: true);
  // Get.lazyPut<FakeRemoteCategoryData>(() => FakeRemoteCategoryData(),
  //     fenix: true);
  Get.lazyPut<IClinicalCaseLocalData>(() => LocalClinicalCaseData(),
      fenix: true);

  // Controllers for current instances
  Get.lazyPut<QuizController>(() => QuizController(), fenix: true);
  // Get.lazyPut<ClinicalCaseController>(() => ClinicalCaseController(), fenix: true);
}
