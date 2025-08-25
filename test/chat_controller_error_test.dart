import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/controllers/profile_controller.dart';
import 'package:ema_educacion_medica_avanzada/core/auth/laravel_auth_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_local_data.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/interfaces/profile_service.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/interfaces/subscription_service.dart';
import 'package:ema_educacion_medica_avanzada/core/countries/country_service.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/model/subscription_model.dart';
import 'package:image_picker/image_picker.dart';
import 'package:ema_educacion_medica_avanzada/core/countries/country_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/core/db/database_service.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:dio/dio.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:sqflite/sqflite.dart';

// Fakes mínimos ajustados a la firma real
class FakeAttachmentService extends AttachmentService {}
class FakeUiObserverService extends UiObserverService { FakeUiObserverService(){ isKeyboardVisible = false.obs; } }

class FakeUserService extends UserService {
  FakeUserService(){ currentUser.value = UserModel(
    id:1, firstName:'A', lastName:'B', email:'a@b', status:true, language:'es', darkMode:false,
    createdAt:DateTime.now(), updatedAt:DateTime.now(), fullName:'A B', profileImage:'', authToken:'token', countryName:''
  ); }
}

// Stubs para dependencias de UserService
class FakeLaravelAuthService extends LaravelAuthService {}

class FakeUserLocalDataService extends UserLocalDataService {
  UserModel? _mem;
  @override Future<void> clear() async { _mem = null; }
  @override Future<UserModel> load() async { if(_mem!=null) return _mem!; throw Exception('No user'); }
  @override Future<void> save(UserModel user) async { _mem = user; }
  @override Future<void> deleteAll() async { _mem = null; }
}

class FakeProfileService implements ProfileService {
  @override Future<UserModel> fetchDetailedProfile(UserModel profile) async => profile;
  @override Future<UserModel> updateProfile(UserModel profile) async => profile;
  @override Future<UserModel> updateProfileImage(UserModel profile, XFile imageFile) async => profile;
}

class FakeSubscriptionService implements SubscriptionService {
  @override Future<Subscription> createSubscription({required int userId, required int subscriptionPlanId, required int frequency, required String authToken}) async => Subscription(
    id:1, name:'Free', currency:'USD', price:0, billing:'Mensual', questionnaires:999, consultations:999, clinicalCases:999, files:999, frequency:frequency, statistics:1);
  @override Future<List<Subscription>> fetchSubscriptions({required String authToken}) async => [];
  @override Future<void> updateSubscriptionQuantities({required int subscriptionId, required String authToken, int? consultations, int? questionnaires, int? clinicalCases, int? files}) async {}
}

class FakeCountryService extends CountryService {
  @override Future<List<CountryModel>> getCountries() async => [];
}

class FakeProfileController extends ProfileController {
  FakeProfileController(){ currentProfile.value = userService.currentUser.value; }
  @override void onInit() { super.onInit(); } // mantiene contrato mustCallSuper
  @override bool canCreateMoreChats() => true;
  @override bool canUploadMoreFiles() => true;
  @override Future<bool> decrementChatQuota() async => false; // server-side now
  @override Future<bool> decrementFileQuota() async => false; // server-side now
  @override void refreshChatQuota() {}
  @override void refreshFileQuota() {}
  @override Future<void> refreshProfile({bool forceCancel = false}) async {}
}

class FakeChatsService extends ChatsService {
  bool throwAfterFirstToken = false;
  bool alwaysThrow = false;

  @override
  Future<ChatMessageModel> sendMessage({required String threadId, required ChatMessageModel userMessage, PdfAttachment? file, void Function(String token)? onStream}) async {
    if (throwAfterFirstToken) {
      onStream?.call('Parcial ');
      throw Exception('DioException 504');
    }
    if (alwaysThrow) {
      throw Exception('DioException 504');
    }
    onStream?.call('Respuesta ');
    return ChatMessageModel.ai(chatId: userMessage.chatId, text: 'Respuesta completa');
  }
}

class FakeApiChatData implements IApiChatData {
  @override Future<ChatStartResponse> startChat(String prompt) async => ChatStartResponse(threadId: 't123', text: '');
  @override Future<ChatModel?> getChatById(String id) async => null;
  @override Future<List<ChatModel>> getChatsByUserId({required String userId}) async => [];
  @override Future<List<ChatMessageModel>> getMessagesById({required String id}) async => [];
  @override Future<ChatMessageModel> sendMessage({required String threadId, required String prompt, CancelToken? cancelToken, void Function(String token)? onStream}) async {
    onStream?.call('Respuesta ');
    return ChatMessageModel.ai(chatId: 'chat1', text: 'Respuesta completa');
  }
  @override Future<ChatMessageModel> sendPdfUpload({required String threadId, required String prompt, required PdfAttachment file, CancelToken? cancelToken, Function(int, int)? onSendProgress, void Function(String token)? onStream}) async {
    onStream?.call('Respuesta ');
    return ChatMessageModel.ai(chatId: 'chat1', text: 'Respuesta completa');
  }
}

void main(){
  setUp(() async {
    Get.reset();
    SharedPreferences.setMockInitialValues({});
  });

  Future<ChatController> _build({bool throwAfterFirstToken=false, bool alwaysThrow=false}) async {
    // Inicializar DB FFI solo una vez
    if (Get.isRegistered<DatabaseService>() == false) {
      sqfliteFfiInit();
      databaseFactory = databaseFactoryFfi;
      final dbService = Get.put(DatabaseService());
      // ignore: invalid_use_of_visible_for_testing_member
  await dbService.init();
      Get.put(LocalActionsData());
      Get.put(LocalChatData());
      Get.put(LocalChatMessageData());
      Get.put(ActionsService());
      Get.put<IApiChatData>(FakeApiChatData());
    }
    Get.put<AttachmentService>(FakeAttachmentService());
    Get.put<UiObserverService>(FakeUiObserverService());
  Get.put<LaravelAuthService>(FakeLaravelAuthService());
  Get.put<UserLocalDataService>(FakeUserLocalDataService());
    Get.put<UserService>(FakeUserService());
  Get.put<ProfileService>(FakeProfileService());
  Get.put<SubscriptionService>(FakeSubscriptionService());
  Get.put<CountryService>(FakeCountryService());
    Get.put<ProfileController>(FakeProfileController());
    final chats = FakeChatsService()
      ..throwAfterFirstToken=throwAfterFirstToken
      ..alwaysThrow=alwaysThrow;
    Get.put<ChatsService>(chats);
    final controller = Get.put(ChatController());
    controller.threadId = 'threadX';
    controller.currentChat.value = ChatModel(
      uid: 'chat1', threadId: 'threadX', userId:1, shortTitle:'stub', createdAt: DateTime.now(), updatedAt: DateTime.now());
    controller.messages.add(ChatMessageModel.user(chatId: 'chat1', text: 'inicio'));
  return controller;
  }

  test('504 sin tokens: estados reseteados y burbuja temporal', () async {
    final c = await _build(alwaysThrow:true);
    await c.sendMessage('hola');
    expect(c.isSending.value, false);
    expect(c.isTyping.value, false);
    final last = c.messages.last;
    expect(last.aiMessage, true);
    expect(last.text.contains('Ups'), true);
  });

  test('504 tras token parcial: conserva parcial + burbuja error', () async {
    final c = await _build(throwAfterFirstToken:true);
    await c.sendMessage('hola');
    final aiMessages = c.messages.where((m)=>m.aiMessage).toList();
    expect(aiMessages.length >= 2, true);
    expect(aiMessages.first.text.startsWith('Parcial'), true);
    expect(aiMessages.last.text.contains('Ups'), true);
    expect(c.isSending.value, false);
    expect(c.isTyping.value, false);
  });

  test('Retry después de error 504 genera respuesta completa', () async {
    final c = await _build(alwaysThrow:true);
    await c.sendMessage('hola');
    // Cambiar a éxito
    final service = Get.find<ChatsService>() as FakeChatsService;
    service.alwaysThrow = false;
    await c.retryLastSend();
    expect(c.messages.any((m)=> m.text.contains('Respuesta completa')), true);
  });
}
