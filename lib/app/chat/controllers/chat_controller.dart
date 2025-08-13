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
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';

import '../../../config/config.dart';
import '../../profiles/profiles.dart';

class ChatController extends GetxService {
  ScrollController? _chatScrollController;
  FocusNode? _focusNode;

  final attachmentService = Get.find<AttachmentService>();
  final chatsService = Get.find<ChatsService>();
  final keyboardService = Get.find<UiObserverService>();
  final profileController = Get.find<ProfileController>();
  final userService = Get.find<UserService>();

  final currentChat = ChatModel.empty().obs;
  final messages = <ChatMessageModel>[].obs;
  final loading = false.obs;
  final error = ''.obs;
  final isTyping = false.obs;
  final isSending = false.obs;

  String threadId = '';
  final threadPref = StringPreference(
    key: 'current_thread_id',
    defaultValue: '',
  );

  final Rx<PdfAttachment?> pendingPdf = Rx<PdfAttachment?>(null);
  final isUploadingPdf = false.obs;
  final userScrolling = false.obs;

  final userManuallyScrolled = false.obs;
  final autoScrollEnabled = true.obs;

  // Snapshot of last failed send to enable retry
  ChatMessageModel? _lastFailedUserMessage;
  PdfAttachment? _lastFailedPdf;

  // App lifecycle tracking for recovery
  final isAppResumed = false.obs;
  final lastSendTime = DateTime.now().obs;

  @override
  void onInit() {
    super.onInit();
    threadPref.getValue().then((value) {
      threadId = value;
      if (threadId.isNotEmpty && messages.isEmpty) {
        // Posible threadId obsoleto tras reinicio: forzar nuevo hilo
        print('🧹 [ChatController] Reset de threadId obsoleto tras reinicio');
        threadId = '';
        threadPref.setValue('');
      }
    });
    keyboardService.isKeyboardVisible.listen((visible) {
      if (visible && messages.isNotEmpty && autoScrollEnabled.value) {
        Future.delayed(const Duration(milliseconds: 150), () {
          scrollToBottom();
        });
      }
    });

    // Listen for app lifecycle changes to detect when app resumes
    _setupAppLifecycleListener();
  }

  void _setupAppLifecycleListener() {
    // TODO: Implement with WidgetsBindingObserver if needed
    // For now, we'll detect stuck states in sendMessage
  }

  void _checkForStuckState() {
    // If we're "sending" for more than 60 seconds, something went wrong
    if (isSending.value || isTyping.value) {
      final timeSinceLastSend = DateTime.now().difference(lastSendTime.value);
      if (timeSinceLastSend.inSeconds > 60) {
        print('⚠️ [ChatController] Detected stuck state, resetting...');
        _resetSendingState();

        // Show retry option to user
        if (_lastFailedUserMessage != null) {
          messages.add(
            ChatMessageModel.temporal(
              '¡Ups! Parece que la respuesta se quedó pendiente. '
              'Puedes intentar nuevamente.',
              true,
            ),
          );
          scrollToBottom();
        }
      }
    }
  }

  /// Force stop current operation and reset state
  void forceStopAndReset() {
    print('🛑 [ChatController] Force stopping current operation');
    _resetSendingState();

    // Remove any loading/typing indicators from messages
    if (messages.isNotEmpty &&
        messages.last.aiMessage &&
        messages.last.text.isEmpty) {
      messages.removeLast();
    }

    // Show user they can retry
    messages.add(
      ChatMessageModel.temporal(
        'Operación detenida. Puedes intentar enviar tu mensaje nuevamente.',
        true,
      ),
    );
    scrollToBottom();
  }

  void _resetSendingState() {
    isSending.value = false;
    isTyping.value = false;
    isUploadingPdf.value = false;
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
      messages.add(
        ChatMessageModel.ai(chatId: currentChat.value.uid, text: start.text),
      );
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

  /// Retry the last failed send without duplicating the user bubble.
  Future<void> retryLastSend() async {
    if (isSending.value) return;
    final failedMsg = _lastFailedUserMessage;
    final failedPdf = _lastFailedPdf;
    if (failedMsg == null) return;

    // Remove the last error bubble if it is present
    if (messages.isNotEmpty &&
        messages.last.aiMessage &&
        (messages.last.text.contains('¡Ups!') ||
            messages.last.text.contains('Error'))) {
      messages.removeLast();
    }

    isSending.value = true;
    isTyping.value = true;
    try {
      final aiMessage = ChatMessageModel.ai(chatId: failedMsg.chatId, text: '');
      var hasFirstToken = false;
      final response = await chatsService.sendMessage(
        threadId: threadId,
        userMessage: failedMsg,
        file: failedPdf,
        onStream: (token) {
          if (!hasFirstToken) {
            hasFirstToken = true;
            aiMessage.text = token;
            messages.add(aiMessage);
          } else {
            aiMessage.text += token;
          }
          messages.refresh();
          scrollToBottom();
        },
      );
      if (!hasFirstToken) {
        aiMessage.text = response.text;
        messages.add(aiMessage);
      } else {
        aiMessage.text = response.text;
      }
      // Success: clear snapshot
      _lastFailedUserMessage = null;
      _lastFailedPdf = null;
      if (failedPdf != null) {
        final success = await profileController.decrementFileQuota();
        if (success) {
          profileController.refreshFileQuota();
        }
      }
    } catch (e) {
      messages.add(
        ChatMessageModel.temporal(
          '¡Ups! Parece que hubo un problema al procesar tu mensaje. '
          'Por favor, intenta nuevamente en unos momentos.',
          true,
        ),
      );
      scrollToBottom();
    } finally {
      isSending.value = false;
      isTyping.value = false;
    }
  }

  Future<void> sendMessage([String? userText]) async {
    final String cleanUserText = userText != null ? userText.trim() : '';
    final PdfAttachment? currentPdf = pendingPdf.value;

    if (cleanUserText.isEmpty && currentPdf == null) return;
    if (isSending.value) {
      // Check if we're stuck before rejecting
      _checkForStuckState();
      return;
    }

    print('🔄 [ChatController] Iniciando envío de mensaje');
    print(
      '📄 [ChatController] Documento adjunto: ${currentPdf != null ? "Sí" : "No"}',
    );
    print('💬 [ChatController] Texto: "$cleanUserText"');

    // Update last send time for stuck detection
    lastSendTime.value = DateTime.now();

    // Validar cuota de chats antes de proceder, pero con política "permitir primero".
    // Consideramos nuevo hilo si no hay threadId o si no hay mensajes (threadId persistido podría ser viejo).
    final isNewThread = threadId.isEmpty || messages.isEmpty;
    if (!profileController.canCreateMoreChats()) {
      print(
        '⚖️ [ChatController] canCreateMoreChats=false (pre refresh), isNewThread=$isNewThread',
      );
      // Sembrar/actualizar perfil una vez para obtener cuotas actuales
      if (profileController.currentProfile.value.id <= 0 ||
          (profileController.currentProfile.value.authToken).isEmpty) {
        final basic = userService.getProfileData();
        if (basic.id > 0 && basic.authToken.isNotEmpty) {
          profileController.currentProfile.value = basic;
        }
      }
      if (profileController.currentProfile.value.id > 0 &&
          profileController.currentProfile.value.authToken.isNotEmpty) {
        await profileController.refreshProfile();
      }
      // Re-evaluar tras refrescar; si sigue sin cuota y ES hilo nuevo, informamos pero seguimos
      if (!profileController.canCreateMoreChats() && isNewThread) {
        print(
          '🚧 [ChatController] Sin cuota para crear nuevo chat; mostrando aviso pero continuando',
        );
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
      }
      // Continuamos; el backend hará cumplir los límites si aplica
    }

    if (currentPdf != null && !profileController.canUploadMoreFiles()) {
      // Seed profile if it's not ready
      if (profileController.currentProfile.value.id <= 0 ||
          (profileController.currentProfile.value.authToken).isEmpty) {
        final basic = userService.getProfileData();
        if (basic.id > 0 && basic.authToken.isNotEmpty) {
          profileController.currentProfile.value = basic;
        }
      }
      if (profileController.currentProfile.value.id > 0 &&
          profileController.currentProfile.value.authToken.isNotEmpty) {
        await profileController.refreshProfile();
      }
      // Si sigue sin cuota, bloquear solo si no es un hilo nuevo; si es nuevo, permitir para evitar falsos negativos
      if (!profileController.canUploadMoreFiles() && !isNewThread) {
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
    }

    try {
      isSending.value = true;
      isTyping.value = true;
      print(
        '✅ [ChatController] Estados inicializados: isSending=${isSending.value}, isTyping=${isTyping.value}',
      );

      // Si el chat es nuevo, intentar iniciar el hilo con el mensaje del usuario.
      // Si falla (p.ej., 500), reintentar con prompt vacío para obtener threadId
      if (threadId.isEmpty || messages.isEmpty) {
        print('🆕 [ChatController] Iniciando nuevo chat');
        try {
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
          messages.add(userMessage);

          // Si el backend no devuelve texto inicial (caso actual), enviar el mensaje y streamear la respuesta
          if ((start.text).trim().isEmpty) {
            final aiMessage = ChatMessageModel.ai(
              chatId: currentChat.value.uid,
              text: '',
            );
            var hasFirstToken = false;

            final response = await chatsService.sendMessage(
              threadId: threadId,
              userMessage: userMessage,
              file: null,
              onStream: (token) {
                if (!hasFirstToken) {
                  hasFirstToken = true;
                  aiMessage.text = token;
                  messages.add(aiMessage);
                } else {
                  aiMessage.text += token;
                }
                messages.refresh();
                scrollToBottom();
              },
            );
            if (!hasFirstToken) {
              aiMessage.text = response.text;
              messages.add(aiMessage);
            } else {
              aiMessage.text = response.text;
            }
            chatsService.chatMessagesLocalData.insertOne(aiMessage);
            pendingPdf.value = null;
            return;
          } else {
            // Mantener comportamiento si el backend llega a devolver texto inicial
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
        } catch (e) {
          // Fallback: crear thread con prompt vacío y luego enviar el mensaje del usuario vía streaming
          print('⚠️ [ChatController] Fallback: start vacío tras error: $e');
          final start = await chatsService.startChat('');
          threadId = start.threadId;
          threadPref.setValue(threadId);

          currentChat.value = await chatsService.generateNewChat(
            currentChat.value,
            cleanUserText,
            null,
            threadId,
          );

          final userMessage = ChatMessageModel.user(
            chatId: currentChat.value.uid,
            text: cleanUserText,
          );

          // Reset scroll state to allow auto scrolling
          resetAutoScroll();

          messages.add(userMessage);
          pendingPdf.value = null;
          scrollToBottom();

          // Mostrar burbuja de AI y transmitir tokens
          final aiMessage = ChatMessageModel.ai(
            chatId: currentChat.value.uid,
            text: '',
          );
          var hasFirstToken = false;

          final response = await chatsService.sendMessage(
            threadId: threadId,
            userMessage: userMessage,
            file: null,
            onStream: (token) {
              if (!hasFirstToken) {
                hasFirstToken = true;
                aiMessage.text = token;
                messages.add(aiMessage);
              } else {
                aiMessage.text += token;
              }
              messages.refresh();
              scrollToBottom();
            },
          );
          if (!hasFirstToken) {
            aiMessage.text = response.text;
            messages.add(aiMessage);
          } else {
            aiMessage.text = response.text;
          }

          // Descontar cuota después de enviar en fallback
          final success = await profileController.decrementChatQuota();
          if (success) {
            profileController.refreshChatQuota();
          }

          return;
        }
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
        final aiMessage = ChatMessageModel.ai(
          chatId: currentChat.value.uid,
          text: '',
        );
        var hasFirstToken = false;

        // Add timeout wrapper for the streaming request
        final response = await Future.any([
          chatsService.sendMessage(
            threadId: threadId,
            userMessage: userMessage,
            file: currentPdf,
            onStream: (token) {
              // Update last activity time on each token
              lastSendTime.value = DateTime.now();
              if (!hasFirstToken) {
                hasFirstToken = true;
                aiMessage.text = token;
                messages.add(aiMessage);
              } else {
                aiMessage.text += token;
              }
              messages.refresh();
              scrollToBottom();
            },
          ),
          // Timeout after 3 minutes
          Future.delayed(
            const Duration(minutes: 3),
          ).then((_) => throw Exception('Request timeout after 3 minutes')),
        ]);

        if (!hasFirstToken) {
          aiMessage.text = response.text;
          messages.add(aiMessage);
        } else {
          aiMessage.text = response.text;
        }
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
        // Keep snapshot for retry
        _lastFailedUserMessage = userMessage;
        _lastFailedPdf = currentPdf;
        messages.add(
          ChatMessageModel.temporal(
            '¡Ups! Parece que hubo un problema al procesar tu mensaje. '
            'Por favor, intenta nuevamente en unos momentos.',
            true,
          ),
        );
        scrollToBottom();
      }
    } catch (e) {
      print('❌ [ChatController] Error general: $e');
      debugPrint('Error sending message: $e');
    } finally {
      isSending.value = false;
      isTyping.value = false;
      print(
        '🏁 [ChatController] Estados finalizados: isSending=${isSending.value}, isTyping=${isTyping.value}',
      );
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
        final duration =
            position.maxScrollExtent - position.pixels > 1000
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
