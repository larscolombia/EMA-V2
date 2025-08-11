// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:ema_educacion_medica_avanzada/core/notify/notify.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/core/preferences/string_preference.dart';

import '../../../config/config.dart';
import '../../profiles/profiles.dart';

class ChatController extends GetxService {
  ScrollController? _chatScrollController;
  FocusNode? _focusNode;

  final attachmentService = Get.find<AttachmentService>();
  final chatsService = Get.find<ChatsService>();
  final keyboardService = Get.find<UiObserverService>();
  final profileController = Get.find<ProfileController>();

  final currentChat = ChatModel.empty().obs;
  final messages = <ChatMessageModel>[].obs;
  final loading = false.obs;
  final error = ''.obs;
  final isTyping = false.obs;
  final isSending = false.obs;

  String threadId = '';
  final threadPref =
      StringPreference(key: 'current_thread_id', defaultValue: '');

  final Rx<PdfAttachment?> pendingPdf = Rx<PdfAttachment?>(null);
  final isUploadingPdf = false.obs;
  final userScrolling = false.obs;

  final userManuallyScrolled = false.obs;
  final autoScrollEnabled = true.obs;

  @override
  void onInit() {
    super.onInit();
    threadPref.getValue().then((value) => threadId = value);
    keyboardService.isKeyboardVisible.listen((visible) {
      if (visible && messages.isNotEmpty && autoScrollEnabled.value) {
        Future.delayed(const Duration(milliseconds: 150), () {
          scrollToBottom();
        });
      }
    });
  }

  void cleanChat() async {
    currentChat.value = ChatModel.empty();
    messages.value = [];
    pendingPdf.value = null;
    error.value = '';
    threadId = '';
    threadPref.setValue('');
  }

  Future<void> _loadChatById(String chatId) async {
    final chat = await chatsService.getChatById(chatId);

    if (chat != null) {
      currentChat.value = chat;
      threadId = chat.threadId;
      threadPref.setValue(threadId);
    }
  }

  Future<void> _loadMessagesByChatId(String chatId) async {
    try {
      loading.value = true;

      final chatMessages = await chatsService.getMessagesById(chatId);

      messages.value = chatMessages;

      scrollToBottom();

      loading.value = false;
    } catch (e) {
      error.value = 'Error al cargar mensajes';
      loading.value = false;
    }
  }

  Future<void> startNewChat() async {
    try {
      cleanChat();
      isTyping.value = true;
      final start = await chatsService.startChat('');
      threadId = start.threadId;
      threadPref.setValue(threadId);
      messages.add(ChatMessageModel.ai(chatId: currentChat.value.uid, text: start.text));
      scrollToBottom();
    } catch (e) {
      error.value = 'Error al iniciar conversación';
    } finally {
      isTyping.value = false;
    }
  }

  void attachPdf(PdfAttachment pdf) {
    pendingPdf.value = pdf;
  }

  Future<void> sendMessage([String? userText]) async {
    final String cleanUserText = userText != null ? userText.trim() : '';
    final PdfAttachment? currentPdf = pendingPdf.value;

    if (cleanUserText.isEmpty && currentPdf == null) return;
    if (isSending.value) return;

    print('🔄 [ChatController] Iniciando envío de mensaje');
    print('📄 [ChatController] Documento adjunto: ${currentPdf != null ? "Sí" : "No"}');
    print('💬 [ChatController] Texto: "$cleanUserText"');

    // Validar la cuota de chats y archivos antes de proceder
    if (!profileController.canCreateMoreChats()) {
      Get.snackbar(
        'Límite alcanzado',
        'Has alcanzado el límite de chats en tu plan actual. Actualiza tu plan para crear más chats.',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.orange,
        colorText: Colors.white,
        duration: const Duration(seconds: 5),
        mainButton: TextButton(
          onPressed: () => Get.toNamed(Routes.subscriptions.name),
          child: const Text(
            'Actualizar Plan',
            style: TextStyle(color: Colors.white),
          ),
        ),
      );
      return;
    }

    if (currentPdf != null && !profileController.canUploadMoreFiles()) {
      Get.snackbar(
        'Límite alcanzado',
        'Has alcanzado el límite de archivos PDF en tu plan actual. Actualiza tu plan para subir más archivos.',
        snackPosition: SnackPosition.TOP,
        backgroundColor: Colors.orange,
        colorText: Colors.white,
        duration: const Duration(seconds: 5),
        mainButton: TextButton(
          onPressed: () => Get.toNamed(Routes.subscriptions.name),
          child: const Text(
            'Actualizar Plan',
            style: TextStyle(color: Colors.white),
          ),
        ),
      );
      return;
    }

    try {
      isSending.value = true;
      isTyping.value = true;
      print('✅ [ChatController] Estados inicializados: isSending=${isSending.value}, isTyping=${isTyping.value}');

      // Si el chat es nuevo, inicializamos el hilo con el asistente usando el primer mensaje
      if (threadId.isEmpty) {
        print('🆕 [ChatController] Iniciando nuevo chat');
        final start = await chatsService.startChat(cleanUserText);
        threadId = start.threadId;
        threadPref.setValue(threadId);

        currentChat.value = await chatsService.generateNewChat(
          currentChat.value,
          cleanUserText,
          null,
          threadId,
        );

        final success = await profileController.decrementChatQuota();
        if (success) {
          profileController.refreshChatQuota();
        }

        final userMessage = ChatMessageModel.user(
          chatId: currentChat.value.uid,
          text: cleanUserText,
        );
        chatsService.chatMessagesLocalData.insertOne(userMessage);
        messages.add(userMessage);

        final aiMessage = ChatMessageModel.ai(
          chatId: currentChat.value.uid,
          text: start.text,
        );
        chatsService.chatMessagesLocalData.insertOne(aiMessage);
        messages.add(aiMessage);
        scrollToBottom();
        pendingPdf.value = null;
        return;
      }

      if (messages.length <= 1) {
        print('🔄 [ChatController] Generando nuevo chat con documento');
        currentChat.value = await chatsService.generateNewChat(
          currentChat.value,
          cleanUserText,
          currentPdf,
          threadId,
        );

        final success = await profileController.decrementChatQuota();
        if (success) {
          profileController.refreshChatQuota();
        }
      }

      if (currentPdf != null) {
        print('📄 [ChatController] Validando archivo PDF');
        isUploadingPdf.value = true;
        await attachmentService.validateFile(currentPdf);
        isUploadingPdf.value = false;
        print('✅ [ChatController] Archivo PDF validado');
      }

      final userMessage = ChatMessageModel.user(
        chatId: currentChat.value.uid,
        text: cleanUserText,
        attach: currentPdf,
      );

      // Reset scroll state to allow auto scrolling
      resetAutoScroll();

      messages.add(userMessage);
      pendingPdf.value = null;
      scrollToBottom();
      print('✅ [ChatController] Mensaje del usuario agregado');

      // Mantener isTyping.value = true durante el envío del documento
      // para que se muestre el indicador de escritura
      if (currentPdf != null) {
        // Asegurar que el indicador de escritura se mantenga visible
        isTyping.value = true;
        print('📝 [ChatController] Manteniendo isTyping=true para documento');
      }

      try {
        print('🚀 [ChatController] Enviando mensaje al servidor...');
        final aiMessage = await chatsService.sendMessage(
          threadId: threadId,
          userMessage: userMessage,
          file: currentPdf,
        );
        messages.add(aiMessage);
        scrollToBottom();
        print('✅ [ChatController] Respuesta del servidor recibida');

        // Descontar la cuota después de enviar el archivo
        if (currentPdf != null) {
          final success = await profileController.decrementFileQuota();
          if (success) {
            profileController.refreshFileQuota();
          }
        }
      } catch (error) {
        print('❌ [ChatController] Error al enviar mensaje: $error');
        messages.add(ChatMessageModel.temporal(
          '¡Ups! Parece que hubo un problema al procesar tu mensaje. '
          'Por favor, intenta nuevamente en unos momentos.',
          true,
        ));
        scrollToBottom();
      }
    } catch (e) {
      print('❌ [ChatController] Error general: $e');
      debugPrint('Error sending message: $e');
    } finally {
      isSending.value = false;
      isTyping.value = false;
      print('🏁 [ChatController] Estados finalizados: isSending=${isSending.value}, isTyping=${isTyping.value}');
    }
  }

  void scrollToBottom() {
    if (_chatScrollController == null || !_chatScrollController!.hasClients) {
      return;
    }

    WidgetsBinding.instance.addPostFrameCallback((_) {
      try {
        final position = _chatScrollController!.position;

        // Skip if we're already at bottom
        if (position.pixels == position.maxScrollExtent) return;

        // Skip if user manually scrolled and we're not forcing scroll
        if (userManuallyScrolled.value && !autoScrollEnabled.value) return;

        final target = position.maxScrollExtent;
        final duration = position.maxScrollExtent - position.pixels > 1000
            ? const Duration(milliseconds: 500)
            : const Duration(milliseconds: 200);

        _chatScrollController?.animateTo(
          target,
          duration: duration,
          curve: Curves.easeOutCubic,
        );
      } catch (e) {
        // Ignore scroll errors
      }
    });
  }

  void resetAutoScroll() {
    // Re-enable auto scrolling (typically after user sends a message)
    autoScrollEnabled.value = true;
    userManuallyScrolled.value = false;
  }

  void setFocusNode(FocusNode? focusNode) {
    _focusNode = focusNode;
  }

  void setScrollController(ScrollController? scrollController) {
    _chatScrollController = scrollController;
  }

  void focusOnChatInputText() {
    if (_focusNode != null) {
      try {
        _focusNode!.requestFocus();
      } catch (e) {
        Logger.error(e.toString());
      }
    }
  }

  void showChat(String chatId) async {
    try {
      loading.value = true;

      messages.clear();

      Get.back(closeOverlays: true);

      WidgetsBinding.instance.addPostFrameCallback((_) {
        _loadChatById(chatId);
        _loadMessagesByChatId(chatId);
      });
    } catch (e) {
      Notify.snackbar('Chats', 'No se encontró el chat.');
      loading.value = false;
    }
  }
}
