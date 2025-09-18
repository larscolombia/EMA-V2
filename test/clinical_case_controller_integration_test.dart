import 'dart:async';

import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/controllers/clinical_case_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_model.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/life_stage.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/sex_and_status.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_type.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:get/get.dart';
import 'package:flutter_test/flutter_test.dart';

/// Integration-style test that fakes dependencies and verifies that
/// the ClinicalCaseController reacts to SSE-style stage tokens
/// forwarded by the ClinicalCasesServices via the `onStream` callback.
void main() {
  setUp(() {
    Get.reset();
  });

  test('SSE stage tokens update currentStage and AI message is added', () async {
    // Arrange: register fake services before creating controller
    final fakeUser = _FakeUserService();
    final fakeUi = _FakeUiObserverService();
    final fakeProfile = _FakeProfileController();
    final fakeClinicalService = _FakeClinicalCasesServices();

    // Instantiate a testable controller that allows setting StateMixin value
    final controller = Get.put(
      TestableClinicalCaseController(
        clinicalCaseServive: fakeClinicalService,
        uiObserverService: fakeUi,
        userService: fakeUser,
        profileController: fakeProfile,
      ),
    );

    // Provide a minimal clinical case via the convenient factory
    final caseModel = ClinicalCaseModel.generate(
      userId: fakeUser.currentUser.value.id,
      type: ClinicalCaseType.analytical,
      lifeStage: LifeStage.adulto,
      sexAndStatus: SexAndStatus.man,
    );

    controller.currentCase.value = caseModel;
    controller.setStateValue(caseModel);

    // Act: start sendMessage but don't await immediately so we can observe stage change
    final sendFuture = controller.sendMessage('Hola IA');

    // Wait for the fake service to emit first stage
    await fakeClinicalService.firstStageEmitted.future.timeout(
      Duration(seconds: 2),
    );

    // Assert that controller observed a stage
    expect(
      controller.currentStage.value.isNotEmpty,
      isTrue,
      reason: 'Expected a non-empty stage value after onStream',
    );

    // Await completion
    await sendFuture;

    // Final assertions: last message should be AI and contain expected text
    expect(controller.messages.isNotEmpty, isTrue);
    final last = controller.messages.last;
    expect(last.aiMessage, isTrue);
    expect(last.text, equals('Respuesta AI final'));
    // After completion the controller clears the stage
    expect(controller.currentStage.value, isEmpty);
  });
}

class _FakeUserService implements UserService {
  // Provide an Rx<UserModel> compatible with the real service
  final Rx<UserModel> currentUser = UserModel.unknow().obs;

  _FakeUserService() {
    currentUser.value.id = 123;
  }

  @override
  // ignore: avoid_returning_null
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeUiObserverService implements UiObserverService {
  @override
  // Expose the Rx<bool> expected by the controller
  Rx<bool> isKeyboardVisible = false.obs;

  _FakeUiObserverService();

  @override
  // ignore: avoid_returning_null
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeProfileController implements ProfileController {
  @override
  bool canCreateMoreClinicalCases() => true;

  @override
  // ignore: avoid_returning_null
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeClinicalCasesServices {
  // Completer used by the test to wait until the first stage was emitted
  final firstStageEmitted = Completer<void>();

  Future<ChatMessageModel> sendMessage(
    ChatMessageModel userMessage, {
    void Function(String token)? onStream,
  }) async {
    // Simulate SSE stage tokens being emitted while the call is in-flight
    // 1) initial stage
    onStream?.call('__STAGE__:rag_search');
    // Signal test it happened
    if (!firstStageEmitted.isCompleted) firstStageEmitted.complete();

    // 2) simulate some work
    await Future.delayed(Duration(milliseconds: 50));

    // 3) later stage
    onStream?.call('__STAGE__:streaming_answer');

    // 4) final AI message returned
    final ai = ChatMessageModel.ai(
      chatId: userMessage.chatId,
      text: 'Respuesta AI final',
    );
    return ai;
  }
}

class TestableClinicalCaseController extends ClinicalCaseController {
  TestableClinicalCaseController({
    dynamic clinicalCaseServive,
    dynamic uiObserverService,
    dynamic userService,
    dynamic profileController,
  }) : super(
         clinicalCaseServive: clinicalCaseServive,
         uiObserverService: uiObserverService,
         userService: userService,
         profileController: profileController,
       );

  void setStateValue(ClinicalCaseModel v) {
    // Allowed because this is inside an instance member of a subclass of StateMixin
    change(v, status: RxStatus.success());
  }
}
