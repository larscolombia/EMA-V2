import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';


enum StateMessageType {
  error,
  download,
  noConection,
  noDocuments,
  noMessages,
  noSearchResults;

  String get fileName {
    switch (this) {
      case StateMessageType.error:
        return 'error.png';
      case StateMessageType.download:
        return 'download.png';
      case StateMessageType.noConection:
        return 'no_connection.png';
      case StateMessageType.noDocuments:
        return 'no_documents.png';
      case StateMessageType.noMessages:
        return 'no_messages.png';
      case StateMessageType.noSearchResults:
        return 'no_search_result.png';
    }
  }
}


class StateMessageWidget extends StatelessWidget {
  final String message;
  final StateMessageType type;
  final VoidCallback? onRetry;
  final Widget? child;
  final bool showHomeButton;
  final bool showLoading;

  const StateMessageWidget({
    super.key,
    this.message = 'Upps, ocurrió un error desconocido',
    this.type = StateMessageType.error,
    this.onRetry,
    this.child,
    this.showHomeButton = false,
    this.showLoading = false,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Image.asset('assets/images/${type.fileName}'),

          SizedBox(height: 16),
          
          Text(message),

          if (showLoading)
          SizedBox(height: 16),

          if (showLoading)
          SizedBox(
            width: 150,
            child: LinearProgressIndicator(
              valueColor: AlwaysStoppedAnimation<Color>(AppStyles.tertiaryColor),
              backgroundColor: AppStyles.grey220,
              borderRadius: BorderRadius.circular(8),
              minHeight: 6,
            ),
          ),

          if (child != null)
          child!,

          if (onRetry != null)
          ElevatedButton(
            onPressed: onRetry,
            child: const Text('Reintentar'),
          ),

          SizedBox(height: 16),

          if (showHomeButton)
          ElevatedButton(
            onPressed: () => Get.offAllNamed(Routes.home.name),
            child: const Text('Volver al inicio'),
          ),
        ],
      ),
    );
  }
}
