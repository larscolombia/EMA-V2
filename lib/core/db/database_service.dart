import 'package:ema_educacion_medica_avanzada/app/actions/data/local_actions_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_data.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/data/local_chat_message_data.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/data/local_clinical_case_data.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';
import 'package:sqflite/sqflite.dart';

class DatabaseService extends GetxService {
  late final Database db;

  Future<void> _createTables(Database db, int version) async {
    await db.execute(LocalActionsData.sqlInstructionCreateTable());
    await db.execute(LocalChatData.sqlInstructionCreateTable());
    await db.execute(LocalChatMessageData.sqlInstructionCreateTable());
    await db.execute(LocalClinicalCaseData.sqlInstructionCreateTable());
    await db.execute(LocalQuestionsData.sqlInstructionCreateTable());
    await db.execute(LocalQuizzData.sqlInstructionCreateTable());
  }

  Future<void> _onUpgrade(Database db, int oldVersion, int newVersion) async {
    Logger.log('DatabaseService.onUpgrade($oldVersion -> $newVersion)');

    // Migración para agregar columna imageAttach si no existe (v20 -> v21)
    if (oldVersion < 21) {
      try {
        // Verificar si la columna ya existe
        final result = await db.rawQuery('PRAGMA table_info(chat_messages_v1)');
        final hasImageAttach = result.any(
          (column) => column['name'] == 'imageAttach',
        );

        if (!hasImageAttach) {
          Logger.log('🔄 Agregando columna imageAttach a chat_messages_v1');
          await db.execute(
            'ALTER TABLE chat_messages_v1 ADD COLUMN imageAttach TEXT',
          );
          Logger.log('✅ Columna imageAttach agregada correctamente');
        } else {
          Logger.log('ℹ️ Columna imageAttach ya existe, omitiendo migración');
        }
      } catch (e) {
        Logger.error('❌ Error en migración de imageAttach: $e');
        // No fallar la app por esto, continuar
      }
    }

    // Migración para agregar columna format si no existe (v21 -> v22)
    if (oldVersion < 22) {
      try {
        // Verificar si la columna ya existe
        final result = await db.rawQuery('PRAGMA table_info(chat_messages_v1)');
        final hasFormat = result.any((column) => column['name'] == 'format');

        if (!hasFormat) {
          Logger.log('🔄 Agregando columna format a chat_messages_v1');
          await db.execute(
            "ALTER TABLE chat_messages_v1 ADD COLUMN format TEXT DEFAULT 'plain'",
          );
          Logger.log('✅ Columna format agregada correctamente');
        } else {
          Logger.log('ℹ️ Columna format ya existe, omitiendo migración');
        }
      } catch (e) {
        Logger.error('❌ Error en migración de format: $e');
        // No fallar la app por esto, continuar
      }
    }
  }

  Future<void> init() async {
    Logger.log('DatabaseService.init()');

    db = await openDatabase(
      'ema_db_v1_19.db',
      version: 22, // Incrementado para forzar migración de format
      onCreate: _createTables,
      onUpgrade: _onUpgrade,
    );
  }
}
