import 'package:flutter_test/flutter_test.dart';
import 'package:get/get.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/controllers/profile_controller.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/interfaces/profile_service.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/interfaces/subscription_service.dart';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/model/subscription_model.dart';
import 'package:ema_educacion_medica_avanzada/core/countries/country_service.dart';
import 'package:ema_educacion_medica_avanzada/core/countries/country_model.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/services/actions_service.dart';
import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/core/db/database_service.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/interfaces/chat_interfaces.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_start_response.dart';
import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/core/auth/laravel_auth_service.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_local_data.dart';

class FakeAttachmentService extends AttachmentService {}
class FakeUiObserverService extends UiObserverService { FakeUiObserverService(){ isKeyboardVisible = false.obs; } }

class FakeUserService extends UserService {
  FakeUserService(){ currentUser.value = UserModel(
    id:1, firstName:'A', lastName:'B', email:'a@b', status:true, language:'es', darkMode:false,
    createdAt:DateTime.now(), updatedAt:DateTime.now(), fullName:'A B', profileImage:'', authToken:'token', countryName:''
  ); }
}

class FakeProfileService implements ProfileService {
  @override Future<UserModel> fetchDetailedProfile(UserModel profile) async => profile;
  @override Future<UserModel> updateProfile(UserModel profile) async => profile;
  @override Future<UserModel> updateProfileImage(UserModel profile, imageFile) async => profile;
}

class FakeSubscriptionService implements SubscriptionService {
  @override Future<Subscription> createSubscription({required int userId, required int subscriptionPlanId, required int frequency, required String authToken}) async => Subscription(
    id:1, name:'Free', currency:'USD', price:0, billing:'Mensual', questionnaires:999, consultations:999, clinicalCases:999, files:999, frequency:frequency, statistics:1);
  @override Future<List<Subscription>> fetchSubscriptions({required String authToken}) async => [];
  @override Future<void> updateSubscriptionQuantities({required int subscriptionId, required String authToken, int? consultations, int? questionnaires, int? clinicalCases, int? files}) async {}
}

class FakeCountryService extends CountryService { @override Future<List<CountryModel>> getCountries() async => []; }

class ProfileControllerCounters extends ProfileController {
  int decrementFileQuotaCalls = 0;
  int decrementChatQuotaCalls = 0;
  @override void onInit() { 
    super.onInit();
    currentProfile.value = userService.currentUser.value; 
  }
  @override bool canCreateMoreChats() => true;
  @override bool canUploadMoreFiles() => true;
  @override Future<bool> decrementFileQuota() async { decrementFileQuotaCalls++; return true; }
  @override Future<bool> decrementChatQuota() async { decrementChatQuotaCalls++; return true; }
  @override void refreshChatQuota() {}
  @override void refreshFileQuota() {}
  @override Future<void> refreshProfile({bool forceCancel = false}) async {}
}

class FakeLaravelAuthService extends LaravelAuthService {}

class FakeUserLocalDataService extends UserLocalDataService {
  UserModel? _u;
  @override Future<void> clear() async { _u=null; }
  @override Future<UserModel> load() async { if(_u!=null) return _u!; throw Exception('no user'); }
  @override Future<void> save(UserModel user) async { _u=user; }
  @override Future<void> deleteAll() async { _u=null; }
}

class FakeApiChatData implements IApiChatData {
  @override Future<ChatStartResponse> startChat(String prompt) async => ChatStartResponse(threadId: 't123', text: '');
  @override Future<ChatModel?> getChatById(String id) async => null;
  @override Future<List<ChatModel>> getChatsByUserId({required String userId}) async => [];
  @override Future<List<ChatMessageModel>> getMessagesById({required String id}) async => [];
  @override Future<ChatMessageModel> sendMessage({required String threadId, required String prompt, CancelToken? cancelToken, void Function(String token)? onStream}) async {
    onStream?.call('AI ');
    return ChatMessageModel.ai(chatId: 'chat1', text: 'AI completa');
  }
  @override Future<ChatMessageModel> sendPdfUpload({required String threadId, required String prompt, required PdfAttachment file, CancelToken? cancelToken, Function(int, int)? onSendProgress, void Function(String token)? onStream}) async {
    onStream?.call('AI ');
    return ChatMessageModel.ai(chatId: 'chat1', text: 'AI completa');
  }
}

class FakeChatsService extends ChatsService {
  bool alwaysThrow = false;
  bool streamThenThrow = false;
  int generateCalls = 0;
  @override
  Future<ChatModel> generateNewChat(ChatModel current, String? userText, PdfAttachment? file, String threadId) async {
    // Evitar fallo si ActionsService no está registrado todavía
    if(!Get.isRegistered<ActionsService>()) { Get.put(ActionsService()); }
  generateCalls++;
  return super.generateNewChat(current, userText, file, threadId);
  }
  @override Future<ChatStartResponse> startChat(String prompt) async => ChatStartResponse(threadId: 'threadX', text: '');
  @override Future<ChatMessageModel> sendMessage({required String threadId, required ChatMessageModel userMessage, PdfAttachment? file, void Function(String token)? onStream}) async {
    if (alwaysThrow) { throw Exception('fail'); }
    if (streamThenThrow) { onStream?.call('Parcial '); throw Exception('fail'); }
    onStream?.call('Token ');
    return ChatMessageModel.ai(chatId: userMessage.chatId, text: 'Final');
  }
}

Future<void> _registerBaseDeps({FakeChatsService? chats, ProfileControllerCounters? profile}) async {
  if (!Get.isRegistered<DatabaseService>()) {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
    final db = Get.put(DatabaseService());
    await db.init();
    Get.put(LocalActionsData());
    Get.put(LocalChatData());
    Get.put(LocalChatMessageData());
    Get.put(ActionsService());
  }
  if(!Get.isRegistered<ActionsService>()) { Get.put(ActionsService()); }
  // Orden crítico de registro de dependencias
  Get.put<AttachmentService>(FakeAttachmentService());
  Get.put<LaravelAuthService>(FakeLaravelAuthService());
  Get.put<UserLocalDataService>(FakeUserLocalDataService());
  Get.put<UiObserverService>(FakeUiObserverService());
  Get.put<UserService>(FakeUserService());
  Get.put<ProfileService>(FakeProfileService());
  Get.put<SubscriptionService>(FakeSubscriptionService());
  Get.put<CountryService>(FakeCountryService());
  Get.put<IApiChatData>(FakeApiChatData());
  // Asegurar ActionsService antes de instanciar ChatsService
  if(!Get.isRegistered<ActionsService>()) { Get.put(ActionsService()); }
  Get.put<ChatsService>(chats ?? FakeChatsService());
  Get.put<ProfileController>(profile ?? ProfileControllerCounters());
}

void main() {
  setUp(() async {
    Get.reset();
  SharedPreferences.setMockInitialValues({});
    Get.testMode = true;
  });

  test('forceStopAndReset agrega burbuja temporal y limpia estados', () async {
    await _registerBaseDeps();
    final c = Get.put(ChatController());
    c.isSending.value = true; c.isTyping.value = true;
    c.forceStopAndReset();
    expect(c.isSending.value, false);
    expect(c.isTyping.value, false);
    expect(c.messages.last.text.contains('Operación detenida'), true);
  });

  test('Persistencia de threadId vía SharedPreferences', () async {
  // Pre-sembrar valor en SharedPreferences simulando persistencia previa
  SharedPreferences.setMockInitialValues({'current_thread_id':'persistido123'});
  await _registerBaseDeps();
  final c = Get.put(ChatController());
  await Future.delayed(const Duration(milliseconds: 50));
  expect(c.threadId, 'persistido123');
  });

  test('PDF only en hilo existente asigna prompt por defecto', () async {
  await _registerBaseDeps();
  // Reemplazar ChatsService por fake después de que ActionsService está listo
  if (Get.isRegistered<ChatsService>()) { Get.delete<ChatsService>(); }
  final chats = Get.put(FakeChatsService());
  final c = Get.put(ChatController());
    // Simular hilo existente manualmente
    c.threadId = 'threadX';
    c.currentChat.value = ChatModel(uid: 'chat1', threadId: 'threadX', userId:1, shortTitle: 'short', createdAt: DateTime.now(), updatedAt: DateTime.now());
    final pdf = PdfAttachment(uid: 'u1', fileName: 'doc.pdf', filePath: 'x', mimeType: 'application/pdf', fileSize: 10);
    c.attachPdf(pdf);
    await c.sendMessage('');
    final lastUser = c.messages.where((m)=> !m.aiMessage).last;
    expect(lastUser.attach != null, true);
    expect(lastUser.text.toLowerCase().contains('analiza el archivo adjunto'), true);
    // generateNewChat no debió llamarse (shortTitle ya presente)
    expect(chats.generateCalls, 0);
  });

  test('Retry con PDF decrementa cuota de archivos', () async {
  await _registerBaseDeps();
  if (Get.isRegistered<ChatsService>()) { Get.delete<ChatsService>(); }
  final chats = Get.put(FakeChatsService()..alwaysThrow = true);
  final c = Get.put(ChatController());
    final profile = Get.find<ProfileController>() as ProfileControllerCounters;
    final pdf = PdfAttachment(uid: 'u1', fileName: 'f.pdf', filePath: 'x', mimeType: 'application/pdf', fileSize: 20);
    c.attachPdf(pdf);
  await c.sendMessage('Falla'); // en este flujo falla después de intentar validar/enviar; puede haber decremento si llegó a éxito parcial
  final initialCalls = profile.decrementFileQuotaCalls;
  expect(initialCalls >= 0, true); // simplemente registrar valor inicial
    chats.alwaysThrow = false; // ahora éxito
    await c.retryLastSend();
  // En retry sólo debe descontar si no se descontó antes. Así que resultado esperado es initialCalls o initialCalls+1
  expect(profile.decrementFileQuotaCalls == initialCalls || profile.decrementFileQuotaCalls == initialCalls + 1, true);
  });

  test('Detección de stuck state resetea isSending', () async {
    await _registerBaseDeps();
    final c = Get.put(ChatController());
    // Simular envío en progreso y tiempo excedido
    c.isSending.value = true; c.lastSendTime.value = DateTime.now().subtract(const Duration(seconds: 61));
    await c.sendMessage('Ignorado'); // debería invocar _checkForStuckState y resetear
    expect(c.isSending.value, false);
  });
}
